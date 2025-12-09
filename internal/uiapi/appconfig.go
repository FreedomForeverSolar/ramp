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
					TerminalApp:   "terminal", // macOS Terminal.app default
				},
			}, nil
		}
		return nil, err
	}

	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Migration: assign sequential orders if all projects have Order=0
	if len(config.Projects) > 1 {
		allZero := true
		for _, p := range config.Projects {
			if p.Order != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			for i := range config.Projects {
				config.Projects[i].Order = i
			}
			// Save migrated config
			_ = SaveAppConfig(&config)
		}
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

	// Generate new ID and calculate order
	id := uuid.New().String()[:8]
	maxOrder := -1
	for _, p := range config.Projects {
		if p.Order > maxOrder {
			maxOrder = p.Order
		}
	}

	config.Projects = append(config.Projects, ProjectRef{
		ID:         id,
		Path:       path,
		AddedAt:    time.Now(),
		Order:      maxOrder + 1,
		IsFavorite: false,
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

// ReorderProjects sets the order of projects based on the provided ID array
func ReorderProjects(projectIDs []string) error {
	config, err := LoadAppConfig()
	if err != nil {
		return err
	}

	// Create a map for quick lookup
	projectMap := make(map[string]*ProjectRef)
	for i := range config.Projects {
		projectMap[config.Projects[i].ID] = &config.Projects[i]
	}

	// Update order based on position in array
	for order, id := range projectIDs {
		if project, ok := projectMap[id]; ok {
			project.Order = order
		}
	}

	return SaveAppConfig(config)
}

// ToggleProjectFavorite toggles the favorite status of a project
func ToggleProjectFavorite(id string) (bool, error) {
	config, err := LoadAppConfig()
	if err != nil {
		return false, err
	}

	var newStatus bool
	found := false
	for i := range config.Projects {
		if config.Projects[i].ID == id {
			config.Projects[i].IsFavorite = !config.Projects[i].IsFavorite
			newStatus = config.Projects[i].IsFavorite
			found = true
			break
		}
	}

	if !found {
		return false, nil
	}

	if err := SaveAppConfig(config); err != nil {
		return false, err
	}

	return newStatus, nil
}
