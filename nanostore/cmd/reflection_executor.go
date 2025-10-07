package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
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
		availableTypes := re.registry.ListTypes()
		return nil, NewTypeError("create store", typeName, availableTypes)
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
		return nil, NewStoreError("invoke method",
			fmt.Errorf("method %s not found on store type %s", methodName, storeType),
			"Check method name spelling",
			"Verify method exists on Store type")
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
			return nil, NewStoreError("convert argument",
				fmt.Errorf("failed to convert argument %d: %w", i, err),
				"Check argument type and format",
				"Ensure argument matches expected type")
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

	return reflect.Value{}, NewStoreError("convert type",
		fmt.Errorf("cannot convert %T to %s", arg, expectedType),
		"Check data type compatibility",
		"Use correct format for the expected type")
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

	return reflect.Value{}, NewStoreError("convert string",
		fmt.Errorf("cannot convert string %q to %s", s, expectedType),
		"Check string format and value",
		"Ensure string can be parsed as expected type")
}

// convertMapToStruct converts a map[string]interface{} to a struct pointer
func (re *ReflectionExecutor) convertMapToStruct(data map[string]interface{}, expectedType reflect.Type) (reflect.Value, error) {
	if expectedType.Kind() != reflect.Ptr || expectedType.Elem().Kind() != reflect.Struct {
		return reflect.Value{}, NewStoreError("convert to struct",
			fmt.Errorf("expected pointer to struct, got %s", expectedType),
			"Ensure target type is a struct pointer",
			"Check document type definition")
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
	// Log the create operation
	logQuery("create", fmt.Sprintf("CREATE %s document with title: %s", typeName, title), []interface{}{data})

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
		return nil, NewTypeError("create", typeName, []string{"Task", "Note"})
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
		return nil, NewTypeError("get", typeName, []string{"Task", "Note"})
	}
}

// ExecuteGetRaw executes a GetRaw method
func (re *ReflectionExecutor) ExecuteGetRaw(typeName, dbPath, id string) (interface{}, error) {
	switch typeName {
	case "Task":
		store, err := re.createTaskStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		return store.GetRaw(id)

	case "Note":
		store, err := re.createNoteStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		return store.GetRaw(id)

	default:
		return nil, NewTypeError("get raw", typeName, []string{"Task", "Note"})
	}
}

// ExecuteGetDimensions executes a GetDimensions method
func (re *ReflectionExecutor) ExecuteGetDimensions(typeName, dbPath, id string) (interface{}, error) {
	switch typeName {
	case "Task":
		store, err := re.createTaskStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		return store.GetDimensions(id)

	case "Note":
		store, err := re.createNoteStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		return store.GetDimensions(id)

	default:
		return nil, NewTypeError("get dimensions", typeName, []string{"Task", "Note"})
	}
}

// ExecuteGetMetadata executes a GetMetadata method
func (re *ReflectionExecutor) ExecuteGetMetadata(typeName, dbPath, id string) (interface{}, error) {
	switch typeName {
	case "Task":
		store, err := re.createTaskStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		return store.GetMetadata(id)

	case "Note":
		store, err := re.createNoteStore(dbPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = store.Close() }()

		return store.GetMetadata(id)

	default:
		return nil, NewTypeError("get metadata", typeName, []string{"Task", "Note"})
	}
}

// ExecuteUpdate executes an Update method
func (re *ReflectionExecutor) ExecuteUpdate(typeName, dbPath, id string, data map[string]interface{}) (interface{}, error) {
	// Log the update operation
	logQuery("update", fmt.Sprintf("UPDATE %s document with id: %s", typeName, id), []interface{}{data})

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
		return nil, NewTypeError("update", typeName, []string{"Task", "Note"})
	}
}

