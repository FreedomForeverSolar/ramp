package operations

import (
	"fmt"
	"os"

	"ramp/internal/config"
	"ramp/internal/features"
)

// LoadDisplayName loads the display name for a feature from metadata.
// Returns empty string if no display name is set or if there's an error.
func LoadDisplayName(projectDir, featureName string) string {
	metadataStore, err := features.NewMetadataStore(projectDir)
	if err != nil {
		return ""
	}
	return metadataStore.GetDisplayName(featureName)
}

// BuildEnvVars builds the environment variables map for env file processing and script execution.
func BuildEnvVars(projectDir, treesDir, featureName, displayName string, allocatedPorts []int, cfg *config.Config, repos map[string]*config.Repo) map[string]string {
	envVars := make(map[string]string)

	// Standard RAMP variables
	envVars["RAMP_PROJECT_DIR"] = projectDir
	envVars["RAMP_TREES_DIR"] = treesDir
	envVars["RAMP_WORKTREE_NAME"] = featureName
	envVars["RAMP_DISPLAY_NAME"] = displayName

	// Add port variables if configured
	if cfg.HasPortConfig() && len(allocatedPorts) > 0 {
		envVars["RAMP_PORT"] = fmt.Sprintf("%d", allocatedPorts[0])
		for i, port := range allocatedPorts {
			envVars[fmt.Sprintf("RAMP_PORT_%d", i+1)] = fmt.Sprintf("%d", port)
		}
	}

	// Add repo path variables
	for name, repo := range repos {
		envVarName := config.GenerateEnvVarName(name)
		repoPath := repo.GetRepoPath(projectDir)
		envVars[envVarName] = repoPath
	}

	// Add local config environment variables (from prompts)
	localCfg, err := config.LoadLocalConfig(projectDir)
	if err == nil && localCfg != nil {
		for key, value := range localCfg.Preferences {
			envVars[key] = value
		}
	}

	return envVars
}

// HasEnvFiles checks if any repository has env_files configured.
func HasEnvFiles(repos map[string]*config.Repo) bool {
	for _, repo := range repos {
		if len(repo.EnvFiles) > 0 {
			return true
		}
	}
	return false
}

// GetLocalEnvVars loads local preferences and returns them as environment variables.
// Returns empty map if no local config exists (not an error).
func GetLocalEnvVars(projectDir string) (map[string]string, error) {
	localCfg, err := config.LoadLocalConfig(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load local config: %w", err)
	}

	if localCfg == nil {
		return make(map[string]string), nil
	}

	return localCfg.Preferences, nil
}

// BuildScriptEnv builds the environment variables for script execution as a slice.
// This wraps BuildEnvVars and converts the result to the format expected by exec.Cmd.Env.
func BuildScriptEnv(projectDir, treesDir, featureName, displayName string, allocatedPorts []int, cfg *config.Config, repos map[string]*config.Repo) []string {
	env := os.Environ()
	envVars := BuildEnvVars(projectDir, treesDir, featureName, displayName, allocatedPorts, cfg, repos)
	for key, value := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}
