package nanostore

import (
	"reflect"
	"strings"
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ParentID", "parent_id"},
		{"Status", "status"},
		{"HTTPCode", "http_code"},
		{"XMLParser", "xml_parser"},
		{"ID", "id"},
		{"HTML", "html"},
		{"SimpleWord", "simple_word"},
		{"lowercase", "lowercase"},
		{"TwoWords", "two_words"},
		{"ThreeWordExample", "three_word_example"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := toSnakeCase(tc.input)
			if result != tc.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestParseStructTags(t *testing.T) {
	t.Run("basic enum field", func(t *testing.T) {
		type TestStruct struct {
			Status string `values:"pending,active,done"`
		}

		metas, err := parseStructTags(reflect.TypeOf(TestStruct{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(metas) != 1 {
			t.Fatalf("expected 1 field meta, got %d", len(metas))
		}

		meta := metas[0]
		if meta.fieldName != "Status" {
			t.Errorf("expected field name 'Status', got %q", meta.fieldName)
		}
		if meta.dimensionName != "status" {
			t.Errorf("expected dimension name 'status', got %q", meta.dimensionName)
		}
		if len(meta.values) != 3 {
			t.Errorf("expected 3 values, got %d", len(meta.values))
		}
		expectedValues := []string{"pending", "active", "done"}
		for i, v := range expectedValues {
			if i >= len(meta.values) || meta.values[i] != v {
				t.Errorf("expected value[%d] = %q, got %q", i, v, meta.values[i])
			}
		}
	})

	t.Run("hierarchical ref field", func(t *testing.T) {
		type TestStruct struct {
			ParentID string `dimension:"parent_id,ref"`
		}

		metas, err := parseStructTags(reflect.TypeOf(TestStruct{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(metas) != 1 {
			t.Fatalf("expected 1 field meta, got %d", len(metas))
		}

		meta := metas[0]
		if meta.dimensionName != "parent_id" {
			t.Errorf("expected dimension name 'parent_id', got %q", meta.dimensionName)
		}
		if !meta.isRef {
			t.Error("expected isRef to be true")
		}
	})

	t.Run("implicit dimension name with ref", func(t *testing.T) {
		type TestStruct struct {
			ParentID string `dimension:",ref"`
		}

		metas, err := parseStructTags(reflect.TypeOf(TestStruct{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		meta := metas[0]
		if meta.dimensionName != "parent_id" {
			t.Errorf("expected dimension name 'parent_id', got %q", meta.dimensionName)
		}
		if !meta.isRef {
			t.Error("expected isRef to be true")
		}
	})

	t.Run("field with prefixes", func(t *testing.T) {
		type TestStruct struct {
			Priority string `values:"low,medium,high" prefix:"high=h,medium=m"`
		}

		metas, err := parseStructTags(reflect.TypeOf(TestStruct{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		meta := metas[0]
		if meta.prefixes["high"] != "h" {
			t.Errorf("expected prefix for 'high' to be 'h', got %q", meta.prefixes["high"])
		}
		if meta.prefixes["medium"] != "m" {
			t.Errorf("expected prefix for 'medium' to be 'm', got %q", meta.prefixes["medium"])
		}
	})

	t.Run("field with default", func(t *testing.T) {
		type TestStruct struct {
			Status string `values:"pending,active" default:"pending"`
		}

		metas, err := parseStructTags(reflect.TypeOf(TestStruct{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		meta := metas[0]
		if meta.defaultValue != "pending" {
			t.Errorf("expected default value 'pending', got %q", meta.defaultValue)
		}
	})

	t.Run("excluded field", func(t *testing.T) {
		type TestStruct struct {
			Status   string `values:"pending,active"`
			Internal string `dimension:"-"`
		}

		metas, err := parseStructTags(reflect.TypeOf(TestStruct{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(metas) != 2 {
			t.Fatalf("expected 2 field metas, got %d", len(metas))
		}

		// Find the Internal field
		var internalMeta *fieldMeta
		for i := range metas {
			if metas[i].fieldName == "Internal" {
				internalMeta = &metas[i]
				break
			}
		}

		if internalMeta == nil {
			t.Fatal("Internal field meta not found")
		}
		if !internalMeta.skipDimension {
			t.Error("expected Internal field to have skipDimension=true")
		}
	})

	t.Run("embedded Document", func(t *testing.T) {
		type TestStruct struct {
			Document
			Status string `values:"pending,active"`
		}

		metas, err := parseStructTags(reflect.TypeOf(TestStruct{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only have Status field, not Document fields
		if len(metas) != 1 {
			t.Fatalf("expected 1 field meta, got %d", len(metas))
		}
		if metas[0].fieldName != "Status" {
			t.Errorf("expected field name 'Status', got %q", metas[0].fieldName)
		}
	})

	t.Run("custom string type without values", func(t *testing.T) {
		type Status string
		type TestStruct struct {
			Status Status
		}

		_, err := parseStructTags(reflect.TypeOf(TestStruct{}))
		if err == nil {
			t.Fatal("expected error for custom string type without values")
		}
		if !strings.Contains(err.Error(), "requires 'values' tag") {
			t.Errorf("expected error about missing values tag, got: %v", err)
		}
	})

	t.Run("non-string field", func(t *testing.T) {
		type TestStruct struct {
			Count int
		}

		_, err := parseStructTags(reflect.TypeOf(TestStruct{}))
		if err == nil {
			t.Fatal("expected error for non-string field")
		}
		if !strings.Contains(err.Error(), "only string dimensions") {
			t.Errorf("expected error about string dimensions, got: %v", err)
		}
	})
}

func TestBuildConfigFromMeta(t *testing.T) {
	t.Run("basic configuration", func(t *testing.T) {
		metas := []fieldMeta{
			{
				fieldName:     "Status",
				dimensionName: "status",
				values:        []string{"pending", "active", "done"},
				defaultValue:  "pending",
				prefixes:      map[string]string{},
			},
			{
				fieldName:     "Priority",
				dimensionName: "priority",
				values:        []string{"low", "medium", "high"},
				prefixes:      map[string]string{"high": "h"},
				defaultValue:  "medium",
			},
		}

		config, err := buildConfigFromMeta(metas)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(config.Dimensions) != 2 {
			t.Fatalf("expected 2 dimensions, got %d", len(config.Dimensions))
		}

		// Find status dimension
		var statusDim *DimensionConfig
		for i := range config.Dimensions {
			if config.Dimensions[i].Name == "status" {
				statusDim = &config.Dimensions[i]
				break
			}
		}
		if statusDim == nil {
			t.Fatal("status dimension not found")
		}

		if statusDim.Type != Enumerated {
			t.Errorf("expected status to be enumerated, got %v", statusDim.Type)
		}
		if statusDim.DefaultValue != "pending" {
			t.Errorf("expected status default to be 'pending', got %q", statusDim.DefaultValue)
		}
		if len(statusDim.Values) != 3 {
			t.Errorf("expected 3 status values, got %d", len(statusDim.Values))
		}

		// Find priority dimension
		var priorityDim *DimensionConfig
		for i := range config.Dimensions {
			if config.Dimensions[i].Name == "priority" {
				priorityDim = &config.Dimensions[i]
				break
			}
		}
		if priorityDim == nil {
			t.Fatal("priority dimension not found")
		}

		if priorityDim.Prefixes["high"] != "h" {
			t.Errorf("expected prefix for 'high' to be 'h', got %q", priorityDim.Prefixes["high"])
		}
	})

	t.Run("hierarchical dimension", func(t *testing.T) {
		metas := []fieldMeta{
			{
				fieldName:     "ParentID",
				dimensionName: "parent_id",
				isRef:         true,
				prefixes:      map[string]string{},
			},
		}

		config, err := buildConfigFromMeta(metas)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Find parent_id dimension
		var parentDim *DimensionConfig
		for i := range config.Dimensions {
			if config.Dimensions[i].Name == "parent_id" {
				parentDim = &config.Dimensions[i]
				break
			}
		}
		if parentDim == nil {
			t.Fatal("parent_id dimension not found")
		}

		if parentDim.Type != Hierarchical {
			t.Errorf("expected parent_id to be hierarchical, got %v", parentDim.Type)
		}
	})

	t.Run("duplicate dimension names", func(t *testing.T) {
		metas := []fieldMeta{
			{
				fieldName:     "Status1",
				dimensionName: "status",
				values:        []string{"a", "b"},
				prefixes:      map[string]string{},
			},
			{
				fieldName:     "Status2",
				dimensionName: "status",
				values:        []string{"c", "d"},
				prefixes:      map[string]string{},
			},
		}

		_, err := buildConfigFromMeta(metas)
		if err == nil {
			t.Fatal("expected error for duplicate dimension names")
		}
		if !strings.Contains(err.Error(), "duplicate dimension name: status") {
			t.Errorf("expected error about duplicate dimension, got: %v", err)
		}
	})

	t.Run("skipped dimensions", func(t *testing.T) {
		metas := []fieldMeta{
			{
				fieldName:     "Status",
				dimensionName: "status",
				values:        []string{"a", "b"},
				prefixes:      map[string]string{},
			},
			{
				fieldName:     "Internal",
				dimensionName: "internal",
				skipDimension: true,
				prefixes:      map[string]string{},
			},
		}

		config, err := buildConfigFromMeta(metas)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(config.Dimensions) != 1 {
			t.Errorf("expected 1 dimension (skipped one), got %d", len(config.Dimensions))
		}
		// Check that internal is not in the dimensions
		for _, dim := range config.Dimensions {
			if dim.Name == "internal" {
				t.Error("expected 'internal' dimension to be skipped")
			}
		}
	})
}
