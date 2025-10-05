package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TypeSchema represents a JSON schema for defining document types
type TypeSchema struct {
	Name       string                     `json:"name"`
	Package    string                     `json:"package,omitempty"`
	Dimensions map[string]DimensionSchema `json:"dimensions"`
	Fields     map[string]FieldSchema     `json:"fields"`
}

// DimensionSchema defines a dimension configuration
type DimensionSchema struct {
	Values   []string          `json:"values,omitempty"`
	Default  string            `json:"default,omitempty"`
	Prefixes map[string]string `json:"prefixes,omitempty"`
	Type     string            `json:"type,omitempty"`      // "enumerated" or "hierarchical"
	RefField string            `json:"ref_field,omitempty"` // For hierarchical dimensions
}

// FieldSchema defines a data field
type FieldSchema struct {
	Type        string `json:"type"` // Go type string (e.g., "string", "*time.Time")
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// TypeDefinition represents a runtime type definition
type TypeDefinition struct {
	Name       string
	Schema     TypeSchema
	GoType     reflect.Type
	StoreType  reflect.Type                      // The Store[T] type
	CreateFunc func(string) (interface{}, error) // Function to create store instance
}

// Enhanced TypeRegistry with JSON schema support
type EnhancedTypeRegistry struct {
	types       map[string]*TypeDefinition
	schemaCache map[string]TypeSchema
}

// NewEnhancedTypeRegistry creates a new enhanced type registry
func NewEnhancedTypeRegistry() *EnhancedTypeRegistry {
	return &EnhancedTypeRegistry{
		types:       make(map[string]*TypeDefinition),
		schemaCache: make(map[string]TypeSchema),
	}
}

// LoadTypeFromJSON loads a type definition from a JSON schema file
func (etr *EnhancedTypeRegistry) LoadTypeFromJSON(schemaPath string) error {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file %s: %w", schemaPath, err)
	}

	var schema TypeSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("failed to parse schema file %s: %w", schemaPath, err)
	}

	// If name is not set, use filename without extension
	if schema.Name == "" {
		schema.Name = strings.TrimSuffix(filepath.Base(schemaPath), filepath.Ext(schemaPath))
	}

	return etr.RegisterTypeFromSchema(schema)
}

// RegisterTypeFromSchema registers a type from a schema definition
func (etr *EnhancedTypeRegistry) RegisterTypeFromSchema(schema TypeSchema) error {
	// Generate a Go struct type dynamically
	goType, err := etr.generateGoTypeFromSchema(schema)
	if err != nil {
		return fmt.Errorf("failed to generate Go type for %s: %w", schema.Name, err)
	}

	// Create store type (Store[T])
	storeType := reflect.TypeOf((*api.Store[struct{}])(nil)).Elem()

	// Create function to instantiate store
	createFunc := func(dbPath string) (interface{}, error) {
		// This is a simplified version - in practice we'd need to use reflection
		// to call api.New[T](dbPath) with the dynamic type T
		return nil, fmt.Errorf("dynamic store creation not yet implemented for type %s", schema.Name)
	}

	definition := &TypeDefinition{
		Name:       schema.Name,
		Schema:     schema,
		GoType:     goType,
		StoreType:  storeType,
		CreateFunc: createFunc,
	}

	etr.types[schema.Name] = definition
	etr.schemaCache[schema.Name] = schema

	return nil
}

// GetTypeDefinition retrieves a type definition by name
func (etr *EnhancedTypeRegistry) GetTypeDefinition(name string) (*TypeDefinition, bool) {
	def, exists := etr.types[name]
	return def, exists
}

// ListTypes returns all registered type names
func (etr *EnhancedTypeRegistry) ListTypes() []string {
	var names []string
	for name := range etr.types {
		names = append(names, name)
	}
	return names
}

// generateGoTypeFromSchema creates a reflect.Type from a schema
// This is a simplified implementation - a full version would need more sophisticated
// dynamic type generation, possibly using code generation or runtime struct creation
func (etr *EnhancedTypeRegistry) generateGoTypeFromSchema(schema TypeSchema) (reflect.Type, error) {
	// For now, return a basic struct type
	// In a full implementation, this would dynamically create struct fields
	// based on the schema dimensions and fields

	// Create a basic struct that embeds Document
	fields := []reflect.StructField{
		{
			Name:      "Document",
			Type:      reflect.TypeOf(nanostore.Document{}),
			Anonymous: true,
		},
	}

	// Add dimension fields
	caser := cases.Title(language.Und)
	for dimName, dimSchema := range schema.Dimensions {
		fieldName := caser.String(dimName)
		fieldType := reflect.TypeOf("")

		var tag string
		if dimSchema.Type == "enumerated" || len(dimSchema.Values) > 0 {
			tag = fmt.Sprintf(`values:"%s"`, strings.Join(dimSchema.Values, ","))
			if dimSchema.Default != "" {
				tag += fmt.Sprintf(` default:"%s"`, dimSchema.Default)
			}
		} else if dimSchema.Type == "hierarchical" {
			tag = fmt.Sprintf(`dimension:"%s,ref"`, dimName)
		}

		fields = append(fields, reflect.StructField{
			Name: fieldName,
			Type: fieldType,
			Tag:  reflect.StructTag(tag),
		})
	}

	// Add data fields
	for fieldName, fieldSchema := range schema.Fields {
		goFieldName := caser.String(fieldName)
		fieldType, err := etr.parseGoType(fieldSchema.Type)
		if err != nil {
			return nil, fmt.Errorf("invalid field type %s for field %s: %w", fieldSchema.Type, fieldName, err)
		}

		fields = append(fields, reflect.StructField{
			Name: goFieldName,
			Type: fieldType,
		})
	}

	// Create the struct type
	structType := reflect.StructOf(fields)
	return structType, nil
}

