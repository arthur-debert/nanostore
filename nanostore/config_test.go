package nanostore_test

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
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

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
	defer store.Close()

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
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

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
	defer store.Close()

	// All operations should return "not implemented" error
	_, err = store.Add("test", nil)
	if err == nil || err.Error() != "not implemented" {
		t.Errorf("expected 'not implemented' error, got %v", err)
	}

	_, err = store.List(nanostore.ListOptions{})
	if err == nil || err.Error() != "not implemented" {
		t.Errorf("expected 'not implemented' error, got %v", err)
	}
}