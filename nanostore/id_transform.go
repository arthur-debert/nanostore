package nanostore

import (
	"fmt"
	"strconv"
	"strings"
)

// IDTransformer handles transformations between full partition form and short form IDs
type IDTransformer struct {
	dimensionSet  *DimensionSet
	canonicalView *CanonicalView
}

// NewIDTransformer creates a new ID transformer
func NewIDTransformer(dimensionSet *DimensionSet, canonicalView *CanonicalView) *IDTransformer {
	return &IDTransformer{
		dimensionSet:  dimensionSet,
		canonicalView: canonicalView,
	}
}

// ToShortForm converts a partition to a user-facing short form ID
// Example: parent:1,status:pending,priority:medium|3 → 1.3 (with canonical status:pending,priority:medium)
func (t *IDTransformer) ToShortForm(partition Partition) string {
	// Extract canonical dimension values (to be omitted)
	canonicalValues := t.canonicalView.ExtractFromPartition(partition)

	// Build map of canonical dimensions for quick lookup
	canonicalMap := make(map[string]bool)
	for _, cv := range canonicalValues {
		canonicalMap[cv.Dimension] = true
	}

	// Collect non-canonical dimension values and prefixes
	var segments []string
	var currentSegment []string

	for _, dv := range partition.Values {
		dim, exists := t.dimensionSet.Get(dv.Dimension)
		if !exists {
			continue
		}

		switch dim.Type {
		case Hierarchical:
			// Hierarchical dimensions are always included in the ID structure
			// regardless of canonical filters
			if dv.Value != "" && dv.Value != "0" { // Skip empty/zero parent values
				// Split hierarchical value by dots and add each part
				parts := strings.Split(dv.Value, ".")
				for _, part := range parts {
					if part != "" {
						segments = append(segments, part)
					}
				}
			}
		case Enumerated:
			// Skip canonical enumerated dimensions
			if canonicalMap[dv.Dimension] {
				continue
			}
			// Non-canonical enumerated values become prefixes
			prefix := dim.GetPrefix(dv.Value)
			if prefix != "" {
				currentSegment = append(currentSegment, prefix)
			}
		}
	}

	// Add position to the current segment
	currentSegment = append(currentSegment, strconv.Itoa(partition.Position))

	// Combine current segment
	if len(currentSegment) > 0 {
		segments = append(segments, strings.Join(currentSegment, ""))
	}

	// Join all segments with dots
	return strings.Join(segments, ".")
}

// FromShortForm parses a short form ID into dimension values
// Returns partial partition information (canonical dimensions will need to be added)
// Example: 1.d2 → parent:1, status:done (inferred from 'd' prefix), position:2
func (t *IDTransformer) FromShortForm(shortForm string) (Partition, error) {
	if shortForm == "" {
		return Partition{}, fmt.Errorf("empty ID")
	}

	// Split by dots for hierarchical segments
	segments := strings.Split(shortForm, ".")

	var values []DimensionValue
	var position int

	// Build hierarchical path from all segments except the last
	var hierarchicalPath []string
	for i := 0; i < len(segments)-1; i++ {
		if segments[i] != "" {
			hierarchicalPath = append(hierarchicalPath, segments[i])
		}
	}

	// Add hierarchical dimension value if we have a path
	if len(hierarchicalPath) > 0 {
		hierarchical := t.dimensionSet.Hierarchical()
		if len(hierarchical) > 0 {
			values = append(values, DimensionValue{
				Dimension: hierarchical[0].Name,
				Value:     strings.Join(hierarchicalPath, "."),
			})
		}
	}

	// Process last segment for prefixes and position
	if len(segments) > 0 {
		lastSegment := segments[len(segments)-1]
		if lastSegment != "" {
			// Extract prefixes and position
			prefixes, pos, err := t.extractPrefixesAndPosition(lastSegment)
			if err != nil {
				return Partition{}, fmt.Errorf("invalid segment %q: %w", lastSegment, err)
			}
			position = pos

			// Convert prefixes to dimension values
			for prefix, dimName := range prefixes {
				dim, exists := t.dimensionSet.Get(dimName)
				if !exists {
					return Partition{}, fmt.Errorf("unknown dimension %q for prefix %q", dimName, prefix)
				}

				// Find value for this prefix
				for value, p := range dim.Prefixes {
					if p == prefix {
						values = append(values, DimensionValue{
							Dimension: dimName,
							Value:     value,
						})
						break
					}
				}
			}
		}
	}

	// Add canonical dimension values
	for _, filter := range t.canonicalView.Filters {
		// Skip wildcard filters for hierarchical dimensions
		if filter.Value == "*" {
			continue
		}

		// Check if we already have this dimension
		found := false
		for _, dv := range values {
			if dv.Dimension == filter.Dimension {
				found = true
				break
			}
		}

		if !found {
			values = append(values, DimensionValue(filter))
		}
	}

	return Partition{
		Values:   values,
		Position: position,
	}, nil
}

// extractPrefixesAndPosition extracts dimension prefixes and position from a segment
// Example: "dh3" → {d: status, h: priority}, position: 3
func (t *IDTransformer) extractPrefixesAndPosition(segment string) (map[string]string, int, error) {
	prefixes := make(map[string]string)

	// Find where the numeric position starts
	numStart := -1
	for i := len(segment) - 1; i >= 0; i-- {
		if segment[i] < '0' || segment[i] > '9' {
			numStart = i + 1
			break
		}
	}

	// If we didn't find any non-numeric character, entire segment is numeric
	if numStart == -1 {
		numStart = 0
	}

	// If entire segment is numeric, it's just a position
	if numStart == 0 {
		pos, err := strconv.Atoi(segment)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid position: %s", segment)
		}
		return prefixes, pos, nil
	}

	// Extract position
	if numStart >= len(segment) {
		return nil, 0, fmt.Errorf("missing position number")
	}

	position, err := strconv.Atoi(segment[numStart:])
	if err != nil {
		return nil, 0, fmt.Errorf("invalid position: %s", segment[numStart:])
	}

	// Extract prefixes
	prefixPart := segment[:numStart]

	// Build reverse map of prefix -> dimension name
	prefixToDim := make(map[string]string)
	for _, dim := range t.dimensionSet.Enumerated() {
		for _, prefix := range dim.Prefixes {
			prefixToDim[prefix] = dim.Name
		}
	}

	// Match prefixes (greedy approach - try longest prefixes first)
	remaining := prefixPart
	for len(remaining) > 0 {
		found := false
		// Try to match progressively shorter prefixes
		for prefixLen := len(remaining); prefixLen > 0; prefixLen-- {
			prefix := remaining[:prefixLen]
			if dimName, exists := prefixToDim[prefix]; exists {
				prefixes[prefix] = dimName
				remaining = remaining[prefixLen:]
				found = true
				break
			}
		}
		if !found {
			return nil, 0, fmt.Errorf("unknown prefix: %s", remaining[:1])
		}
	}

	return prefixes, position, nil
}

// NormalizeID ensures an ID is in the correct short form
// This is useful for handling various input formats
func (t *IDTransformer) NormalizeID(id string) (string, error) {
	// First try to parse as partition format
	if strings.Contains(id, ":") && strings.Contains(id, "|") {
		partition, err := ParsePartition(id)
		if err == nil {
			return t.ToShortForm(partition), nil
		}
	}

	// Otherwise parse as short form and reconstruct
	partition, err := t.FromShortForm(id)
	if err != nil {
		return "", err
	}

	return t.ToShortForm(partition), nil
}