// ExecuteUpdateByDimension executes UpdateByDimension on the store
func (re *ReflectionExecutor) ExecuteUpdateByDimension(typeName, dbPath string, filters map[string]interface{}, data map[string]interface{}) (interface{}, error) {
	// Log the update operation
	logQuery("update-by-dimension", fmt.Sprintf("UPDATE %s documents with filters: %v", typeName, filters), []interface{}{data})

	// Use generic ExecuteMethod to call UpdateByDimension
	args := []interface{}{dbPath, filters, data}
	return re.ExecuteMethod(typeName, "UpdateByDimension", args)
}

// ExecuteDelete executes a Delete method
func (re *ReflectionExecutor) ExecuteDelete(typeName, dbPath, id string, cascade bool) error {
	// Log the delete operation
	logQuery("delete", fmt.Sprintf("DELETE %s document with id: %s (cascade: %v)", typeName, id, cascade), nil)

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
		return NewTypeError("delete", typeName, []string{"Task", "Note"})
	}
}

// BuildWhereFromQuery translates a Query object into a SQL-like WHERE clause and arguments.
func (re *ReflectionExecutor) BuildWhereFromQuery(query *Query) (string, []interface{}) {
	if query == nil || len(query.Groups) == 0 {
		return "", nil
	}

	var finalClause strings.Builder
	var finalArgs []interface{}

	for i, group := range query.Groups {
		if len(group.Conditions) == 0 {
			continue
		}

		var groupClauses []string
		var groupArgs []interface{}

		for _, cond := range group.Conditions {
			sqlOp := operatorMap[cond.Operator]
			if sqlOp == "" {
				sqlOp = "=" // Default to equality
			}

			value := cond.Value
			// Add wildcards for LIKE operators
			if sqlOp == "LIKE" {
				switch cond.Operator {
				case "contains":
					value = "%" + value.(string) + "%"
				case "startswith":
					value = value.(string) + "%"
				case "endswith":
					value = "%" + value.(string)
				}
			}

			groupClauses = append(groupClauses, fmt.Sprintf("%s %s ?", cond.Field, sqlOp))
			groupArgs = append(groupArgs, value)
		}

		if len(groupClauses) > 0 {
			// Add the group clause, wrapped in parentheses only if there are multiple groups or multiple conditions in the group
			if finalClause.Len() > 0 {
				// Use the logical operator that connects this group to the previous one
				if i-1 < len(query.Operators) {
					finalClause.WriteString(fmt.Sprintf(" %s ", query.Operators[i-1]))
				} else {
					finalClause.WriteString(" AND ") // Default to AND if something is wrong
				}
			}

			groupClause := strings.Join(groupClauses, " AND ")
			// Only wrap in parentheses if there are multiple groups or multiple conditions in this group
			if len(query.Groups) > 1 || len(groupClauses) > 1 {
				finalClause.WriteString("(" + groupClause + ")")
			} else {
				finalClause.WriteString(groupClause)
			}
			finalArgs = append(finalArgs, groupArgs...)
		}
	}

	return finalClause.String(), finalArgs
}

// A simple map to translate from our DSL operators to SQL-like operators
var operatorMap = map[string]string{
	"eq":         "=",
	"ne":         "!=",
	"gt":         ">",
	"gte":        ">=",
	"lt":         "<",
	"lte":        "<=",
	"contains":   "LIKE",
	"startswith": "LIKE",
	"endswith":   "LIKE",
	// "in" would be more complex and is not handled here
}

// ExecuteList now uses the Query object from the context.
func (re *ReflectionExecutor) ExecuteList(typeName, dbPath string, query *Query, sort string, limit, offset int) (interface{}, error) {
	whereClause, whereArgs := re.BuildWhereFromQuery(query)

	// Log the generated query
	if whereClause != "" {
		logQuery("list", whereClause, whereArgs)
	}

	return re.ExecuteQuery(typeName, dbPath, whereClause, whereArgs, sort, limit, offset)
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
		return nil, NewTypeError("query", typeName, []string{"Task", "Note"})
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
