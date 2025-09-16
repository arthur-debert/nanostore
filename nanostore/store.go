package nanostore

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Embedded SQL files
//
//go:embed sql/schema/base_schema.sql
var baseSchemaSQL string

//go:embed sql/queries/check_circular_reference.sql
var checkCircularReferenceSQL string

//go:embed sql/queries/delete_cascade.sql
var deleteCascadeSQL string

// store implements the Store interface with dynamic dimension configuration
type store struct {
	db           *sql.DB
	config       Config
	queryBuilder *queryBuilder
	sqlBuilder   *sqlBuilder
	idEngine     *IDEngine
	// Query planning components
	queryAnalyzer  *QueryAnalyzer
	queryOptimizer *QueryOptimizer
	sqlGenerator   *QuerySQLGenerator
}

// newConfigurableStore creates a new store instance with custom dimension configuration
func newConfigurableStore(dbPath string, config Config) (Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure SQLite for better concurrency with modernc.org/sqlite
	// Set busy timeout first to help with concurrent access during initialization
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Configure other pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",   // Write-Ahead Logging for better concurrency
		"PRAGMA synchronous = NORMAL", // Balance between safety and performance
		"PRAGMA cache_size = -2000",   // 2MB cache
		"PRAGMA temp_store = MEMORY",  // Use memory for temp tables
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			// For WAL mode, ignore "database is locked" errors on secondary connections
			// as the first connection will have already set it
			if pragma == "PRAGMA journal_mode = WAL" && strings.Contains(err.Error(), "database is locked") {
				continue
			}
			_ = db.Close()
			return nil, fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	// Configure connection pool for modernc.org/sqlite
	db.SetMaxOpenConns(1)    // Single writer connection for SQLite
	db.SetMaxIdleConns(1)    // Keep connection alive
	db.SetConnMaxLifetime(0) // Don't close connections automatically

	s := &store{
		db:           db,
		config:       config,
		queryBuilder: newQueryBuilder(config),
		sqlBuilder:   newSQLBuilder(),
		idEngine:     NewIDEngine(config, db),
		// Initialize query planning components
		queryAnalyzer:  NewQueryAnalyzer(config),
		queryOptimizer: NewQueryOptimizer(config),
	}

	// Initialize sqlGenerator with idEngine
	s.sqlGenerator = NewQuerySQLGenerator(config, s.idEngine)

	// Run base migrations
	if err := s.migrateBase(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run base migrations: %w", err)
	}

	// Apply dimension-specific schema
	if err := s.applyDimensionSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to apply dimension schema: %w", err)
	}

	return s, nil
}

// Close releases database resources
func (s *store) Close() error {
	return s.db.Close()
}

// migrateBase runs the core schema migrations
func (s *store) migrateBase() error {
	// Execute base schema from embedded SQL file
	if _, err := s.db.Exec(baseSchemaSQL); err != nil {
		return fmt.Errorf("failed to create base schema: %w", err)
	}

	return nil
}

