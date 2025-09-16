package nanostore

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// IDEngine handles the core ID generation and resolution logic for nanostore.
// This is the "secret sauce" that converts between UUIDs and user-facing sequential IDs.
//
// The IDEngine encapsulates:
// 1. ID Generation: Creates SQL SELECT clauses with ROW_NUMBER() for sequential IDs
// 2. ID Resolution: Converts user-facing IDs (e.g., "h1.2") back to UUIDs
// 3. ID Parsing: Parses hierarchical ID strings into structured filters
//
// This separation allows the core ID logic to be isolated from database operations
// and makes it easier to integrate with different query builders or ORMs in the future.
type IDEngine struct {
	config    Config
	db        *sql.DB
	prefixMap map[string]prefixMapping
}

// prefixMapping maps a prefix to its dimension and value
type prefixMapping struct {
	dimension string
	value     string
}

// parsedID represents a parsed hierarchical ID
type parsedID struct {
	Levels []parsedLevel
}

// parsedLevel represents a single level in a hierarchical ID
type parsedLevel struct {
	Offset           int               // 0-based offset within the filtered set
	DimensionFilters map[string]string // dimension -> value filters extracted from prefixes
}

// dimensionCombination represents a combination of dimension values and resulting prefix
type dimensionCombination struct {
	values map[string]string // dimension name -> value
	prefix string            // resulting prefix (alphabetically ordered)
}

// NewIDEngine creates a new ID engine with the given configuration and database connection
func NewIDEngine(config Config, db *sql.DB) *IDEngine {
	engine := &IDEngine{
		config:    config,
		db:        db,
		prefixMap: make(map[string]prefixMapping),
	}

	// Build prefix mapping from configuration
	for _, dim := range config.Dimensions {
		if dim.Type == Enumerated {
			for value, prefix := range dim.Prefixes {
				if prefix != "" {
					engine.prefixMap[prefix] = prefixMapping{
						dimension: dim.Name,
						value:     value,
					}
				}
			}
		}
	}

	return engine
}

// GenerateIDSelectClause creates a SQL SELECT clause that generates user-facing IDs
// using ROW_NUMBER() with proper partitioning by dimensions.
//
// For enumerated dimensions with prefixes, this generates a CASE statement that:
// 1. Matches dimension value combinations
// 2. Applies the appropriate prefix
// 3. Uses ROW_NUMBER() partitioned by those dimensions
//
// Example output:
//
//	CASE
//	  WHEN status = 'done' THEN 'd' || CAST(ROW_NUMBER() OVER (PARTITION BY status ORDER BY created_at) AS TEXT)
//	  WHEN priority = 'high' THEN 'h' || CAST(ROW_NUMBER() OVER (PARTITION BY priority ORDER BY created_at) AS TEXT)
//	  ELSE CAST(ROW_NUMBER() OVER (ORDER BY created_at) AS TEXT)
//	END AS user_facing_id
func (e *IDEngine) GenerateIDSelectClause(isRoot bool) string {
	enumDims := e.config.GetEnumeratedDimensions()

	if len(enumDims) == 0 {
		// No enumerated dimensions, just use row number
		return e.generateSimpleRowNumber(isRoot)
	}

	// Build CASE statement for prefix generation
	var expr strings.Builder
	expr.WriteString("CASE\n")

	// Generate all combinations of dimension values
	combinations := e.generateDimensionCombinations(enumDims)

	for _, combo := range combinations {
		expr.WriteString("        WHEN ")

		// Build condition for this combination
		conditions := make([]string, 0, len(combo.values))
		for dimName, value := range combo.values {
			conditions = append(conditions, fmt.Sprintf("%s = '%s'", dimName, value))
		}
		sort.Strings(conditions) // Ensure consistent ordering
		expr.WriteString(strings.Join(conditions, " AND "))

		expr.WriteString(" THEN\n")
		expr.WriteString("            '")
		expr.WriteString(combo.prefix)
		expr.WriteString("' || CAST(ROW_NUMBER() OVER (\n")
		expr.WriteString("                PARTITION BY ")

		// Partition by all dimensions (maintains separate numbering)
		partitions := make([]string, 0, len(enumDims))
		for _, dim := range enumDims {
			partitions = append(partitions, dim.Name)
		}
		if !isRoot {
			// For children, also partition by parent
			hierDim := e.findHierarchicalDimension()
			if hierDim != nil {
				partitions = append(partitions, hierDim.RefField)
			}
		}
		expr.WriteString(strings.Join(partitions, ", "))

		expr.WriteString("\n                ORDER BY created_at\n")
		expr.WriteString("            ) AS TEXT)\n")
	}

	// Default case (shouldn't happen with proper constraints)
	expr.WriteString("        ELSE CAST(ROW_NUMBER() OVER (ORDER BY created_at) AS TEXT)\n")
	expr.WriteString("    END")

	return expr.String()
}

