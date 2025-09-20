package nanostore_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// EXCEPTION: This test validates error conditions and configuration failures.
// It creates fresh stores to test specific validation scenarios.

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  nanostore.Config
		wantErr bool
	}{
		{
			name:    "empty config",
			config:  nanostore.Config{},
			wantErr: true, // Empty config requires at least one dimension
		},
		{
			name: "valid enumerated dimension",
			config: nanostore.Config{
				Dimensions: []nanostore.DimensionConfig{
					{
						Name:         "status",
						Type:         nanostore.Enumerated,
						Values:       []string{"todo", "done"},
						DefaultValue: "todo",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid hierarchical dimension",
			config: nanostore.Config{
				Dimensions: []nanostore.DimensionConfig{
					{
						Name:     "parent",
						Type:     nanostore.Hierarchical,
						RefField: "parent_uuid",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nanostore.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
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

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
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

	store, err := nanostore.New(tmpfile.Name(), nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
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
