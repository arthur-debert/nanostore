package store

import (
	"time"

	"github.com/arthur-debert/nanostore/types"
)

// HybridDocument extends the standard document with body storage metadata
type HybridDocument struct {
	// Standard document fields
	UUID       string                 `json:"uuid"`
	SimpleID   string                 `json:"id,omitempty"`
	Title      string                 `json:"title"`
	Body       string                 `json:"body,omitempty"`      // Only used when BodyMeta.Type is embedded
	BodyMeta   *BodyMetadata          `json:"body_meta,omitempty"` // Metadata about body storage
	Dimensions map[string]interface{} `json:"dimensions,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// ToStandardDocument converts a HybridDocument to a standard types.Document
func (h *HybridDocument) ToStandardDocument() types.Document {
	return types.Document{
		UUID:       h.UUID,
		SimpleID:   h.SimpleID,
		Title:      h.Title,
		Body:       h.Body, // This will be populated by ReadBody when needed
		Dimensions: h.Dimensions,
		CreatedAt:  h.CreatedAt,
		UpdatedAt:  h.UpdatedAt,
	}
}

// FromStandardDocument creates a HybridDocument from a standard document
func FromStandardDocument(doc types.Document, bodyMeta *BodyMetadata, embeddedBody string) *HybridDocument {
	return &HybridDocument{
		UUID:       doc.UUID,
		SimpleID:   doc.SimpleID,
		Title:      doc.Title,
		Body:       embeddedBody, // Only set if body is embedded
		BodyMeta:   bodyMeta,
		Dimensions: doc.Dimensions,
		CreatedAt:  doc.CreatedAt,
		UpdatedAt:  doc.UpdatedAt,
	}
}

// HybridStoreData represents the JSON file structure for hybrid storage
type HybridStoreData struct {
	Documents []HybridDocument `json:"documents"`
	Metadata  HybridMetadata   `json:"metadata"`
}

// HybridMetadata extends the standard metadata with hybrid storage info
type HybridMetadata struct {
	Version        string    `json:"version"`
	StorageVersion string    `json:"storage_version"` // "hybrid_v1"
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Hybrid storage specific metadata
	BodyStorageConfig struct {
		EmbedSizeLimit int64  `json:"embed_size_limit"`
		BodiesDir      string `json:"bodies_dir"`
	} `json:"body_storage,omitempty"`
}