// generateSimpleRowNumber creates a basic ROW_NUMBER expression without prefixes
func (e *IDEngine) generateSimpleRowNumber(isRoot bool) string {
	var partition string
	if !isRoot {
		hierDim := e.findHierarchicalDimension()
		if hierDim != nil {
			partition = fmt.Sprintf("PARTITION BY %s ", hierDim.RefField)
		}
	}

	return fmt.Sprintf("CAST(ROW_NUMBER() OVER (%sORDER BY created_at) AS TEXT)", partition)
}

// generateDimensionCombinations creates all combinations of enumerated dimension values
// and calculates the appropriate prefix for each combination
func (e *IDEngine) generateDimensionCombinations(enumDims []DimensionConfig) []dimensionCombination {
	if len(enumDims) == 0 {
		return []dimensionCombination{}
	}

	// Start with the first dimension
	var combinations []dimensionCombination
	for _, value := range enumDims[0].Values {
		combo := dimensionCombination{
			values: map[string]string{enumDims[0].Name: value},
		}
		combinations = append(combinations, combo)
	}

	// Add each subsequent dimension
	for _, dim := range enumDims[1:] {
		var newCombinations []dimensionCombination
		for _, combo := range combinations {
			for _, value := range dim.Values {
				newCombo := dimensionCombination{
					values: make(map[string]string),
				}
				// Copy existing values
				for k, v := range combo.values {
					newCombo.values[k] = v
				}
				// Add new dimension value
				newCombo.values[dim.Name] = value
				newCombinations = append(newCombinations, newCombo)
			}
		}
		combinations = newCombinations
	}

	// Calculate prefixes for each combination
	for i := range combinations {
		combinations[i].prefix = e.calculateCombinationPrefix(combinations[i].values, enumDims)
	}

	// Sort combinations by prefix for consistent output
	sort.Slice(combinations, func(i, j int) bool {
		return combinations[i].prefix < combinations[j].prefix
	})

	return combinations
}

// calculateCombinationPrefix determines the prefix for a combination of dimension values
func (e *IDEngine) calculateCombinationPrefix(values map[string]string, enumDims []DimensionConfig) string {
	var prefixParts []string

	// Sort dimension names for consistent prefix ordering
	dimNames := make([]string, 0, len(enumDims))
	for _, dim := range enumDims {
		dimNames = append(dimNames, dim.Name)
	}
	sort.Strings(dimNames)

	// Build prefix from dimension values in alphabetical order
	for _, dimName := range dimNames {
		if value, exists := values[dimName]; exists {
			// Find the dimension config
			for _, dim := range enumDims {
				if dim.Name == dimName {
					if prefix, hasPrefix := dim.Prefixes[value]; hasPrefix && prefix != "" {
						prefixParts = append(prefixParts, prefix)
					}
					break
				}
			}
		}
	}

	return strings.Join(prefixParts, "")
}

