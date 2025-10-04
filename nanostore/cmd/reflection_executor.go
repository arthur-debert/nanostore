package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
	"github.com/arthur-debert/nanostore/types"
)

// ReflectionExecutor handles actual Store method invocation using reflection
type ReflectionExecutor struct {
	registry *EnhancedTypeRegistry
}

// NewReflectionExecutor creates a new reflection-based method executor
func NewReflectionExecutor(registry *EnhancedTypeRegistry) *ReflectionExecutor {
	return &ReflectionExecutor{registry: registry}
}

// CreateStoreInstance creates an actual Store instance using reflection
func (re *ReflectionExecutor) CreateStoreInstance(typeName, dbPath string) (interface{}, error) {
	_, exists := re.registry.GetTypeDefinition(typeName)
	if !exists {
		return nil, fmt.Errorf("type %s not registered", typeName)
	}

	// Use reflection to call api.New with the dynamic type
	// This is complex because we need to call a generic function with a runtime type
	// For MVP, we'll create stores for our built-in types directly

	switch typeName {
	case "Task":
		return re.createTaskStore(dbPath)
	case "Note":
		return re.createNoteStore(dbPath)
	default:
		return nil, fmt.Errorf("dynamic store creation not yet implemented for type %s", typeName)
	}
}

// createTaskStore creates a Task-specific store
func (re *ReflectionExecutor) createTaskStore(dbPath string) (*api.Store[TaskDocument], error) {
	return api.New[TaskDocument](dbPath)
}

// createNoteStore creates a Note-specific store
func (re *ReflectionExecutor) createNoteStore(dbPath string) (*api.Store[NoteDocument], error) {
	return api.New[NoteDocument](dbPath)
}

// TaskDocument represents a Task document type for actual store operations
type TaskDocument struct {
	nanostore.Document
	Status      string `values:"pending,active,done" default:"pending"`
	Priority    string `values:"low,medium,high" default:"medium"`
	ParentID    string `dimension:"parent_id,ref"`
	Description string
	Assignee    string
	DueDate     *time.Time
}

// NoteDocument represents a Note document type for actual store operations
type NoteDocument struct {
	nanostore.Document
	Category string `values:"personal,work,idea,reference" default:"personal"`
	Tags     string
	Content  string
}

// ExecuteMethod executes the specified method on a Store instance
func (re *ReflectionExecutor) ExecuteMethod(typeName, methodName string, args []interface{}) (interface{}, error) {
	// Get the store instance
	storeInterface, err := re.CreateStoreInstance(typeName, args[0].(string)) // dbPath is first arg
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Get the reflect.Value of the store
	storeValue := reflect.ValueOf(storeInterface)
	storeType := storeValue.Type()

	// Find the method
	method := storeValue.MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found on store type %s", methodName, storeType)
	}

	// Convert arguments to proper types
	methodType := method.Type()
	reflectArgs := make([]reflect.Value, 0, len(args)-1) // Skip dbPath

	for i := 1; i < len(args); i++ { // Skip dbPath (first arg)
		if i-1 >= methodType.NumIn() {
			break // Skip extra args
		}

		expectedType := methodType.In(i - 1)
		convertedArg, err := re.convertArgument(args[i], expectedType)
		if err != nil {
			return nil, fmt.Errorf("failed to convert argument %d: %w", i, err)
		}
		reflectArgs = append(reflectArgs, convertedArg)
	}

	// Call the method
	results := method.Call(reflectArgs)

	// Convert results back to interface{}
	return re.convertResults(results)
}

