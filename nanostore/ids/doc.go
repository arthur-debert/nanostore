// Package ids provides the core ID generation and transformation system for nanostore.
//
//	Overview
//
// The ID system is designed to generate human-readable, hierarchical, and stable IDs
// for documents while maintaining performance and avoiding collisions. This system
// replaces traditional auto-incrementing IDs with a sophisticated partition-based
// approach that reflects the document's dimensional characteristics.
//
//	Core Concepts
//
//	types.Partitions
//
// A partition represents a logical grouping of documents based on their dimension values.
// Documents with the same dimension values belong to the same partition and are assigned
// sequential positions within that partition.
//
// Example partition: `status:active,priority:high|3`
//
//   - Dimension values: status=active, priority=high
//
//   - Position within partition: 3
//
//   - This represents the 3rd document with status=active AND priority=high
//
//     Canonical View
//
// The canonical view defines "default" dimension values that can be omitted from IDs
// to keep them shorter and cleaner. When a document has canonical dimension values,
// those dimensions are not included in the visible ID.
//
// Example with canonical view (status:active, priority:medium):
//
//   - Document with status=active, priority=medium, position=5 → ID: "5"
//
//   - Document with status=done, priority=high, position=2 → ID: "dh2"
//
//     Hierarchical Dimensions
//
// Hierarchical dimensions create parent-child relationships between documents,
// resulting in nested ID structures that reflect the document hierarchy.
//
// Example with hierarchical "location" dimension:
//
//   - Root document (location=null): ID "1"
//
//   - Child document (parent=1): ID "1.1"
//
//   - Grandchild document (parent=1.1): ID "1.1.1"
//
//     Prefixes
//
// Enumerated dimensions can have single-character prefixes that replace full
// dimension values in the short form ID, making IDs more compact.
//
// Example with prefixes:
//
//   - status: active="", done="d", pending="p"
//
//   - priority: low="l", medium="", high="h"
//
//   - Document with status=done, priority=high, position=3 → ID: "dh3"
//
//     ID Transformation Process
//
//     Short Form Generation (ToShortForm)
//
// 1. Extract Canonical Values: Identify which dimension values match the canonical view
// 2. Process Hierarchical Dimensions: Build the hierarchical path (e.g., "1.2.3")
// 3. Apply Prefixes: Convert non-canonical enumerated values to prefixes
// 4. Append Position: Add the document's position within its partition
// 5. Combine Segments: Join hierarchical and enumerated parts with dots
//
// Examples:
//
//   - types.Partition `status:active,priority:medium|5` → ID: `5` (all canonical)
//
//   - types.Partition `parent:1,status:done,priority:high|2` → ID: `1.dh2`
//
//   - types.Partition `parent:1.2,status:active,priority:low|3` → ID: `1.2.l3`
//
//     Short Form Parsing (FromShortForm)
//
// 1. Split by Dots: Separate hierarchical segments from the final segment
// 2. Extract Hierarchical Path: Build parent relationships from dot-separated parts
// 3. Parse Final Segment: Extract prefixes and position number
// 4. Resolve Prefixes: Convert single-character prefixes back to dimension values
// 5. Add Canonical Values: Fill in default values from canonical view
// 6. Construct types.Partition: Build complete partition with all dimension values
//
// Examples:
//
//   - ID `5` → types.Partition: `status:active,priority:medium|5`
//
//   - ID `1.dh2` → types.Partition: `parent:1,status:done,priority:high|2`
//
//   - ID `1.2.l3` → types.Partition: `parent:1.2,status:active,priority:low|3`
//
//     ID Generation Process
//
//     Multi-Pass Assignment
//
// The ID generator uses a multi-pass approach to handle hierarchical relationships:
//
// 1. Pass 1: Assign IDs to root documents (no parent)
// 2. Pass 2: Assign IDs to documents whose parents now have IDs
// 3. Pass N: Continue until all documents have IDs or maximum depth reached
//
// This ensures that parent IDs are always available when generating child IDs.
//
//	Stable Positioning
//
// Positions within partitions are stable and based on document creation time.
// Once assigned, a document's position never changes, even if other documents
// in the same partition are deleted.
//
// Implementation details:
//
//   - Documents are sorted by CreatedAt timestamp within each partition
//
//   - Positions are assigned sequentially (1, 2, 3, ...)
//
//   - Deleted documents leave gaps in the sequence
//
//   - New documents always get the next available position
//
//     Historical types.Partition Mapping
//
// The generator considers the historical membership of documents in partitions
// to maintain stable positions even when dimension values change:
//
//   - Track all documents that have ever belonged to each partition
//
//   - Sort by creation time to determine original insertion order
//
//   - Assign positions based on this historical order
//
//   - Only count documents currently in the partition for actual ID assignment
//
//     Performance Characteristics
//
//     Time Complexity
//
//   - ID Generation: O(n log n) where n = number of documents
//
//   - ID Resolution: O(log n) with efficient partition lookup
//
//   - Transformation: O(1) for short form conversion
//
//     Space Complexity
//
//   - Memory usage is linear with document count
//
//   - types.Partition maps are cached for efficient lookups
//
//   - No persistent ID storage required (generated on-demand)
//
//     Scalability Considerations
//
//   - types.Partition-based approach scales well with document growth
//
//   - Hierarchical dimensions may create deep nesting (configurable max depth)
//
//   - Prefix conflicts are detected during configuration validation
//
//     Thread Safety
//
// All ID operations are thread-safe:
//
//   - IDGenerator and IDTransformer are immutable after creation
//
//   - Document lists are copied before sorting to avoid mutation
//
//   - Concurrent ID generation calls are safe
//
//     Error Handling
//
// The system provides detailed error messages for common issues:
//
//   - Invalid ID format during parsing
//
//   - Unknown prefixes or dimension values
//
//   - Circular parent relationships
//
//   - Missing required dimension values
//
//   - types.Partition key generation failures
//
//     Usage Examples
//
//     // Create ID system components
//     dimensionSet := config.Gettypes.DimensionSet()
//     canonicalView := Newtypes.CanonicalView(
//     CanonicalFilter{Dimension: "status", Value: "active"},
//     CanonicalFilter{Dimension: "priority", Value: "medium"},
//     )
//
//     // Initialize generator and transformer
//     generator := NewIDGenerator(dimensionSet, canonicalView)
//     transformer := NewIDTransformer(dimensionSet, canonicalView)
//
//     // Generate IDs for documents
//     idMap := generator.GenerateIDs(documents) // SimpleID -> UUID
//
//     // Transform between formats
//     shortID := transformer.ToShortForm(partition)     // "1.dh3"
//     partition, err := transformer.FromShortForm("1.dh3") // Parse back
//
//     // Resolve SimpleID to UUID
//     uuid, err := generator.ResolveID("1.dh3", documents)
//
//     Integration with Store
//
// The ID system integrates seamlessly with the document store:
//   - IDs are generated dynamically during List operations
//   - Command preprocessing resolves SimpleIDs to UUIDs before operations
//   - No persistent ID storage is required
//   - Dimension changes automatically update ID structure
package ids
