package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/arthur-debert/nanostore/nanostore"
)

// Global store registry to manage store instances
var stores = make(map[string]nanostore.Store)
var storeCounter = 0

// Helper to generate unique store handles
func nextStoreHandle() string {
	storeCounter++
	return fmt.Sprintf("store_%d", storeCounter)
}

//export nanostore_new
func nanostore_new(dbPath *C.char, configJSON *C.char) *C.char {
	goDbPath := C.GoString(dbPath)
	goConfigJSON := C.GoString(configJSON)

	// Parse JSON config
	var config nanostore.Config
	if err := json.Unmarshal([]byte(goConfigJSON), &config); err != nil {
		return C.CString(fmt.Sprintf(`{"error": "invalid config: %s"}`, err.Error()))
	}

	// Create store
	store, err := nanostore.New(goDbPath, config)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "failed to create store: %s"}`, err.Error()))
	}

	// Register store and return handle
	handle := nextStoreHandle()
	stores[handle] = store

	result := map[string]string{"handle": handle}
	resultJSON, _ := json.Marshal(result)
	return C.CString(string(resultJSON))
}

//export nanostore_add
func nanostore_add(handle *C.char, title *C.char, dimensionsJSON *C.char) *C.char {
	goHandle := C.GoString(handle)
	goTitle := C.GoString(title)
	goDimensionsJSON := C.GoString(dimensionsJSON)

	store, exists := stores[goHandle]
	if !exists {
		return C.CString(`{"error": "invalid store handle"}`)
	}

	// Parse dimensions
	var dimensions map[string]interface{}
	if goDimensionsJSON != "" {
		if err := json.Unmarshal([]byte(goDimensionsJSON), &dimensions); err != nil {
			return C.CString(fmt.Sprintf(`{"error": "invalid dimensions: %s"}`, err.Error()))
		}
	}

	// Add document
	uuid, err := store.Add(goTitle, dimensions)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "failed to add: %s"}`, err.Error()))
	}

	result := map[string]string{"uuid": uuid}
	resultJSON, _ := json.Marshal(result)
	return C.CString(string(resultJSON))
}

//export nanostore_list
func nanostore_list(handle *C.char, filtersJSON *C.char) *C.char {
	goHandle := C.GoString(handle)
	goFiltersJSON := C.GoString(filtersJSON)

	store, exists := stores[goHandle]
	if !exists {
		return C.CString(`{"error": "invalid store handle"}`)
	}

	// Parse filters
	var filters map[string]interface{}
	if goFiltersJSON != "" {
		if err := json.Unmarshal([]byte(goFiltersJSON), &filters); err != nil {
			return C.CString(fmt.Sprintf(`{"error": "invalid filters: %s"}`, err.Error()))
		}
	}

	// Create ListOptions
	opts := nanostore.ListOptions{Filters: filters}

	// List documents
	docs, err := store.List(opts)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "failed to list: %s"}`, err.Error()))
	}

	// Convert to JSON-serializable format
	result := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		result[i] = map[string]interface{}{
			"uuid":           doc.UUID,
			"user_facing_id": doc.UserFacingID,
			"title":          doc.Title,
			"body":           doc.Body,
			"dimensions":     doc.Dimensions,
			"created_at":     doc.CreatedAt.Unix(),
			"updated_at":     doc.UpdatedAt.Unix(),
		}
	}

	resultJSON, _ := json.Marshal(result)
	return C.CString(string(resultJSON))
}

//export nanostore_update
func nanostore_update(handle *C.char, id *C.char, updatesJSON *C.char) *C.char {
	goHandle := C.GoString(handle)
	goID := C.GoString(id)
	goUpdatesJSON := C.GoString(updatesJSON)

	store, exists := stores[goHandle]
	if !exists {
		return C.CString(`{"error": "invalid store handle"}`)
	}

	// Parse updates
	var updateData map[string]interface{}
	if err := json.Unmarshal([]byte(goUpdatesJSON), &updateData); err != nil {
		return C.CString(fmt.Sprintf(`{"error": "invalid updates: %s"}`, err.Error()))
	}

	// Build UpdateRequest
	var updates nanostore.UpdateRequest
	if title, ok := updateData["title"].(string); ok {
		updates.Title = &title
	}
	if body, ok := updateData["body"].(string); ok {
		updates.Body = &body
	}
	if dims, ok := updateData["dimensions"].(map[string]interface{}); ok {
		updates.Dimensions = make(map[string]string)
		for k, v := range dims {
			if strVal, ok := v.(string); ok {
				updates.Dimensions[k] = strVal
			}
		}
	}

	// Update document
	if err := store.Update(goID, updates); err != nil {
		return C.CString(fmt.Sprintf(`{"error": "failed to update: %s"}`, err.Error()))
	}

	return C.CString(`{"success": true}`)
}

//export nanostore_delete
func nanostore_delete(handle *C.char, id *C.char, cascade C.int) *C.char {
	goHandle := C.GoString(handle)
	goID := C.GoString(id)
	goCascade := cascade != 0

	store, exists := stores[goHandle]
	if !exists {
		return C.CString(`{"error": "invalid store handle"}`)
	}

	// Delete document
	if err := store.Delete(goID, goCascade); err != nil {
		return C.CString(fmt.Sprintf(`{"error": "failed to delete: %s"}`, err.Error()))
	}

	return C.CString(`{"success": true}`)
}

//export nanostore_resolve_uuid
func nanostore_resolve_uuid(handle *C.char, userFacingID *C.char) *C.char {
	goHandle := C.GoString(handle)
	goUserFacingID := C.GoString(userFacingID)

	store, exists := stores[goHandle]
	if !exists {
		return C.CString(`{"error": "invalid store handle"}`)
	}

	// Resolve UUID
	uuid, err := store.ResolveUUID(goUserFacingID)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "failed to resolve: %s"}`, err.Error()))
	}

	result := map[string]string{"uuid": uuid}
	resultJSON, _ := json.Marshal(result)
	return C.CString(string(resultJSON))
}

//export nanostore_close
func nanostore_close(handle *C.char) *C.char {
	goHandle := C.GoString(handle)

	store, exists := stores[goHandle]
	if !exists {
		return C.CString(`{"error": "invalid store handle"}`)
	}

	// Close store
	if err := store.Close(); err != nil {
		return C.CString(fmt.Sprintf(`{"error": "failed to close: %s"}`, err.Error()))
	}

	// Remove from registry
	delete(stores, goHandle)

	return C.CString(`{"success": true}`)
}

//export nanostore_free_string
func nanostore_free_string(str *C.char) {
	C.free(unsafe.Pointer(str))
}

func main() {
	// Required for CGO but not used
}