// convertArgument converts an interface{} to the expected reflect.Type
func (re *ReflectionExecutor) convertArgument(arg interface{}, expectedType reflect.Type) (reflect.Value, error) {
	if arg == nil {
		return reflect.Zero(expectedType), nil
	}

	argValue := reflect.ValueOf(arg)

	// If types match directly, return as-is
	if argValue.Type().AssignableTo(expectedType) {
		return argValue, nil
	}

	// Handle string conversions
	if argValue.Kind() == reflect.String {
		return re.convertFromString(argValue.String(), expectedType)
	}

	// Handle map[string]interface{} to struct conversion
	if argValue.Kind() == reflect.Map && expectedType.Kind() == reflect.Ptr && expectedType.Elem().Kind() == reflect.Struct {
		return re.convertMapToStruct(arg.(map[string]interface{}), expectedType)
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %T to %s", arg, expectedType)
}

// convertFromString converts a string to the expected type
func (re *ReflectionExecutor) convertFromString(s string, expectedType reflect.Type) (reflect.Value, error) {
	switch expectedType.Kind() {
	case reflect.String:
		return reflect.ValueOf(s), nil
	case reflect.Int:
		i, err := strconv.Atoi(s)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(i), nil
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(b), nil
	case reflect.Slice:
		if expectedType.Elem().Kind() == reflect.String {
			slice := strings.Split(s, ",")
			return reflect.ValueOf(slice), nil
		}
	}

	return reflect.Value{}, fmt.Errorf("cannot convert string %q to %s", s, expectedType)
}

// convertMapToStruct converts a map[string]interface{} to a struct pointer
func (re *ReflectionExecutor) convertMapToStruct(data map[string]interface{}, expectedType reflect.Type) (reflect.Value, error) {
	if expectedType.Kind() != reflect.Ptr || expectedType.Elem().Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("expected pointer to struct, got %s", expectedType)
	}

	// Create new instance of the struct
	structType := expectedType.Elem()
	structPtr := reflect.New(structType)
	structValue := structPtr.Elem()

	// Set fields from map
	for key, value := range data {
		field := structValue.FieldByName(re.toPascalCase(key))
		if field.IsValid() && field.CanSet() {
			fieldValue, err := re.convertArgument(value, field.Type())
			if err != nil {
				continue // Skip fields that can't be converted
			}
			field.Set(fieldValue)
		}
	}

	return structPtr, nil
}

// convertResults converts method results to interface{}
func (re *ReflectionExecutor) convertResults(results []reflect.Value) (interface{}, error) {
	if len(results) == 0 {
		return nil, nil
	}

	// Handle error return
	if len(results) >= 2 {
		errValue := results[len(results)-1]
		if !errValue.IsNil() {
			return nil, errValue.Interface().(error)
		}
	}

	// Return first non-error result
	if len(results) >= 1 {
		result := results[0]
		if result.IsValid() {
			return result.Interface(), nil
		}
	}

	return nil, nil
}

// ExecuteCreate executes a Create method with proper type conversion
func (re *ReflectionExecutor) ExecuteCreate(typeName, dbPath, title string, data map[string]interface{}) (interface{}, error) {
	switch typeName {
	case "Task":
		store, err := re.createTaskStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		task := &TaskDocument{}
		task.Title = title
		re.populateDocumentFromMap(task, data)

		return store.Create(title, task)

	case "Note":
		store, err := re.createNoteStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		note := &NoteDocument{}
		note.Title = title
		re.populateDocumentFromMap(note, data)

		return store.Create(title, note)

	default:
		return nil, fmt.Errorf("create not implemented for type %s", typeName)
	}
}

// ExecuteGet executes a Get method
func (re *ReflectionExecutor) ExecuteGet(typeName, dbPath, id string) (interface{}, error) {
	switch typeName {
	case "Task":
		store, err := re.createTaskStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		return store.Get(id)

	case "Note":
		store, err := re.createNoteStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		return store.Get(id)

	default:
		return nil, fmt.Errorf("get not implemented for type %s", typeName)
	}
}

// ExecuteUpdate executes an Update method
func (re *ReflectionExecutor) ExecuteUpdate(typeName, dbPath, id string, data map[string]interface{}) (interface{}, error) {
	switch typeName {
	case "Task":
		store, err := re.createTaskStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		task := &TaskDocument{}
		re.populateDocumentFromMap(task, data)

		return store.Update(id, task)

	case "Note":
		store, err := re.createNoteStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		note := &NoteDocument{}
		re.populateDocumentFromMap(note, data)

		return store.Update(id, note)

	default:
		return nil, fmt.Errorf("update not implemented for type %s", typeName)
	}
}

