package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/arthur-debert/nanostore/nanostore/types"
)

// IDParser handles configurable ID parsing based on dimension configuration
type IDParser struct {
	config types.Config
	// prefixToDimension maps single-letter prefixes to dimension name and value
	prefixMap map[string]prefixMapping
}

// prefixMapping maps a prefix to its dimension and value
type prefixMapping struct {
	dimension string
	value     string
}

// NewIDParser creates a new ID parser for the given configuration
func NewIDParser(config types.Config) *IDParser {
	parser := &IDParser{
		config:    config,
		prefixMap: make(map[string]prefixMapping),
	}

	// Build prefix mapping from configuration
	for _, dim := range config.Dimensions {
		if dim.Type == types.Enumerated {
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

// ParsedID represents a parsed hierarchical ID
type ParsedID struct {
	Levels []ParsedLevel
}

// ParsedLevel represents one level in a hierarchical ID
type ParsedLevel struct {
	// DimensionFilters maps dimension name to value for this level
	DimensionFilters map[string]string
	// Offset is the 0-based position within the filtered set
	Offset int
}

// ParseID parses a user-facing ID into structured components
func (p *IDParser) ParseID(userFacingID string) (*ParsedID, error) {
	// Validate input doesn't contain SQL injection attempts
	if strings.ContainsAny(userFacingID, "'\"`;\\") {
		return nil, fmt.Errorf("invalid ID format: contains illegal characters")
	}

	// Split by dots for hierarchy levels
	parts := strings.Split(userFacingID, ".")

	parsed := &ParsedID{
		Levels: make([]ParsedLevel, len(parts)),
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
func (p *IDParser) parseLevel(part string) (ParsedLevel, error) {
	if part == "" {
		return ParsedLevel{}, fmt.Errorf("empty ID segment")
	}

	level := ParsedLevel{
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
		return ParsedLevel{}, fmt.Errorf("missing number in ID: %s", part)
	}

	number, err := strconv.Atoi(numberPart)
	if err != nil {
		return ParsedLevel{}, fmt.Errorf("invalid number format: %s", numberPart)
	}

	if number < 1 {
		return ParsedLevel{}, fmt.Errorf("ID number must be positive: %d", number)
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
				return ParsedLevel{}, fmt.Errorf("unknown prefix: %s", prefixStr)
			}

			// Check for duplicate dimension filters
			if _, exists := level.DimensionFilters[mapping.dimension]; exists {
				return ParsedLevel{}, fmt.Errorf("duplicate dimension filter for %s", mapping.dimension)
			}

			level.DimensionFilters[mapping.dimension] = mapping.value
		}
	}

	// Fill in default values for missing enumerated dimensions
	p.fillDefaultValues(&level)

	return level, nil
}

// fillDefaultValues adds default values for enumerated dimensions not specified in prefixes
func (p *IDParser) fillDefaultValues(level *ParsedLevel) {
	for _, dim := range p.config.Dimensions {
		if dim.Type == types.Enumerated {
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

// GenerateID creates a user-facing ID from dimension values
func (p *IDParser) GenerateID(dimensionValues map[string]string, offset int) string {
	// Build a list of dimension-prefix pairs
	type dimPrefix struct {
		dimension string
		prefix    string
	}

	var dimPrefixes []dimPrefix

	// Collect prefixes for each dimension
	for _, dim := range p.config.Dimensions {
		if dim.Type == types.Enumerated {
			value, exists := dimensionValues[dim.Name]
			if !exists {
				continue
			}

			// Get prefix for this value
			if prefix, hasPrefix := dim.Prefixes[value]; hasPrefix && prefix != "" {
				dimPrefixes = append(dimPrefixes, dimPrefix{
					dimension: dim.Name,
					prefix:    prefix,
				})
			}
		}
	}

	// Sort by dimension name for alphabetical ordering
	sort.Slice(dimPrefixes, func(i, j int) bool {
		return dimPrefixes[i].dimension < dimPrefixes[j].dimension
	})

	// Build prefix string
	var prefixStr strings.Builder
	for _, dp := range dimPrefixes {
		prefixStr.WriteString(dp.prefix)
	}

	return fmt.Sprintf("%s%d", prefixStr.String(), offset+1)
}

// NormalizePrefixes reorders prefixes alphabetically by dimension name
// This allows "ph1" and "hp1" to both resolve to the same normalized form
func (p *IDParser) NormalizePrefixes(prefixes string) string {
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

// ValidateConfiguration checks if the parser configuration is valid
func (p *IDParser) ValidateConfiguration() error {
	// Check for prefix conflicts (same prefix used by multiple dimensions)
	prefixUsage := make(map[string][]string)

	for _, dim := range p.config.Dimensions {
		if dim.Type == types.Enumerated {
			for value, prefix := range dim.Prefixes {
				if prefix != "" {
					key := fmt.Sprintf("%s.%s", dim.Name, value)
					prefixUsage[prefix] = append(prefixUsage[prefix], key)
				}
			}
		}
	}

	// Report any conflicts
	for prefix, usages := range prefixUsage {
		if len(usages) > 1 {
			return fmt.Errorf("prefix '%s' is used by multiple dimension values: %s",
				prefix, strings.Join(usages, ", "))
		}
	}

	return nil
}
