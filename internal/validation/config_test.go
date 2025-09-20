package validation_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// EXCEPTION: This test validates error conditions and configuration failures.
// It creates fresh stores to test specific validation scenarios.

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/internal/validation"
	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/types"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  types.Config
		wantErr bool
	}{
		{
			name:    "empty config",
			config:  types.Config{},
			wantErr: true, // Empty config requires at least one dimension
		},
		{
			name: "valid enumerated dimension",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:         "status",
						Type:         types.Enumerated,
						Values:       []string{"todo", "done"},
						DefaultValue: "todo",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid hierarchical dimension",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "parent",
						Type:     types.Hierarchical,
						RefField: "parent_uuid",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create DimensionSet from Config to test validation
			ds := types.DimensionSetFromConfig(types.Config{
				Dimensions: tt.config.Dimensions,
			})
			err := validation.Validate(ds)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewJSONStore(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"todo", "done"},
				DefaultValue: "todo",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// The store should be created successfully
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNotImplemented(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := nanostore.New(tmpfile.Name(), types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"todo", "done"},
				DefaultValue: "todo",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// All bulk operations are now implemented!
}