// ExecuteDelete executes a Delete method
func (re *ReflectionExecutor) ExecuteDelete(typeName, dbPath, id string, cascade bool) error {
	switch typeName {
	case "Task":
		store, err := re.createTaskStore(dbPath)
		if err != nil {
			return err
		}
		defer func() { _ = store.Close() }()

		return store.Delete(id, cascade)

	case "Note":
		store, err := re.createNoteStore(dbPath)
		if err != nil {
			return err
		}
		defer func() { _ = store.Close() }()

		return store.Delete(id, cascade)

	default:
		return fmt.Errorf("delete not implemented for type %s", typeName)
	}
}

// ExecuteList executes a List method
func (re *ReflectionExecutor) ExecuteList(typeName, dbPath string, options types.ListOptions) (interface{}, error) {
	switch typeName {
	case "Task":
		store, err := re.createTaskStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		return store.List(options)

	case "Note":
		store, err := re.createNoteStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		return store.List(options)

	default:
		return nil, fmt.Errorf("list not implemented for type %s", typeName)
	}
}

// ExecuteQuery executes a WHERE clause query using the Query API
func (re *ReflectionExecutor) ExecuteQuery(typeName, dbPath, whereClause string, whereArgs []interface{}, orderBy string, limit, offset int) (interface{}, error) {
	switch typeName {
	case "Task":
		store, err := re.createTaskStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		query := store.Query()

		// Add WHERE clause if provided
		if whereClause != "" {
			query = query.Where(whereClause, whereArgs...)
		}

		// Add ORDER BY if provided
		if orderBy != "" {
			query = query.OrderBy(orderBy)
		}

		// Add LIMIT if provided
		if limit > 0 {
			query = query.Limit(limit)
		}

		// Add OFFSET if provided
		if offset > 0 {
			query = query.Offset(offset)
		}

		return query.Find()

	case "Note":
		store, err := re.createNoteStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		query := store.Query()

		// Add WHERE clause if provided
		if whereClause != "" {
			query = query.Where(whereClause, whereArgs...)
		}

		// Add ORDER BY if provided
		if orderBy != "" {
			query = query.OrderBy(orderBy)
		}

		// Add LIMIT if provided
		if limit > 0 {
			query = query.Limit(limit)
		}

		// Add OFFSET if provided
		if offset > 0 {
			query = query.Offset(offset)
		}

		return query.Find()

	default:
		return nil, fmt.Errorf("query not implemented for type %s", typeName)
	}
}

// populateDocumentFromMap populates a document struct from a map
func (re *ReflectionExecutor) populateDocumentFromMap(doc interface{}, data map[string]interface{}) {
	docValue := reflect.ValueOf(doc).Elem()

	for key, value := range data {
		if key == "title" || value == nil || value == "" {
			continue // Skip empty values and title (handled separately)
		}

		fieldName := re.toPascalCase(key)
		field := docValue.FieldByName(fieldName)

		if !field.IsValid() || !field.CanSet() {
			continue
		}

		re.setFieldValue(field, value)
	}
}

// setFieldValue sets a field value with proper type conversion
func (re *ReflectionExecutor) setFieldValue(field reflect.Value, value interface{}) {
	if value == nil {
		return
	}

	valueStr := fmt.Sprintf("%v", value)
	if valueStr == "" {
		return
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(valueStr)
	case reflect.Int, reflect.Int64:
		if i, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			field.SetInt(i)
		}
	case reflect.Bool:
		if b, err := strconv.ParseBool(valueStr); err == nil {
			field.SetBool(b)
		}
	case reflect.Ptr:
		if field.Type().Elem().Kind() == reflect.Struct {
			// Handle *time.Time
			if field.Type() == reflect.TypeOf((*time.Time)(nil)) {
				if t, err := time.Parse(time.RFC3339, valueStr); err == nil {
					field.Set(reflect.ValueOf(&t))
				}
			}
		}
	}
}

