package nanostore

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/arthur-debert/nanostore/nanostore/ids"
)

// IDResolutionError indicates that a SimpleID could not be resolved to a UUID
type IDResolutionError struct {
	ID           string
	WrappedError error
}

// Error implements the error interface
func (e *IDResolutionError) Error() string {
	return fmt.Sprintf("failed to resolve ID %q: %v", e.ID, e.WrappedError)
}

// Unwrap allows error unwrapping
func (e *IDResolutionError) Unwrap() error {
	return e.WrappedError
}

// commandPreprocessor handles centralized preprocessing of commands including ID resolution.
//
// This system provides a unified approach to preparing commands before execution,
// with a primary focus on resolving SimpleIDs to UUIDs throughout nested data structures.
// The preprocessor uses reflection to traverse arbitrary command structures and locate
// ID fields that require resolution.
//
// # Core Responsibilities
//
// 1. **ID Resolution**: Convert SimpleIDs to UUIDs in command structures
// 2. **Nested Traversal**: Handle complex nested structs, slices, pointers, and maps
// 3. **Error Handling**: Distinguish between resolution failures and other errors
// 4. **Reference Fields**: Process hierarchical dimension references (parent_id, etc.)
//
// # Architecture Benefits
//
// The centralized preprocessing approach solves several architectural problems:
//
// - **Eliminates Code Duplication**: No need for scattered ID resolution logic
// - **Prevents Omissions**: Automatic discovery ensures all ID fields are processed
// - **Consistent Error Handling**: Unified approach to resolution failures
// - **Future-Proof**: New command types automatically benefit from preprocessing
// - **Testable**: Complex resolution logic is isolated and thoroughly tested
//
// # Reflection-Based Field Discovery
//
// The preprocessor uses Go's reflection system to automatically discover ID fields:
//
// 1. **Field Name Matching**: Fields named "ID", "ParentID", or "UUID"
// 2. **Tag-Based Discovery**: Fields with `id:"true"` struct tags
// 3. **Reference Field Detection**: Hierarchical dimension reference fields
// 4. **Nested Structure Traversal**: Recursive processing of embedded structs
//
// Example field discovery:
//
//	type UpdateCommand struct {
//	    ID      string `id:"true"`     // Discovered by tag
//	    ParentID string               // Discovered by name
//	    Data    NestedStruct          // Recursively processed
//	    Items   []SubCommand          // Slice elements processed
//	}
//
// # Error Handling Strategy
//
// The preprocessor distinguishes between different types of errors:
//
// - **IDResolutionError**: SimpleID could not be resolved (non-fatal)
//   - Allows external references or IDs that don't exist yet
//   - Original value is preserved for later validation
//   - Calling method decides if this should be fatal
//
// - **Other Errors**: Structural problems, invalid data types (fatal)
//   - Reflection errors, type conversion failures
//   - These indicate bugs or invalid command structures
//
// This approach supports scenarios like:
//   - Bulk imports where parent documents might not exist yet
//   - External system integrations with foreign IDs
//   - Partial command validation during development
//
// # Supported Data Structures
//
// The preprocessor handles a wide variety of Go data structures:
//
// - **Simple Fields**: string fields with ID values
// - **Pointer Fields**: *string fields that may be nil
// - **Nested Structs**: Embedded structures with their own ID fields
// - **Slices**: []Struct with ID fields in each element
// - **Maps**: map[string]interface{} with special handling for reference fields
// - **Mixed Structures**: Complex combinations of the above
//
// # Integration with Store Operations
//
// The preprocessor integrates seamlessly with store operations:
//
//	// Before preprocessing
//	cmd := &UpdateCommand{
//	    ID: "1.dh3",                    // SimpleID
//	    Request: UpdateRequest{
//	        Dimensions: map[string]interface{}{
//	            "parent_id": "1.2",     // SimpleID in reference field
//	        },
//	    },
//	}
//
//	// After preprocessing
//	cmd := &UpdateCommand{
//	    ID: "550e8400-e29b-41d4-a716-446655440000",  // Resolved UUID
//	    Request: UpdateRequest{
//	        Dimensions: map[string]interface{}{
//	            "parent_id": "550e8400-e29b-41d4-a716-446655440001", // Resolved UUID
//	        },
//	    },
//	}
//
// # Performance Characteristics
//
// - **Time Complexity**: O(n) where n = number of fields in the command structure
// - **Space Complexity**: O(1) additional memory (in-place modification)
// - **Reflection Overhead**: Minimal - only used for structure traversal
// - **Resolution Caching**: Store-level caching minimizes UUID lookup overhead
//
// # Thread Safety
//
// The commandPreprocessor is thread-safe:
//   - Immutable after creation (only stores reference to store)
//   - No shared mutable state between preprocessing calls
//   - Store operations are protected by the centralized lock manager
type commandPreprocessor struct {
	store *jsonFileStore
}

// newCommandPreprocessor creates a new command preprocessor
func newCommandPreprocessor(store *jsonFileStore) *commandPreprocessor {
	return &commandPreprocessor{store: store}
}

// preprocessCommand processes any command, resolving IDs and performing validation
func (cp *commandPreprocessor) preprocessCommand(cmd interface{}) error {
	// Use reflection to find and resolve ID fields
	return cp.resolveIDsInStruct(cmd)
}

