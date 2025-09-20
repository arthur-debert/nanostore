package ids

import (
	"errors"
	"fmt"
	"reflect"
)

// IDResolver defines the interface for resolving SimpleIDs to UUIDs
type IDResolver interface {
	ResolveID(simpleID string) (string, error)
}

// FieldInfo provides information about dimension fields
type FieldInfo interface {
	IsReferenceField(fieldName string) bool
}

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

// CommandPreprocessor handles centralized preprocessing of commands including ID resolution
type CommandPreprocessor struct {
	resolver  IDResolver
	fieldInfo FieldInfo
}

// NewCommandPreprocessor creates a new command preprocessor
func NewCommandPreprocessor(resolver IDResolver, fieldInfo FieldInfo) *CommandPreprocessor {
	return &CommandPreprocessor{
		resolver:  resolver,
		fieldInfo: fieldInfo,
	}
}

// PreprocessCommand processes any command, resolving IDs and performing validation
func (cp *CommandPreprocessor) PreprocessCommand(cmd interface{}) error {
	// Use reflection to find and resolve ID fields
	return cp.resolveIDsInStruct(cmd)
}

// resolveIDsInStruct recursively resolves SimpleIDs to UUIDs in a struct
func (cp *CommandPreprocessor) resolveIDsInStruct(v interface{}) error {
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
func (cp *CommandPreprocessor) isIDField(field reflect.StructField) bool {
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
func (cp *CommandPreprocessor) resolveIDField(field reflect.Value) error {
	if !field.CanSet() || field.Kind() != reflect.String {
		return nil
	}

	id := field.String()
	if id == "" || IsValidUUID(id) {
		return nil // Empty or already a UUID
	}

	// Resolve SimpleID to UUID
	uuid, err := cp.resolver.ResolveID(id)
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
func (cp *CommandPreprocessor) resolveIDsInMap(mapVal reflect.Value) error {
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
		if cp.fieldInfo.IsReferenceField(keyStr) {
			value := mapVal.MapIndex(key)
			if value.Kind() == reflect.Interface {
				value = value.Elem()
			}
			if value.Kind() == reflect.String {
				id := value.String()
				if id != "" && !IsValidUUID(id) {
					// Try to resolve
					uuid, err := cp.resolver.ResolveID(id)
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