// toPascalCase converts snake_case to PascalCase
func (re *ReflectionExecutor) toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// parseListOptions converts CLI parameters to types.ListOptions
func (re *ReflectionExecutor) parseListOptions(filters []string, sort string, limit, offset int) types.ListOptions {
	options := types.ListOptions{
		Filters: make(map[string]interface{}),
	}

	// Set limit and offset as pointers
	if limit > 0 {
		options.Limit = &limit
	}
	if offset > 0 {
		options.Offset = &offset
	}

	if sort != "" {
		options.OrderBy = []types.OrderClause{
			{Column: sort, Descending: false},
		}
	}

	// Parse filters (format: "key=value")
	for _, filter := range filters {
		parts := strings.SplitN(filter, "=", 2)
		if len(parts) == 2 {
			options.Filters[parts[0]] = parts[1]
		}
	}

	return options
}

// buildFilterWhere builds WHERE clauses from all filter types
func (re *ReflectionExecutor) buildFilterWhere(
	createdAfter, createdBefore, updatedAfter, updatedBefore string,
	nullFields, notNullFields []string,
	searchText, titleContains, bodyContains string, caseSensitive bool,
	filterEq, filterNe, filterGt, filterLt, filterGte, filterLte, filterLike, filterIn []string,
	status, priority string, statusIn, priorityIn []string) (string, []interface{}, error) {
	var clauses []string
	var args []interface{}

	// Handle date range filters
	if createdAfter != "" {
		t, err := time.Parse(time.RFC3339, createdAfter)
		if err != nil {
			return "", nil, fmt.Errorf("invalid created-after date format: %w", err)
		}
		clauses = append(clauses, "created_at > ?")
		args = append(args, t)
	}

	if createdBefore != "" {
		t, err := time.Parse(time.RFC3339, createdBefore)
		if err != nil {
			return "", nil, fmt.Errorf("invalid created-before date format: %w", err)
		}
		clauses = append(clauses, "created_at < ?")
		args = append(args, t)
	}

	if updatedAfter != "" {
		t, err := time.Parse(time.RFC3339, updatedAfter)
		if err != nil {
			return "", nil, fmt.Errorf("invalid updated-after date format: %w", err)
		}
		clauses = append(clauses, "updated_at > ?")
		args = append(args, t)
	}

	if updatedBefore != "" {
		t, err := time.Parse(time.RFC3339, updatedBefore)
		if err != nil {
			return "", nil, fmt.Errorf("invalid updated-before date format: %w", err)
		}
		clauses = append(clauses, "updated_at < ?")
		args = append(args, t)
	}

	// Handle NULL field filters using simple parameter-based approach
	// Instead of complex IS NULL syntax, use a special marker value
	for _, field := range nullFields {
		// Use a simple equality check with special NULL marker
		fieldName := field
		if !strings.Contains(field, ".") {
			fieldName = fmt.Sprintf("_data.%s", field)
		}
		clauses = append(clauses, fmt.Sprintf("%s = ?", fieldName))
		args = append(args, "__NULL_CHECK__")
	}

	// Handle NOT NULL field filters
	for _, field := range notNullFields {
		// Use a simple inequality check with special NULL marker
		fieldName := field
		if !strings.Contains(field, ".") {
			fieldName = fmt.Sprintf("_data.%s", field)
		}
		clauses = append(clauses, fmt.Sprintf("%s != ?", fieldName))
		args = append(args, "__NULL_CHECK__")
	}

	// Handle text search filters using LIKE operator
	if searchText != "" {
		// Create pattern with wildcards
		pattern := "%" + searchText + "%"
		// For case-insensitive search, convert to lowercase (WhereEvaluator will handle case)
		if !caseSensitive {
			pattern = strings.ToLower(pattern)
		}
		// Use special search marker for title OR body matching
		clauses = append(clauses, "__SEARCH_TITLE_OR_BODY__ LIKE ?")
		args = append(args, pattern)
	}

	if titleContains != "" {
		pattern := "%" + titleContains + "%"
		if !caseSensitive {
			pattern = strings.ToLower(pattern)
		}
		clauses = append(clauses, "title LIKE ?")
		args = append(args, pattern)
	}

	if bodyContains != "" {
		pattern := "%" + bodyContains + "%"
		if !caseSensitive {
			pattern = strings.ToLower(pattern)
		}
		clauses = append(clauses, "body LIKE ?")
		args = append(args, pattern)
	}

	// Handle enhanced filter flags with operators
	filterMap := map[string][]string{
		"=":    filterEq,
		"!=":   filterNe,
		">":    filterGt,
		"<":    filterLt,
		">=":   filterGte,
		"<=":   filterLte,
		"LIKE": filterLike,
	}

	for operator, filters := range filterMap {
		for _, filter := range filters {
			field, value, err := re.parseFieldValue(filter)
			if err != nil {
				return "", nil, fmt.Errorf("invalid filter format '%s': %w", filter, err)
			}
			clauses = append(clauses, fmt.Sprintf("%s %s ?", field, operator))
			args = append(args, value)
		}
	}

	// Handle IN filters (convert to multiple equality conditions)
	// Since WhereEvaluator doesn't support IN operator, we create multiple equality clauses
	// Note: This will use AND logic, so IN filters work as "must match all values" instead of "match any value"
	// This is a limitation - for true OR logic, we'd need to enhance WhereEvaluator
	for _, filter := range filterIn {
		field, valueList, err := re.parseFieldValue(filter)
		if err != nil {
			return "", nil, fmt.Errorf("invalid filter-in format '%s': %w", filter, err)
		}
		values := strings.Split(valueList, ",")
		// For now, we'll skip IN filters as they require OR logic which isn't supported
		// TODO: Enhance WhereEvaluator to support OR logic for proper IN filter implementation
		_ = field
		_ = values
		// Skip IN filters for now
		continue
	}

	// Handle convenience flags
	if status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, status)
	}
	if priority != "" {
		clauses = append(clauses, "priority = ?")
		args = append(args, priority)
	}
	// Handle statusIn and priorityIn convenience flags
	// Skip these for now since WhereEvaluator doesn't support IN operator
	// TODO: Implement OR logic support in WhereEvaluator for proper IN functionality
	if len(statusIn) > 0 {
		// Skip statusIn for now - requires OR logic
		_ = statusIn
	}
	if len(priorityIn) > 0 {
		// Skip priorityIn for now - requires OR logic
		_ = priorityIn
	}

	if len(clauses) == 0 {
		return "", nil, nil
	}

	whereClause := strings.Join(clauses, " AND ")
	return whereClause, args, nil
}

// parseFieldValue parses "field=value" format into field and value components
func (re *ReflectionExecutor) parseFieldValue(filter string) (string, string, error) {
	parts := strings.SplitN(filter, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected format 'field=value', got '%s'", filter)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

// combineWhereClauses combines explicit WHERE clause with date/NULL filters
func (re *ReflectionExecutor) combineWhereClauses(explicitWhere string, explicitArgs []interface{}, dateNullWhere string, dateNullArgs []interface{}) (string, []interface{}) {
	if explicitWhere == "" && dateNullWhere == "" {
		return "", nil
	}

	if explicitWhere == "" {
		return dateNullWhere, dateNullArgs
	}

	if dateNullWhere == "" {
		return explicitWhere, explicitArgs
	}

	// Combine both WHERE clauses
	combinedWhere := fmt.Sprintf("(%s) AND (%s)", explicitWhere, dateNullWhere)
	combinedArgs := append(explicitArgs, dateNullArgs...)

	return combinedWhere, combinedArgs
}
