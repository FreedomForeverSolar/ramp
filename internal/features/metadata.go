package features

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const MetadataFile = "feature_metadata.json"

// FeatureMetadata holds metadata for a single feature.
type FeatureMetadata struct {
	DisplayName string `json:"displayName,omitempty"`
}

// MetadataStore manages feature metadata persistence.
type MetadataStore struct {
	metadata map[string]FeatureMetadata
	filePath string
}

// NewMetadataStore creates a new metadata store for the given project directory.
func NewMetadataStore(projectDir string) (*MetadataStore, error) {
	filePath := filepath.Join(projectDir, ".ramp", MetadataFile)

	ms := &MetadataStore{
		metadata: make(map[string]FeatureMetadata),
		filePath: filePath,
	}

	if err := ms.load(); err != nil {
		return nil, fmt.Errorf("failed to load feature metadata: %w", err)
	}

	return ms, nil
}

func (ms *MetadataStore) load() error {
	if _, err := os.Stat(ms.filePath); os.IsNotExist(err) {
		// File doesn't exist yet, that's fine
		return nil
	}

	data, err := os.ReadFile(ms.filePath)
	if err != nil {
		return fmt.Errorf("failed to read feature metadata file: %w", err)
	}

	if len(data) == 0 {
		// Empty file, that's fine
		return nil
	}

	if err := json.Unmarshal(data, &ms.metadata); err != nil {
		return fmt.Errorf("failed to parse feature metadata file: %w", err)
	}

	return nil
}

func (ms *MetadataStore) save() error {
	// Ensure .ramp directory exists
	if err := os.MkdirAll(filepath.Dir(ms.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create .ramp directory: %w", err)
	}

	data, err := json.MarshalIndent(ms.metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal feature metadata: %w", err)
	}

	if err := os.WriteFile(ms.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write feature metadata file: %w", err)
	}

	return nil
}

// GetDisplayName returns the display name for a feature, or empty string if not set.
func (ms *MetadataStore) GetDisplayName(featureName string) string {
	if meta, exists := ms.metadata[featureName]; exists {
		return meta.DisplayName
	}
	return ""
}

// SetDisplayName sets the display name for a feature.
// Pass empty string to clear the display name.
func (ms *MetadataStore) SetDisplayName(featureName, displayName string) error {
	if displayName == "" {
		// Remove the entry if display name is cleared
		delete(ms.metadata, featureName)
	} else {
		ms.metadata[featureName] = FeatureMetadata{
			DisplayName: displayName,
		}
	}

	if err := ms.save(); err != nil {
		return fmt.Errorf("failed to save display name: %w", err)
	}

	return nil
}

// RemoveFeature removes all metadata for a feature.
func (ms *MetadataStore) RemoveFeature(featureName string) error {
	if _, exists := ms.metadata[featureName]; !exists {
		// Already removed or never had metadata
		return nil
	}

	delete(ms.metadata, featureName)
	if err := ms.save(); err != nil {
		return fmt.Errorf("failed to save after removing feature metadata: %w", err)
	}

	return nil
}

// ListMetadata returns a copy of all feature metadata.
func (ms *MetadataStore) ListMetadata() map[string]FeatureMetadata {
	result := make(map[string]FeatureMetadata)
	for feature, meta := range ms.metadata {
		result[feature] = meta
	}
	return result
}
