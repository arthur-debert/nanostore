package validation

import (
	"fmt"
	"strings"

	"github.com/arthur-debert/nanostore/types"
)

// Validate checks the dimension set for consistency
func Validate(ds *types.DimensionSet) error {
	if ds.Count() == 0 {
		return fmt.Errorf("at least one dimension must be configured")
	}

	// Enforce dimension limit for performance
	const maxDimensions = 7
	if ds.Count() > maxDimensions {
		return fmt.Errorf("too many dimensions: %d (maximum %d)", ds.Count(), maxDimensions)
	}

	// Check for duplicate names
	seen := make(map[string]bool)
	for _, dim := range ds.All() {
		if seen[dim.Name] {
			return fmt.Errorf("duplicate dimension name: %s", dim.Name)
		}
		seen[dim.Name] = true
	}

	// Track prefixes to check for conflicts
	prefixesSeen := make(map[string]string)

	for _, dim := range ds.All() {
		// Validate dimension name
		if dim.Name == "" {
			return fmt.Errorf("dimension name cannot be empty")
		}

		// Check for reserved column names
		if IsReservedColumnName(dim.Name) {
			return fmt.Errorf("'%s' is a reserved column name", dim.Name)
		}

		// Validate based on dimension type
		switch dim.Type {
		case types.Enumerated:
			if err := validateEnumeratedDim(&dim, prefixesSeen); err != nil {
				return err
			}
		case types.Hierarchical:
			if err := validateHierarchicalDim(&dim); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid dimension type %d for %s", dim.Type, dim.Name)
		}
	}

	return nil
}

// validateEnumeratedDim validates an enumerated dimension
func validateEnumeratedDim(dim *types.Dimension, prefixesSeen map[string]string) error {
	// Must have at least one value
	if len(dim.Values) == 0 {
		return fmt.Errorf("dimension %s: enumerated dimensions must have at least one value", dim.Name)
	}

	// Check for duplicate values
	valuesSeen := make(map[string]bool)
	for _, value := range dim.Values {
		if value == "" {
			return fmt.Errorf("dimension %s: values cannot be empty", dim.Name)
		}
		if valuesSeen[value] {
			return fmt.Errorf("dimension %s: duplicate value '%s'", dim.Name, value)
		}
		valuesSeen[value] = true
	}

	// Validate default value if specified
	if dim.DefaultValue != "" {
		if !dim.IsValid(dim.DefaultValue) {
			return fmt.Errorf("dimension %s: default value '%s' is not in the list of valid values", dim.Name, dim.DefaultValue)
		}
	}

	// Validate prefixes
	for value, prefix := range dim.Prefixes {
		if !dim.IsValid(value) {
			return fmt.Errorf("dimension %s: prefix defined for invalid value '%s'", dim.Name, value)
		}
		if prefix == "" {
			return fmt.Errorf("dimension %s: empty prefix for value '%s'", dim.Name, value)
		}

		// Prefix must be valid for ID parsing
		if !IsValidPrefix(prefix) {
			return fmt.Errorf("dimension %s: prefix '%s' contains invalid characters", dim.Name, prefix)
		}

		// Check for prefix conflicts
		if existingDim, exists := prefixesSeen[prefix]; exists {
			return fmt.Errorf("dimension %s: prefix '%s' conflicts with dimension %s", dim.Name, prefix, existingDim)
		}
		prefixesSeen[prefix] = dim.Name
	}

	return nil
}

// validateHierarchicalDim validates a hierarchical dimension
func validateHierarchicalDim(dim *types.Dimension) error {
	// Must have RefField
	if dim.RefField == "" {
		return fmt.Errorf("dimension %s: hierarchical dimensions must specify RefField", dim.Name)
	}

	// RefField cannot be a reserved name
	if IsReservedColumnName(dim.RefField) {
		return fmt.Errorf("dimension %s: RefField '%s' is a reserved column name", dim.Name, dim.RefField)
	}

	// Should not have values or prefixes
	if len(dim.Values) > 0 {
		return fmt.Errorf("dimension %s: hierarchical dimensions should not have values", dim.Name)
	}
	if len(dim.Prefixes) > 0 {
		return fmt.Errorf("dimension %s: hierarchical dimensions should not have prefixes", dim.Name)
	}

	return nil
}

// IsReservedColumnName checks if a column name is reserved by the system
func IsReservedColumnName(name string) bool {
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

// IsValidPrefix checks if a prefix contains only valid characters for ID parsing
func IsValidPrefix(prefix string) bool {
	// Prefixes should only contain lowercase letters
	// This avoids conflicts with numbers and special characters used in ID parsing
	for _, r := range prefix {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}