// applyDimensionSchema adds dimension-specific columns and indexes
func (s *store) applyDimensionSchema() error {
	schemaBuilder := newSchemaBuilder(s.config)

	// Generate and execute dimension columns
	for _, ddl := range schemaBuilder.generateDimensionColumns() {
		if _, err := s.db.Exec(ddl); err != nil {
			// Column might already exist, which is fine
			if !isColumnExistsError(err) {
				return fmt.Errorf("failed to add dimension column: %w", err)
			}
		}
	}

	// Generate and execute indexes
	for _, ddl := range schemaBuilder.generateIndexes() {
		if _, err := s.db.Exec(ddl); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// List returns documents with dynamically generated IDs
func (s *store) List(opts ListOptions) ([]Document, error) {
	// Phase 1: Analyze - Convert ListOptions to QueryPlan
	plan, err := s.queryAnalyzer.Analyze(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}

	// Phase 2: Optimize - Apply optimizations to the plan
	plan = s.queryOptimizer.Optimize(plan)

	// Phase 3: Generate SQL from the plan
	query, args, err := s.sqlGenerator.GenerateSQL(plan)
	if err != nil {
		// Fall back to legacy query generation for unsupported features
		if plan.Type == HierarchicalQuery {
			filters := s.convertListOptionsToFilters(opts)
			query, args, err = s.queryBuilder.GenerateListQuery(filters)
			if err != nil {
				return nil, fmt.Errorf("failed to generate query: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to generate SQL: %w", err)
		}
	}

	// Phase 4: Execute the query
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Phase 5: Scan results
	var results []Document
	for rows.Next() {
		doc, err := s.scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// Add creates a new document with dimension values
func (s *store) Add(title string, dimensions map[string]interface{}) (string, error) {
	// Merge provided dimensions with defaults
	dimensionValues := make(map[string]interface{})

	// Copy provided dimensions
	for k, v := range dimensions {
		dimensionValues[k] = v
	}

	// Set default values for enumerated dimensions not provided
	for _, dim := range s.config.Dimensions {
		if dim.Type == Enumerated {
			// Check if user provided a value (either by dimension name or for hierarchical ref field)
			provided := false
			var val interface{}

			if v, ok := dimensionValues[dim.Name]; ok {
				provided = true
				val = v
			}

			if provided {
				// Validate the provided value
				strVal := fmt.Sprintf("%v", val)
				validValue := false
				for _, allowedVal := range dim.Values {
					if strVal == allowedVal {
						validValue = true
						break
					}
				}
				if !validValue {
					return "", fmt.Errorf("invalid value '%s' for dimension '%s'", strVal, dim.Name)
				}
				dimensionValues[dim.Name] = strVal
			} else {
				// Use default value
				if dim.DefaultValue != "" {
					dimensionValues[dim.Name] = dim.DefaultValue
				} else if len(dim.Values) > 0 {
					dimensionValues[dim.Name] = dim.Values[0]
				}
			}
		}
	}

	// Handle hierarchical dimension - support smart ID detection for parent references
	hierDim := s.findHierarchicalDimension()
	if hierDim != nil {
		// Support smart ID detection for parent references
		if parentID, ok := dimensionValues[hierDim.RefField]; ok && parentID != nil && parentID != "" {
			parentIDStr := fmt.Sprintf("%v", parentID)
			resolvedUUID, err := s.resolveIDToUUID(parentIDStr)
			if err != nil {
				return "", fmt.Errorf("invalid parent ID '%s': %w", parentIDStr, err)
			}
			dimensionValues[hierDim.RefField] = resolvedUUID
		}
	}

	return s.addWithDimensions(title, dimensionValues)
}

// addWithDimensions creates a new document with specific dimension values
func (s *store) addWithDimensions(title string, dimensionValues map[string]interface{}) (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	id := uuid.New().String()
	now := time.Now().Unix()

	// Build dynamic INSERT statement
	columns := []string{"uuid", "title", "body", "created_at", "updated_at"}
	values := []interface{}{id, title, "", now, now}

	// Add dimension columns
	for _, dim := range s.config.Dimensions {
		if val, exists := dimensionValues[dim.Name]; exists {
			columns = append(columns, dim.Name)
			values = append(values, val)
		} else if dim.Type == Hierarchical {
			// For hierarchical dimensions, check RefField
			if val, exists := dimensionValues[dim.RefField]; exists {
				columns = append(columns, dim.RefField)
				values = append(values, val)
			}
		}
	}

	// Use SQL builder for safe query construction
	query, args, err := s.sqlBuilder.buildInsert("documents", columns, values)
	if err != nil {
		return "", fmt.Errorf("failed to build insert query: %w", err)
	}

	_, err = tx.Exec(query, args...)
	if err != nil {
		return "", fmt.Errorf("failed to insert document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return id, nil
}

// Update modifies an existing document
func (s *store) Update(id string, updates UpdateRequest) error {
	// Resolve ID to UUID (handles both UUIDs and user-facing IDs)
	actualUUID, err := s.resolveIDToUUID(id)
	if err != nil {
		return err
	}

	// Pre-resolve parent IDs before starting transaction
	hierDim := s.findHierarchicalDimension()
	if hierDim != nil && updates.Dimensions != nil {
		if parentValue, hasParent := updates.Dimensions[hierDim.RefField]; hasParent && parentValue != nil && parentValue != "" {
			parentStr := fmt.Sprintf("%v", parentValue)
			if !s.idEngine.IsUUIDFormat(parentStr) {
				// Resolve user-facing ID to UUID before transaction
				resolvedUUID, err := s.resolveIDToUUID(parentStr)
				if err != nil {
					return fmt.Errorf("invalid parent ID '%s': %w", parentStr, err)
				}
				updates.Dimensions[hierDim.RefField] = resolvedUUID
			}
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check for circular references if updating parent through hierarchical dimension
	if hierDim != nil && updates.Dimensions != nil {
		if parentValue, hasParent := updates.Dimensions[hierDim.RefField]; hasParent {
			parentStr := fmt.Sprintf("%v", parentValue)
			if parentStr != "" {
				if parentStr == actualUUID {
					return fmt.Errorf("cannot set document as its own parent")
				}

				// Check for circular references using embedded SQL
				query := fmt.Sprintf(checkCircularReferenceSQL, hierDim.RefField, hierDim.RefField, hierDim.RefField)
				var cycle int
				err = tx.QueryRow(query, parentStr, actualUUID).Scan(&cycle)
				if err != nil {
					return fmt.Errorf("failed to check for circular reference: %w", err)
				}

				if cycle > 0 {
					return fmt.Errorf("cannot set parent: would create circular reference")
				}
			}
		}
	}

	// Build dynamic UPDATE statement
	columns := []string{"updated_at"}
	values := []interface{}{time.Now().Unix()}

	if updates.Title != nil {
		columns = append(columns, "title")
		values = append(values, *updates.Title)
	}

	if updates.Body != nil {
		columns = append(columns, "body")
		values = append(values, *updates.Body)
	}

	// Handle dimension updates
	if updates.Dimensions != nil {
		for dimName, dimValue := range updates.Dimensions {
			// Convert to string
			dimValueStr := fmt.Sprintf("%v", dimValue)

			// Handle hierarchical dimensions (parent updates)
			if hierDim != nil && dimName == hierDim.RefField {
				columns = append(columns, dimName)
				if dimValueStr == "" {
					values = append(values, nil)
				} else {
					// Parent ID was already resolved before transaction started
					values = append(values, dimValueStr)
				}
				continue
			}

			// Handle enumerated dimensions
			dimFound := false
			for _, dim := range s.config.Dimensions {
				if dim.Name == dimName && dim.Type == Enumerated {
					dimFound = true
					// Validate the value
					validValue := false
					for _, allowedVal := range dim.Values {
						if dimValueStr == allowedVal {
							validValue = true
							break
						}
					}
					if !validValue {
						return fmt.Errorf("invalid value '%s' for dimension '%s'", dimValueStr, dimName)
					}
					columns = append(columns, dimName)
					values = append(values, dimValueStr)
					break
				}
			}
			if !dimFound {
				return fmt.Errorf("unknown dimension '%s'", dimName)
			}
		}
	}

	// Use SQL builder for safe query construction
	query, args, err := s.sqlBuilder.buildDynamicUpdate(columns, values, actualUUID)
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("document not found: %s", id)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ResolveUUID converts a user-facing ID to a UUID
// Supports smart ID detection - returns UUID unchanged if already a UUID
func (s *store) ResolveUUID(userFacingID string) (string, error) {
	return s.idEngine.ResolveID(userFacingID)
}

// Delete removes a document and optionally its children
func (s *store) Delete(id string, cascade bool) error {
	// Resolve ID to UUID (handles both UUIDs and user-facing IDs)
	actualUUID, err := s.resolveIDToUUID(id)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if cascade {
		hierDim := s.findHierarchicalDimension()
		if hierDim != nil {
			// Use recursive CTE to delete all descendants from embedded SQL
			query := fmt.Sprintf(deleteCascadeSQL, hierDim.RefField)

			result, err := tx.Exec(query, actualUUID)
			if err != nil {
				return fmt.Errorf("failed to delete with cascade: %w", err)
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to get rows affected: %w", err)
			}

			if rowsAffected == 0 {
				return fmt.Errorf("document not found: %s", id)
			}
		} else {
			// No hierarchy - just delete the single document
			result, err := tx.Exec("DELETE FROM documents WHERE uuid = ?", actualUUID)
			if err != nil {
				return fmt.Errorf("failed to delete document: %w", err)
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to get rows affected: %w", err)
			}

			if rowsAffected == 0 {
				return fmt.Errorf("document not found: %s", id)
			}
		}
	} else {
		// Check if document has children first
		hierDim := s.findHierarchicalDimension()
		if hierDim != nil {
			var hasChildren int
			// Use SQL builder for count query
			query, args, err := s.sqlBuilder.buildSelectCount("documents", squirrel.Eq{hierDim.RefField: actualUUID})
			if err != nil {
				return fmt.Errorf("failed to build count query: %w", err)
			}
			err = tx.QueryRow(query, args...).Scan(&hasChildren)
			if err != nil {
				return fmt.Errorf("failed to check for children: %w", err)
			}

			if hasChildren > 0 {
				return fmt.Errorf("cannot delete document with children unless cascade is true")
			}
		}

		// Delete single document
		result, err := tx.Exec("DELETE FROM documents WHERE uuid = ?", actualUUID)
		if err != nil {
			return fmt.Errorf("failed to delete document: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("document not found: %s", id)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteByDimension removes all documents matching dimension filters
func (s *store) DeleteByDimension(filters map[string]interface{}) (int, error) {
	if len(filters) == 0 {
		return 0, fmt.Errorf("no filters provided")
	}

	// Validate dimensions and values
	conditions := squirrel.Eq{}
	for dimension, value := range filters {
		// Validate that the dimension exists in the configuration
		dimensionExists := false
		for _, dim := range s.config.Dimensions {
			if dim.Name == dimension {
				dimensionExists = true
				// For enumerated dimensions, validate the value
				if dim.Type == Enumerated {
					strValue := fmt.Sprintf("%v", value)
					valueValid := false
					for _, v := range dim.Values {
						if v == strValue {
							valueValid = true
							break
						}
					}
					if !valueValid {
						return 0, fmt.Errorf("invalid value '%s' for dimension '%s'", strValue, dimension)
					}
				}
				break
			}
		}

		if !dimensionExists {
			return 0, fmt.Errorf("dimension '%s' not found in configuration", dimension)
		}

		conditions[dimension] = value
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Build query using SQL builder
	query, args, err := s.sqlBuilder.buildDelete("documents", conditions)
	if err != nil {
		return 0, fmt.Errorf("failed to build delete query: %w", err)
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete documents: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(rowsAffected), nil
}

// DeleteWhere removes all documents matching a custom WHERE clause
func (s *store) DeleteWhere(whereClause string, args ...interface{}) (int, error) {
	if whereClause == "" {
		return 0, fmt.Errorf("where clause cannot be empty")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Build the DELETE query using SQL builder
	query, sqlArgs, err := s.sqlBuilder.buildDeleteWhere("documents", whereClause, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to build delete query: %w", err)
	}

	result, err := tx.Exec(query, sqlArgs...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete documents with where clause '%s': %w", whereClause, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(rowsAffected), nil
}

// UpdateByDimension updates all documents matching dimension filters
func (s *store) UpdateByDimension(filters map[string]interface{}, updates UpdateRequest) (int, error) {
	if len(filters) == 0 {
		return 0, fmt.Errorf("no filters provided")
	}

	// Validate dimensions and values
	conditions := squirrel.Eq{}
	for dimension, value := range filters {
		// Validate that the dimension exists in the configuration
		dimensionExists := false
		for _, dim := range s.config.Dimensions {
			if dim.Name == dimension {
				dimensionExists = true
				// For enumerated dimensions, validate the value
				if dim.Type == Enumerated {
					strValue := fmt.Sprintf("%v", value)
					valueValid := false
					for _, v := range dim.Values {
						if v == strValue {
							valueValid = true
							break
						}
					}
					if !valueValid {
						return 0, fmt.Errorf("invalid value '%s' for dimension '%s'", strValue, dimension)
					}
				}
				break
			}
		}

		if !dimensionExists {
			return 0, fmt.Errorf("dimension '%s' not found in configuration", dimension)
		}

		conditions[dimension] = value
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Build the columns and values for the update
	columns, values, err := s.buildUpdateColumnsAndValues(updates)
	if err != nil {
		return 0, err
	}

	// Always add updated_at
	columns = append(columns, "updated_at")
	values = append(values, time.Now().Unix())

	// Build query using SQL builder
	query, args, err := s.sqlBuilder.buildUpdateByCondition("documents", columns, values, conditions)
	if err != nil {
		return 0, fmt.Errorf("failed to build update query: %w", err)
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to update documents: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(rowsAffected), nil
}

// UpdateWhere updates all documents matching a custom WHERE clause
func (s *store) UpdateWhere(whereClause string, updates UpdateRequest, args ...interface{}) (int, error) {
	if whereClause == "" {
		return 0, fmt.Errorf("where clause cannot be empty")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Build the columns and values for the update
	columns, values, err := s.buildUpdateColumnsAndValues(updates)
	if err != nil {
		return 0, err
	}

	// Always add updated_at
	columns = append(columns, "updated_at")
	values = append(values, time.Now().Unix())

	// Build the UPDATE query using SQL builder
	query, sqlArgs, err := s.sqlBuilder.buildUpdateWhere("documents", columns, values, whereClause, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to build update query: %w", err)
	}

	result, err := tx.Exec(query, sqlArgs...)
	if err != nil {
		return 0, fmt.Errorf("failed to update documents with where clause '%s': %w", whereClause, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(rowsAffected), nil
}

// Helper methods

// buildUpdateColumnsAndValues prepares columns and values for bulk update operations
func (s *store) buildUpdateColumnsAndValues(updates UpdateRequest) ([]string, []interface{}, error) {
	var columns []string
	var values []interface{}

	if updates.Title != nil {
		columns = append(columns, "title")
		values = append(values, *updates.Title)
	}

	if updates.Body != nil {
		columns = append(columns, "body")
		values = append(values, *updates.Body)
	}

	// Pre-resolve parent IDs if needed
	hierDim := s.findHierarchicalDimension()
	if hierDim != nil && updates.Dimensions != nil {
		if parentValue, hasParent := updates.Dimensions[hierDim.RefField]; hasParent && parentValue != nil && parentValue != "" {
			parentStr := fmt.Sprintf("%v", parentValue)
			if !s.idEngine.IsUUIDFormat(parentStr) {
				// Resolve user-facing ID to UUID
				resolvedUUID, err := s.resolveIDToUUID(parentStr)
				if err != nil {
					return nil, nil, fmt.Errorf("invalid parent ID '%s': %w", parentStr, err)
				}
				updates.Dimensions[hierDim.RefField] = resolvedUUID
			}
		}
	}

	// Handle dimension updates
	if updates.Dimensions != nil {
		for dimName, dimValue := range updates.Dimensions {
			// Convert to string
			dimValueStr := fmt.Sprintf("%v", dimValue)

			// Handle hierarchical dimensions (parent updates)
			if hierDim != nil && dimName == hierDim.RefField {
				columns = append(columns, dimName)
				if dimValueStr == "" {
					values = append(values, nil)
				} else {
					// Parent ID was already resolved above
					values = append(values, dimValueStr)
				}
				continue
			}

			// Handle enumerated dimensions
			dimFound := false
			for _, dim := range s.config.Dimensions {
				if dim.Name == dimName && dim.Type == Enumerated {
					dimFound = true
					// Validate the value
					validValue := false
					for _, v := range dim.Values {
						if v == dimValueStr {
							validValue = true
							break
						}
					}
					if !validValue {
						return nil, nil, fmt.Errorf("invalid value '%s' for dimension '%s'", dimValueStr, dimName)
					}
					columns = append(columns, dimName)
					values = append(values, dimValueStr)
					break
				}
			}

			if !dimFound {
				return nil, nil, fmt.Errorf("dimension '%s' not found in configuration", dimName)
			}
		}
	}

	if len(columns) == 0 {
		return nil, nil, fmt.Errorf("no fields to update")
	}

	return columns, values, nil
}

func (s *store) findHierarchicalDimension() *DimensionConfig {
	for _, dim := range s.config.Dimensions {
		if dim.Type == Hierarchical {
			return &dim
		}
	}
	return nil
}

func (s *store) convertListOptionsToFilters(opts ListOptions) map[string]interface{} {
	filters := make(map[string]interface{})

	// Use the new generic Filters map
	if opts.Filters != nil {
		for key, value := range opts.Filters {
			filters[key] = value
		}
	}

	// Convert search filter
	if opts.FilterBySearch != "" {
		filters["search"] = opts.FilterBySearch
	}

	return filters
}

func (s *store) scanDocument(rows *sql.Rows) (Document, error) {
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return Document{}, err
	}

	// Create a slice to hold the values
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan the row
	if err := rows.Scan(valuePtrs...); err != nil {
		return Document{}, err
	}

	// Build document
	doc := Document{
		Dimensions: make(map[string]interface{}),
	}

	// Map values to document fields
	for i, col := range columns {
		switch col {
		case "uuid":
			doc.UUID = values[i].(string)
		case "user_facing_id":
			doc.UserFacingID = values[i].(string)
		case "title":
			doc.Title = values[i].(string)
		case "body":
			doc.Body = values[i].(string)
		case "created_at":
			doc.CreatedAt = time.Unix(values[i].(int64), 0)
		case "updated_at":
			doc.UpdatedAt = time.Unix(values[i].(int64), 0)
		default:
			// Handle all dimension columns
			dimensionFound := false

			// Check enumerated dimensions
			for _, dim := range s.config.Dimensions {
				if dim.Name == col && dim.Type == Enumerated {
					if values[i] != nil {
						doc.Dimensions[col] = values[i].(string)
					}
					dimensionFound = true
					break
				}
			}

			// Check hierarchical dimension
			if !dimensionFound {
				hierDim := s.findHierarchicalDimension()
				if hierDim != nil && col == hierDim.RefField {
					if values[i] != nil {
						doc.Dimensions[col] = values[i].(string)
					}
					dimensionFound = true
				}
			}
		}
	}

	return doc, nil
}

func isColumnExistsError(err error) bool {
	// SQLite returns "duplicate column name" when column already exists
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}

// Note: normalizeUserFacingID has been moved to IDEngine

// Note: normalizeLevelID has been moved to IDEngine

// Note: resolveUUIDFlat has been moved to IDEngine

// Note: resolveUUIDHierarchical has been moved to IDEngine

// Note: resolveChildUUID has been moved to IDEngine

// Note: isUUIDFormat has been moved to IDEngine

// resolveIDToUUID resolves either a UUID or user-facing ID to a UUID
func (s *store) resolveIDToUUID(id string) (string, error) {
	if s.idEngine.IsUUIDFormat(id) {
		return id, nil
	}

	// Not a UUID, try to resolve as user-facing ID
	uuid, err := s.ResolveUUID(id)
	if err != nil {
		return "", fmt.Errorf("invalid ID '%s': %w", id, err)
	}

	return uuid, nil
}