// resolveIDsInStruct recursively resolves SimpleIDs to UUIDs in a struct
func (cp *commandPreprocessor) resolveIDsInStruct(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Check for ID fields by name or tag
		if cp.isIDField(fieldType) && field.Kind() == reflect.String {
			if err := cp.resolveIDField(field); err != nil {
				// Check if it's an IDResolutionError
				var resErr *IDResolutionError
				if errors.As(err, &resErr) {
					// For ID fields, we treat resolution failures as non-fatal
					// to support external references or IDs that don't exist yet
					// The calling method will validate if the ID must exist
					continue
				}
				return fmt.Errorf("failed to resolve ID in field %s: %w", fieldType.Name, err)
			}
		} else if field.Kind() == reflect.Map {
			// Handle maps (like dimensions)
			if err := cp.resolveIDsInMap(field); err != nil {
				return fmt.Errorf("failed to resolve IDs in map field %s: %w", fieldType.Name, err)
			}
		} else if field.Kind() == reflect.Struct {
			// Recursively process nested structs
			if err := cp.resolveIDsInStruct(field.Addr().Interface()); err != nil {
				return err
			}
		} else if field.Kind() == reflect.Ptr && !field.IsNil() {
			// Handle pointer fields
			if field.Elem().Kind() == reflect.Struct {
				// Recursively process pointed-to struct
				if err := cp.resolveIDsInStruct(field.Interface()); err != nil {
					return err
				}
			} else if field.Elem().Kind() == reflect.String && cp.isIDField(fieldType) {
				// Handle *string ID fields
				if err := cp.resolveIDField(field.Elem()); err != nil {
					var resErr *IDResolutionError
					if errors.As(err, &resErr) {
						continue
					}
					return fmt.Errorf("failed to resolve pointer ID in field %s: %w", fieldType.Name, err)
				}
			}
		} else if field.Kind() == reflect.Slice {
			// Handle slice fields
			for i := 0; i < field.Len(); i++ {
				elem := field.Index(i)
				if elem.Kind() == reflect.Struct {
					if err := cp.resolveIDsInStruct(elem.Addr().Interface()); err != nil {
						return err
					}
				} else if elem.Kind() == reflect.Ptr && !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
					if err := cp.resolveIDsInStruct(elem.Interface()); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// isIDField checks if a field should have ID resolution
func (cp *commandPreprocessor) isIDField(field reflect.StructField) bool {
	// Check field name
	name := field.Name
	if name == "ID" || name == "ParentID" || name == "UUID" {
		return true
	}

	// Check for id tag
	if tag := field.Tag.Get("id"); tag == "true" {
		return true
	}

	return false
}

// resolveIDField resolves a single ID field
func (cp *commandPreprocessor) resolveIDField(field reflect.Value) error {
	if !field.CanSet() || field.Kind() != reflect.String {
		return nil
	}

	id := field.String()
	if id == "" || ids.IsValidUUID(id) {
		return nil // Empty or already a UUID
	}

	// Resolve SimpleID to UUID
	uuid, err := cp.store.resolveUUIDInternal(id)
	if err != nil {
		// Return a wrapped error that allows the caller to decide
		// whether to treat this as fatal or continue with the original value
		return &IDResolutionError{
			ID:           id,
			WrappedError: err,
		}
	}

	field.SetString(uuid)
	return nil
}

// resolveIDsInMap handles ID resolution in maps (like dimension maps)
func (cp *commandPreprocessor) resolveIDsInMap(mapVal reflect.Value) error {
	if mapVal.Kind() != reflect.Map {
		return nil
	}

	// Check for parent_id or other ID fields in dimensions
	for _, key := range mapVal.MapKeys() {
		if key.Kind() != reflect.String {
			continue
		}

		keyStr := key.String()
		// Check if this is a hierarchical ref field
		if cp.isRefField(keyStr) {
			value := mapVal.MapIndex(key)
			if value.Kind() == reflect.Interface {
				value = value.Elem()
			}
			if value.Kind() == reflect.String {
				id := value.String()
				if id != "" && !ids.IsValidUUID(id) {
					// Try to resolve
					uuid, err := cp.store.resolveUUIDInternal(id)
					if err != nil {
						// For reference fields in maps (like parent_id in dimensions),
						// we don't treat resolution failures as fatal since the parent
						// might not exist yet (e.g., bulk import scenarios)
						// The store will validate referential integrity if needed
						continue
					}
					// Set the resolved UUID back
					mapVal.SetMapIndex(key, reflect.ValueOf(uuid))
				}
			}
		}
	}

	return nil
}

// isRefField checks if a field name is a hierarchical reference field
func (cp *commandPreprocessor) isRefField(fieldName string) bool {
	// Check all hierarchical dimensions for ref fields
	for _, dim := range cp.store.dimensionSet.Hierarchical() {
		if dim.RefField == fieldName {
			return true
		}
	}
	return false
}

// Command types for preprocessing

// UpdateCommand represents an update operation
type UpdateCommand struct {
	ID      string `id:"true"`
	Request UpdateRequest
}

// DeleteCommand represents a delete operation
type DeleteCommand struct {
	ID      string `id:"true"`
	Cascade bool
}

// AddCommand represents an add operation
type AddCommand struct {
	Title      string
	Dimensions map[string]interface{}
}