// findHierarchicalDimension returns the first hierarchical dimension in the config
func (e *IDEngine) findHierarchicalDimension() *DimensionConfig {
	for _, dim := range e.config.Dimensions {
		if dim.Type == Hierarchical {
			return &dim
		}
	}
	return nil
}

// ResolveID converts a user-facing ID (e.g., "h1.2") to a UUID
// This handles both simple IDs ("3") and hierarchical IDs ("1.2.h3")
func (e *IDEngine) ResolveID(userFacingID string) (string, error) {
	// Check if it's already a UUID
	if isUUIDFormat(userFacingID) {
		return userFacingID, nil
	}

	// Normalize the input ID to handle different prefix orders
	normalizedID, err := e.normalizeUserFacingID(userFacingID)
	if err != nil {
		return "", fmt.Errorf("failed to normalize ID: %w", err)
	}

	// Parse the normalized ID to extract hierarchy and filters
	parsedID, err := e.parseID(normalizedID)
	if err != nil {
		return "", fmt.Errorf("failed to parse ID: %w", err)
	}

	// Use optimized SQL-based resolution for single level IDs
	if len(parsedID.Levels) == 1 {
		return e.resolveUUIDFlat(parsedID.Levels[0])
	}

	// For multi-level hierarchical IDs, resolve level by level
	return e.resolveUUIDHierarchical(parsedID)
}

// parseID parses a user-facing ID string into a structured parsedID
func (e *IDEngine) parseID(userFacingID string) (*parsedID, error) {
	// Validate input doesn't contain SQL injection attempts
	if strings.ContainsAny(userFacingID, "'\"`;\\") {
		return nil, fmt.Errorf("invalid ID format: contains illegal characters")
	}

	// Split by dots for hierarchy levels
	parts := strings.Split(userFacingID, ".")

	parsed := &parsedID{
		Levels: make([]parsedLevel, len(parts)),
	}

	// Parse each level
	for i, part := range parts {
		level, err := e.parseLevel(part)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format at level %d: %w", i+1, err)
		}
		parsed.Levels[i] = level
	}

	return parsed, nil
}

// parseLevel parses a single level of an ID (e.g., "hp2" or "c1" or "3")
func (e *IDEngine) parseLevel(part string) (parsedLevel, error) {
	if part == "" {
		return parsedLevel{}, fmt.Errorf("empty ID segment")
	}

	level := parsedLevel{
		DimensionFilters: make(map[string]string),
	}

	// Extract prefixes (consecutive lowercase letters at the start)
	prefixEnd := 0
	for i, r := range part {
		if r >= 'a' && r <= 'z' {
			prefixEnd = i + 1
		} else {
			break
		}
	}

	// Parse the numeric part
	numberPart := part[prefixEnd:]
	if numberPart == "" {
		return parsedLevel{}, fmt.Errorf("missing number in ID: %s", part)
	}

	number, err := strconv.Atoi(numberPart)
	if err != nil {
		return parsedLevel{}, fmt.Errorf("invalid number format: %s", numberPart)
	}

	level.Offset = number - 1 // Convert to 0-based offset

	// Process prefixes
	if prefixEnd > 0 {
		prefixStr := part[:prefixEnd]
		err := e.processPrefixes(prefixStr, &level)
		if err != nil {
			return parsedLevel{}, fmt.Errorf("invalid prefix in %s: %w", part, err)
		}
	}

	return level, nil
}

