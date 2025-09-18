package nanostore

import (
	"fmt"
	"reflect"
)

// commandPreprocessor handles centralized preprocessing of commands including ID resolution
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
	if id == "" || isValidUUID(id) {
		return nil // Empty or already a UUID
	}

	// Resolve SimpleID to UUID
	uuid, err := cp.store.resolveUUIDInternal(id)
	if err != nil {
		// If resolution fails, keep the original value
		// This allows for new documents or external references
		return nil
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
				if id != "" && !isValidUUID(id) {
					// Try to resolve
					if uuid, err := cp.store.resolveUUIDInternal(id); err == nil {
						// Set the resolved UUID back
						mapVal.SetMapIndex(key, reflect.ValueOf(uuid))
					}
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
