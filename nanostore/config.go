package nanostore

import (
	"fmt"
	"strings"

	"github.com/arthur-debert/nanostore/nanostore/types"
)

// DefaultConfig returns a configuration that matches the original hardcoded behavior
// This provides backward compatibility for existing applications
func DefaultConfig() types.Config {
	return types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     types.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}
}

// ValidateConfig checks the configuration for consistency and completeness
func ValidateConfig(c types.Config) error {
	if len(c.Dimensions) == 0 {
		return fmt.Errorf("at least one dimension must be configured")
	}

	// Enforce dimension limit for performance
	const maxDimensions = 7
	if len(c.Dimensions) > maxDimensions {
		return fmt.Errorf("too many dimensions: %d (maximum %d)", len(c.Dimensions), maxDimensions)
	}

	// Track dimension names to check for duplicates
	namesSeen := make(map[string]bool)

	// Track prefixes to check for conflicts
	prefixesSeen := make(map[string]string)

	for i, dim := range c.Dimensions {
		// Validate dimension name
		if dim.Name == "" {
			return fmt.Errorf("dimension %d: name cannot be empty", i)
		}

		// Check for reserved column names
		if isReservedColumnName(dim.Name) {
			return fmt.Errorf("dimension %d: '%s' is a reserved column name", i, dim.Name)
		}

		// Check for duplicate names
		if namesSeen[dim.Name] {
			return fmt.Errorf("dimension %d: duplicate dimension name '%s'", i, dim.Name)
		}
		namesSeen[dim.Name] = true

		// Validate based on dimension type
		switch dim.Type {
		case types.Enumerated:
			if err := validateEnumeratedDimension(dim, i, prefixesSeen); err != nil {
				return err
			}
		case types.Hierarchical:
			if err := validateHierarchicalDimension(dim, i); err != nil {
				return err
			}
		default:
			return fmt.Errorf("dimension %d: invalid dimension type %d", i, dim.Type)
		}
	}

	return nil
}

// validateEnumeratedDimension validates an enumerated dimension configuration
func validateEnumeratedDimension(dim types.DimensionConfig, index int, prefixesSeen map[string]string) error {
	// Must have at least one value
	if len(dim.Values) == 0 {
		return fmt.Errorf("dimension %d (%s): enumerated dimensions must have at least one value", index, dim.Name)
	}

	// Check for duplicate values
	valuesSeen := make(map[string]bool)
	for _, value := range dim.Values {
		if value == "" {
			return fmt.Errorf("dimension %d (%s): values cannot be empty", index, dim.Name)
		}
		if valuesSeen[value] {
			return fmt.Errorf("dimension %d (%s): duplicate value '%s'", index, dim.Name, value)
		}
		valuesSeen[value] = true
	}

	// Validate default value if specified
	if dim.DefaultValue != "" {
		if !valuesSeen[dim.DefaultValue] {
			return fmt.Errorf("dimension %d (%s): default value '%s' is not in values list", index, dim.Name, dim.DefaultValue)
		}
	}

	// Validate prefixes
	for value, prefix := range dim.Prefixes {
		// Value must be in the values list
		if !valuesSeen[value] {
			return fmt.Errorf("dimension %d (%s): prefix defined for unknown value '%s'", index, dim.Name, value)
		}

		// Prefix cannot be empty
		if prefix == "" {
			return fmt.Errorf("dimension %d (%s): prefix for value '%s' cannot be empty", index, dim.Name, value)
		}

		// Prefix must be valid for ID parsing
		if !isValidPrefix(prefix) {
			return fmt.Errorf("dimension %d (%s): prefix '%s' contains invalid characters", index, dim.Name, prefix)
		}

		// Check for prefix conflicts across dimensions
		if existingDim, exists := prefixesSeen[prefix]; exists {
			return fmt.Errorf("dimension %d (%s): prefix '%s' conflicts with dimension '%s'", index, dim.Name, prefix, existingDim)
		}
		prefixesSeen[prefix] = dim.Name
	}

	// RefField should not be set for enumerated dimensions
	if dim.RefField != "" {
		return fmt.Errorf("dimension %d (%s): RefField should not be set for enumerated dimensions", index, dim.Name)
	}

	return nil
}

// validateHierarchicalDimension validates a hierarchical dimension configuration
func validateHierarchicalDimension(dim types.DimensionConfig, index int) error {
	// Must have RefField
	if dim.RefField == "" {
		return fmt.Errorf("dimension %d (%s): hierarchical dimensions must specify RefField", index, dim.Name)
	}

	// RefField cannot be a reserved name
	if isReservedColumnName(dim.RefField) {
		return fmt.Errorf("dimension %d (%s): RefField '%s' is a reserved column name", index, dim.Name, dim.RefField)
	}

	// Values should not be set for hierarchical dimensions
	if len(dim.Values) > 0 {
		return fmt.Errorf("dimension %d (%s): Values should not be set for hierarchical dimensions", index, dim.Name)
	}

	// Prefixes should not be set for hierarchical dimensions
	if len(dim.Prefixes) > 0 {
		return fmt.Errorf("dimension %d (%s): Prefixes should not be set for hierarchical dimensions", index, dim.Name)
	}

	// DefaultValue should not be set for hierarchical dimensions
	if dim.DefaultValue != "" {
		return fmt.Errorf("dimension %d (%s): DefaultValue should not be set for hierarchical dimensions", index, dim.Name)
	}

	return nil
}

// isReservedColumnName checks if a column name is reserved by the system
func isReservedColumnName(name string) bool {
	reserved := []string{
		"uuid", "title", "body", "created_at", "updated_at",
		// SQL keywords that could cause issues
		"select", "from", "where", "order", "by", "group", "having",
		"insert", "update", "delete", "create", "drop", "alter",
	}

	name = strings.ToLower(name)
	for _, reservedName := range reserved {
		if name == reservedName {
			return true
		}
	}

	return false
}

// isValidPrefix checks if a prefix contains only valid characters for ID parsing
func isValidPrefix(prefix string) bool {
	// Prefixes should only contain lowercase letters
	// This avoids conflicts with numbers and special characters used in ID parsing
	for _, r := range prefix {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}