// processPrefixes extracts dimension filters from prefix string
func (e *IDEngine) processPrefixes(prefixStr string, level *parsedLevel) error {
	// Extract individual prefixes by matching against known prefixes
	remainingPrefixes := prefixStr
	seenDimensions := make(map[string]bool)

	for remainingPrefixes != "" {
		found := false

		// Try to match known prefixes (longest first to handle overlaps)
		var sortedPrefixes []string
		for prefix := range e.prefixMap {
			sortedPrefixes = append(sortedPrefixes, prefix)
		}
		sort.Slice(sortedPrefixes, func(i, j int) bool {
			return len(sortedPrefixes[i]) > len(sortedPrefixes[j])
		})

		for _, prefix := range sortedPrefixes {
			if strings.HasPrefix(remainingPrefixes, prefix) {
				mapping := e.prefixMap[prefix]

				// Check for duplicate dimension
				if seenDimensions[mapping.dimension] {
					return fmt.Errorf("duplicate dimension prefix for %s", mapping.dimension)
				}
				seenDimensions[mapping.dimension] = true

				level.DimensionFilters[mapping.dimension] = mapping.value
				remainingPrefixes = remainingPrefixes[len(prefix):]
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("unknown prefix: %s", string(remainingPrefixes[0]))
		}
	}

	return nil
}

// normalizeUserFacingID sorts prefixes within each level to create a canonical form
func (e *IDEngine) normalizeUserFacingID(userFacingID string) (string, error) {
	levels := strings.Split(userFacingID, ".")
	normalizedLevels := make([]string, len(levels))

	for i, level := range levels {
		normalized, err := e.normalizeLevelPrefixes(level)
		if err != nil {
			return "", err
		}
		normalizedLevels[i] = normalized
	}

	return strings.Join(normalizedLevels, "."), nil
}

// normalizeLevelPrefixes sorts prefixes within a single level
func (e *IDEngine) normalizeLevelPrefixes(level string) (string, error) {
	// Extract prefixes and number
	prefixEnd := 0
	for i, r := range level {
		if r >= 'a' && r <= 'z' {
			prefixEnd = i + 1
		} else {
			break
		}
	}

	if prefixEnd == 0 {
		return level, nil // No prefixes to normalize
	}

	prefixStr := level[:prefixEnd]
	numberStr := level[prefixEnd:]

	// Parse prefixes into a temporary level
	tempLevel := parsedLevel{DimensionFilters: make(map[string]string)}
	err := e.processPrefixes(prefixStr, &tempLevel)
	if err != nil {
		return "", err
	}

	// Rebuild prefix string in alphabetical order
	var prefixParts []string
	var dimensionNames []string
	for dimName := range tempLevel.DimensionFilters {
		dimensionNames = append(dimensionNames, dimName)
	}
	sort.Strings(dimensionNames)

	for _, dimName := range dimensionNames {
		value := tempLevel.DimensionFilters[dimName]
		// Find the prefix for this dimension/value combination
		for _, dim := range e.config.Dimensions {
			if dim.Name == dimName {
				if prefix, hasPrefix := dim.Prefixes[value]; hasPrefix && prefix != "" {
					prefixParts = append(prefixParts, prefix)
				}
				break
			}
		}
	}

	sort.Strings(prefixParts)
	return strings.Join(prefixParts, "") + numberStr, nil
}

// resolveUUIDFlat resolves a single-level ID using optimized SQL
func (e *IDEngine) resolveUUIDFlat(level parsedLevel) (string, error) {
	// Build SQL query with ROW_NUMBER() partitioning
	var whereClauses []string
	var args []interface{}

	// Add dimension filters
	enumDims := e.config.GetEnumeratedDimensions()
	var partitionCols []string

	for _, dim := range enumDims {
		if filterValue, hasFilter := level.DimensionFilters[dim.Name]; hasFilter {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", dim.Name))
			args = append(args, filterValue)
			partitionCols = append(partitionCols, dim.Name)
		} else {
			// Use default value if no specific filter
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", dim.Name))
			args = append(args, dim.DefaultValue)
			partitionCols = append(partitionCols, dim.Name)
		}
	}

	// Handle hierarchical dimension (should be NULL for root level)
	hierDims := e.config.GetHierarchicalDimensions()
	for _, dim := range hierDims {
		whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", dim.RefField))
		partitionCols = append(partitionCols, dim.RefField)
	}

	// Build the query
	partitionBy := strings.Join(partitionCols, ", ")
	whereClause := strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		WITH numbered_docs AS (
			SELECT uuid, ROW_NUMBER() OVER (PARTITION BY %s ORDER BY created_at) as row_num
			FROM documents 
			WHERE %s
		)
		SELECT uuid FROM numbered_docs WHERE row_num = ?`,
		partitionBy, whereClause)

	args = append(args, level.Offset+1) // Convert 0-based to 1-based

	var uuid string
	err := e.db.QueryRow(query, args...).Scan(&uuid)
	if err != nil {
		return "", fmt.Errorf("document not found")
	}

	return uuid, nil
}

// resolveUUIDHierarchical resolves multi-level hierarchical IDs by walking the tree level by level
func (e *IDEngine) resolveUUIDHierarchical(parsedID *parsedID) (string, error) {
	// Start with the root level
	currentUUID, err := e.resolveUUIDFlat(parsedID.Levels[0])
	if err != nil {
		return "", fmt.Errorf("failed to resolve root level: %w", err)
	}

	// Resolve each subsequent level
	for i := 1; i < len(parsedID.Levels); i++ {
		currentUUID, err = e.resolveChildUUID(currentUUID, parsedID.Levels[i])
		if err != nil {
			return "", fmt.Errorf("failed to resolve level %d: %w", i+1, err)
		}
	}

	return currentUUID, nil
}

// resolveChildUUID finds a child document given parent UUID and level constraints
func (e *IDEngine) resolveChildUUID(parentUUID string, level parsedLevel) (string, error) {
	// Get hierarchical dimension
	hierDims := e.config.GetHierarchicalDimensions()
	if len(hierDims) == 0 {
		return "", fmt.Errorf("no hierarchical dimension configured")
	}
	hierDim := hierDims[0] // Use first hierarchical dimension

	// Build SQL query with ROW_NUMBER() partitioning for children
	var whereClauses []string
	var args []interface{}

	// Parent constraint
	whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", hierDim.RefField))
	args = append(args, parentUUID)

	// Add dimension filters
	enumDims := e.config.GetEnumeratedDimensions()
	var partitionCols []string
	partitionCols = append(partitionCols, hierDim.RefField) // Always partition by parent

	for _, dim := range enumDims {
		if filterValue, hasFilter := level.DimensionFilters[dim.Name]; hasFilter {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", dim.Name))
			args = append(args, filterValue)
			partitionCols = append(partitionCols, dim.Name)
		} else {
			// Use default value if no specific filter
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", dim.Name))
			args = append(args, dim.DefaultValue)
			partitionCols = append(partitionCols, dim.Name)
		}
	}

	// Build the query
	partitionBy := strings.Join(partitionCols, ", ")
	whereClause := strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		WITH numbered_docs AS (
			SELECT uuid, ROW_NUMBER() OVER (PARTITION BY %s ORDER BY created_at) as row_num
			FROM documents 
			WHERE %s
		)
		SELECT uuid FROM numbered_docs WHERE row_num = ?`,
		partitionBy, whereClause)

	args = append(args, level.Offset+1) // Convert 0-based to 1-based

	var uuid string
	err := e.db.QueryRow(query, args...).Scan(&uuid)
	if err != nil {
		return "", fmt.Errorf("child document not found")
	}

	return uuid, nil
}

// IsUUIDFormat checks if a string is in UUID format (exported method)
func (e *IDEngine) IsUUIDFormat(id string) bool {
	return isUUIDFormat(id)
}

// isUUIDFormat checks if a string is in UUID format
func isUUIDFormat(id string) bool {
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 characters)
	if len(id) != 36 {
		return false
	}

	// Check dash positions
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}

	// Check that all other characters are hex digits
	for i, r := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // Skip dashes
		}
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}

	return true
}
