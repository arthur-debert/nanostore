package nanostore

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// idParser handles configurable ID parsing based on dimension configuration
type idParser struct {
	config Config
	// prefixToDimension maps single-letter prefixes to dimension name and value
	prefixMap map[string]prefixMapping
}

// prefixMapping maps a prefix to its dimension and value
type prefixMapping struct {
	dimension string
	value     string
}

// newIDParser creates a new ID parser for the given configuration
func newIDParser(config Config) *idParser {
	parser := &idParser{
		config:    config,
		prefixMap: make(map[string]prefixMapping),
	}

	// Build prefix mapping from configuration
	for _, dim := range config.Dimensions {
		if dim.Type == Enumerated {
			for value, prefix := range dim.Prefixes {
				if prefix != "" {
					parser.prefixMap[prefix] = prefixMapping{
						dimension: dim.Name,
						value:     value,
					}
				}
			}
		}
	}

	return parser
}

// parsedID represents a parsed hierarchical ID
type parsedID struct {
	Levels []parsedLevel
}

// parsedLevel represents one level in a hierarchical ID
type parsedLevel struct {
	// DimensionFilters maps dimension name to value for this level
	DimensionFilters map[string]string
	// Offset is the 0-based position within the filtered set
	Offset int
}

// parseID parses a user-facing ID into structured components for SQL resolution.
// This reverses the ID generation process - converting "hp2.c1" back into filters and offsets.
//
// Parsing Algorithm:
// 1. Split by "." to handle hierarchical levels (e.g., "1.2.3" = 3 levels)
// 2. For each level, extract prefixes and numeric ID:
//   - "hp2" -> prefixes: ["h", "p"], offset: 1 (2-1, convert to 0-based)
//   - "c1"  -> prefixes: ["c"], offset: 0
//   - "3"   -> prefixes: [], offset: 2
//
// 3. Map prefixes to dimension values using configuration
// 4. Return structured parsedID with filters for SQL query generation
//
// Example Input/Output:
// Input: "hp2.c1"
//
//	Output: parsedID{
//	  Levels: [
//	    {DimensionFilters: {"priority": "high", "status": "pending"}, Offset: 1},
//	    {DimensionFilters: {"status": "completed"}, Offset: 0}
//	  ]
//	}
//
// Security: Validates input against SQL injection patterns before processing.
func (p *idParser) parseID(userFacingID string) (*parsedID, error) {
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
		level, err := p.parseLevel(part)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format at level %d: %w", i+1, err)
		}
		parsed.Levels[i] = level
	}

	return parsed, nil
}

// parseLevel parses a single level of an ID (e.g., "hp2" or "c1" or "3")
func (p *idParser) parseLevel(part string) (parsedLevel, error) {
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

	if number < 1 {
		return parsedLevel{}, fmt.Errorf("ID number must be positive: %d", number)
	}

	level.Offset = number - 1 // Convert to 0-based offset

	// Parse prefixes if any
	if prefixEnd > 0 {
		prefixes := part[:prefixEnd]

		// Handle each prefix character
		for _, prefix := range prefixes {
			prefixStr := string(prefix)

			mapping, found := p.prefixMap[prefixStr]
			if !found {
				return parsedLevel{}, fmt.Errorf("unknown prefix: %s", prefixStr)
			}

			// Check for duplicate dimension filters
			if _, exists := level.DimensionFilters[mapping.dimension]; exists {
				return parsedLevel{}, fmt.Errorf("duplicate dimension filter for %s", mapping.dimension)
			}

			level.DimensionFilters[mapping.dimension] = mapping.value
		}
	}

	// Fill in default values for missing enumerated dimensions
	p.fillDefaultValues(&level)

	return level, nil
}

// fillDefaultValues adds default values for enumerated dimensions not specified in prefixes
func (p *idParser) fillDefaultValues(level *parsedLevel) {
	for _, dim := range p.config.Dimensions {
		if dim.Type == Enumerated {
			// Skip if already specified via prefix
			if _, exists := level.DimensionFilters[dim.Name]; exists {
				continue
			}

			// Add default value
			defaultValue := dim.DefaultValue
			if defaultValue == "" && len(dim.Values) > 0 {
				defaultValue = dim.Values[0]
			}

			if defaultValue != "" {
				level.DimensionFilters[dim.Name] = defaultValue
			}
		}
	}
}


// normalizePrefixes reorders prefixes alphabetically by dimension name
// This allows "ph1" and "hp1" to both resolve to the same normalized form
func (p *idParser) normalizePrefixes(prefixes string) string {
	if len(prefixes) == 0 {
		return ""
	}

	// Map each prefix to its dimension, tracking seen dimensions
	type prefixDim struct {
		prefix    string
		dimension string
	}

	var prefixDims []prefixDim
	seenDimensions := make(map[string]bool)

	for _, r := range prefixes {
		prefix := string(r)
		if mapping, found := p.prefixMap[prefix]; found {
			// Skip if we've already seen this dimension
			if seenDimensions[mapping.dimension] {
				continue
			}
			seenDimensions[mapping.dimension] = true

			prefixDims = append(prefixDims, prefixDim{
				prefix:    prefix,
				dimension: mapping.dimension,
			})
		}
	}

	// Sort by dimension name
	sort.Slice(prefixDims, func(i, j int) bool {
		return prefixDims[i].dimension < prefixDims[j].dimension
	})

	// Rebuild prefix string
	var normalized strings.Builder
	for _, pd := range prefixDims {
		normalized.WriteString(pd.prefix)
	}

	return normalized.String()
}

