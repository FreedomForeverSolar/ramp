package uiapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// getConfigPath is a function variable to allow mocking in tests
var getConfigPath = getAppConfigPathImpl

// GetAppConfigPath returns the platform-specific config path
func GetAppConfigPath() (string, error) {
	return getConfigPath()
}

// getAppConfigPathImpl is the actual implementation
func getAppConfigPathImpl() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, "Library", "Application Support", "ramp-ui")
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "ramp-ui")
	default: // linux and others
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config", "ramp-ui")
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

// LoadAppConfig loads the app configuration
func LoadAppConfig() (*AppConfig, error) {
	configPath, err := GetAppConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &AppConfig{
				Projects: []ProjectRef{},
				Preferences: Preferences{
					Theme:         "system",
					ShowGitStatus: true,
				},
			}, nil
		}
		return nil, err
	}

	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveAppConfig saves the app configuration
func SaveAppConfig(config *AppConfig) error {
	configPath, err := GetAppConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// AddProjectToConfig adds a project reference to the app config
func AddProjectToConfig(path string) (string, error) {
	config, err := LoadAppConfig()
	if err != nil {
		return "", err
	}

	// Check if project already exists
	for _, p := range config.Projects {
		if p.Path == path {
			return p.ID, nil
		}
	}

	// Generate new ID
	id := uuid.New().String()[:8]

	config.Projects = append(config.Projects, ProjectRef{
		ID:      id,
		Path:    path,
		AddedAt: time.Now(),
	})

	if err := SaveAppConfig(config); err != nil {
		return "", err
	}

	return id, nil
}

// RemoveProjectFromConfig removes a project reference from the app config
func RemoveProjectFromConfig(id string) error {
	config, err := LoadAppConfig()
	if err != nil {
		return err
	}

	// Find and remove the project
	newProjects := make([]ProjectRef, 0, len(config.Projects))
	for _, p := range config.Projects {
		if p.ID != id {
			newProjects = append(newProjects, p)
		}
	}

	config.Projects = newProjects
	return SaveAppConfig(config)
}

// GetProjectRefByID gets a project reference by ID
func GetProjectRefByID(id string) (*ProjectRef, error) {
	config, err := LoadAppConfig()
	if err != nil {
		return nil, err
	}

	for _, p := range config.Projects {
		if p.ID == id {
			return &p, nil
		}
	}

	return nil, nil
}