// parseGoType converts a string type representation to reflect.Type
func (etr *EnhancedTypeRegistry) parseGoType(typeStr string) (reflect.Type, error) {
	switch typeStr {
	case "string":
		return reflect.TypeOf(""), nil
	case "*string":
		return reflect.TypeOf((*string)(nil)), nil
	case "int":
		return reflect.TypeOf(0), nil
	case "*int":
		return reflect.TypeOf((*int)(nil)), nil
	case "int64":
		return reflect.TypeOf(int64(0)), nil
	case "*int64":
		return reflect.TypeOf((*int64)(nil)), nil
	case "float64":
		return reflect.TypeOf(float64(0)), nil
	case "*float64":
		return reflect.TypeOf((*float64)(nil)), nil
	case "bool":
		return reflect.TypeOf(false), nil
	case "*bool":
		return reflect.TypeOf((*bool)(nil)), nil
	case "time.Time":
		return reflect.TypeOf(time.Time{}), nil
	case "*time.Time":
		return reflect.TypeOf((*time.Time)(nil)), nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", typeStr)
	}
}

// LoadBuiltinTypes registers some common built-in type schemas
func (etr *EnhancedTypeRegistry) LoadBuiltinTypes() error {
	// Task type schema
	taskSchema := TypeSchema{
		Name: "Task",
		Dimensions: map[string]DimensionSchema{
			"status": {
				Values:  []string{"pending", "active", "done"},
				Default: "pending",
				Prefixes: map[string]string{
					"done": "d",
				},
				Type: "enumerated",
			},
			"priority": {
				Values:  []string{"low", "medium", "high"},
				Default: "medium",
				Prefixes: map[string]string{
					"high": "h",
				},
				Type: "enumerated",
			},
			"parent_id": {
				Type:     "hierarchical",
				RefField: "parent_id",
			},
		},
		Fields: map[string]FieldSchema{
			"description": {Type: "string", Description: "Task description"},
			"assignee":    {Type: "string", Description: "Person assigned to task"},
			"due_date":    {Type: "*time.Time", Description: "Due date"},
		},
	}

	if err := etr.RegisterTypeFromSchema(taskSchema); err != nil {
		return fmt.Errorf("failed to register Task type: %w", err)
	}

	// Note type schema
	noteSchema := TypeSchema{
		Name: "Note",
		Dimensions: map[string]DimensionSchema{
			"category": {
				Values:  []string{"personal", "work", "idea", "reference"},
				Default: "personal",
				Type:    "enumerated",
			},
		},
		Fields: map[string]FieldSchema{
			"tags":    {Type: "string", Description: "Comma-separated tags"},
			"content": {Type: "string", Description: "Note content"},
		},
	}

	if err := etr.RegisterTypeFromSchema(noteSchema); err != nil {
		return fmt.Errorf("failed to register Note type: %w", err)
	}

	return nil
}

// GetSchemaJSON returns the JSON representation of a type schema
func (etr *EnhancedTypeRegistry) GetSchemaJSON(typeName string) (string, error) {
	schema, exists := etr.schemaCache[typeName]
	if !exists {
		return "", fmt.Errorf("type %s not found", typeName)
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	return string(data), nil
}

// CreateStoreInstance creates a store instance for the given type
func (etr *EnhancedTypeRegistry) CreateStoreInstance(typeName, dbPath string) (interface{}, error) {
	definition, exists := etr.GetTypeDefinition(typeName)
	if !exists {
		return nil, fmt.Errorf("type %s not registered", typeName)
	}

	return definition.CreateFunc(dbPath)
}

// ValidateTypeSchema validates a type schema for consistency
func (etr *EnhancedTypeRegistry) ValidateTypeSchema(schema TypeSchema) error {
	if schema.Name == "" {
		return fmt.Errorf("type name is required")
	}

	// Validate dimensions
	for dimName, dimSchema := range schema.Dimensions {
		if dimSchema.Type == "enumerated" {
			if len(dimSchema.Values) == 0 {
				return fmt.Errorf("enumerated dimension %s must have values", dimName)
			}

			if dimSchema.Default != "" {
				found := false
				for _, value := range dimSchema.Values {
					if value == dimSchema.Default {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("default value %s for dimension %s not in values list", dimSchema.Default, dimName)
				}
			}
		} else if dimSchema.Type == "hierarchical" {
			if dimSchema.RefField == "" {
				return fmt.Errorf("hierarchical dimension %s must specify ref_field", dimName)
			}
		}
	}

	// Validate fields
	for fieldName, fieldSchema := range schema.Fields {
		if fieldSchema.Type == "" {
			return fmt.Errorf("field %s must have a type", fieldName)
		}

		// Check if type is supported
		if _, err := etr.parseGoType(fieldSchema.Type); err != nil {
			return fmt.Errorf("field %s has invalid type: %w", fieldName, err)
		}
	}

	return nil
}
