package config

// MergedConfig holds the result of merging project, local, and user configs.
// Used internally by operations that need access to merged commands and hooks.
type MergedConfig struct {
	// From project config only (not merged)
	Name                string
	Repos               []*Repo
	Setup               string
	Cleanup             string
	DefaultBranchPrefix string
	BasePort            int
	MaxPorts            int
	PortsPerFeature     int
	Prompts             []*Prompt

	// Merged from all levels (project > local > user precedence for commands)
	Commands []*Command

	// Aggregated from all levels (all hooks from all sources execute)
	// Order: project hooks -> local hooks -> user hooks
	Hooks []*Hook

	// Reference to original project config
	ProjectConfig *Config
}

// LoadMergedConfig loads and merges all three config levels.
// This is the recommended way to get config when you need commands or hooks.
func LoadMergedConfig(projectDir string) (*MergedConfig, error) {
	projectCfg, err := LoadConfig(projectDir)
	if err != nil {
		return nil, err
	}

	localCfg, _ := LoadLocalConfig(projectDir) // nil is fine
	userCfg, _ := LoadUserConfig()             // nil is fine

	return MergeConfigs(projectCfg, localCfg, userCfg), nil
}

// MergeConfigs merges project, local, and user configs according to these rules:
// - Commands: First match wins (project > local > user precedence)
// - Hooks: Execute ALL hooks from project -> local -> user order
// - Other settings: Project only (repos, setup, cleanup, etc.)
func MergeConfigs(projectCfg *Config, localCfg *LocalConfig, userCfg *UserConfig) *MergedConfig {
	merged := &MergedConfig{
		// Project-only fields
		Name:                projectCfg.Name,
		Repos:               projectCfg.Repos,
		Setup:               projectCfg.Setup,
		Cleanup:             projectCfg.Cleanup,
		DefaultBranchPrefix: projectCfg.DefaultBranchPrefix,
		BasePort:            projectCfg.BasePort,
		MaxPorts:            projectCfg.MaxPorts,
		PortsPerFeature:     projectCfg.PortsPerFeature,
		Prompts:             projectCfg.Prompts,
		ProjectConfig:       projectCfg,
	}

	// Merge commands with precedence (project > local > user)
	merged.Commands = mergeCommands(projectCfg.Commands, localCfg, userCfg)

	// Aggregate hooks from all levels (project -> local -> user order)
	merged.Hooks = aggregateHooks(projectCfg.Hooks, localCfg, userCfg)

	return merged
}

// mergeCommands merges commands with first-match-wins precedence.
func mergeCommands(projectCmds []*Command, localCfg *LocalConfig, userCfg *UserConfig) []*Command {
	seenNames := make(map[string]bool)
	result := make([]*Command, 0)

	// Add project commands first (highest priority)
	for _, cmd := range projectCmds {
		if !seenNames[cmd.Name] {
			result = append(result, cmd)
			seenNames[cmd.Name] = true
		}
	}

	// Add local commands if name not already seen
	if localCfg != nil {
		for _, cmd := range localCfg.Commands {
			if !seenNames[cmd.Name] {
				result = append(result, cmd)
				seenNames[cmd.Name] = true
			}
		}
	}

	// Add user commands if name not already seen
	if userCfg != nil {
		for _, cmd := range userCfg.Commands {
			if !seenNames[cmd.Name] {
				result = append(result, cmd)
				seenNames[cmd.Name] = true
			}
		}
	}

	return result
}

// aggregateHooks collects all hooks from all levels.
// Execution order: project hooks -> local hooks -> user hooks
func aggregateHooks(projectHooks []*Hook, localCfg *LocalConfig, userCfg *UserConfig) []*Hook {
	result := make([]*Hook, 0)

	// Project hooks first
	result = append(result, projectHooks...)

	// Local hooks second
	if localCfg != nil {
		result = append(result, localCfg.Hooks...)
	}

	// User hooks last
	if userCfg != nil {
		result = append(result, userCfg.Hooks...)
	}

	return result
}

// GetCommand returns the first command matching the name from merged sources.
func (m *MergedConfig) GetCommand(name string) *Command {
	for _, cmd := range m.Commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

// GetHooksForEvent returns all hooks for a specific event.
func (m *MergedConfig) GetHooksForEvent(event string) []*Hook {
	result := make([]*Hook, 0)
	for _, hook := range m.Hooks {
		if hook.Event == event {
			result = append(result, hook)
		}
	}
	return result
}

// GetRepos returns repos from the project config.
func (m *MergedConfig) GetRepos() map[string]*Repo {
	return m.ProjectConfig.GetRepos()
}

// GetBranchPrefix returns the branch prefix from the project config.
func (m *MergedConfig) GetBranchPrefix() string {
	return m.DefaultBranchPrefix
}

// GetBasePort returns the base port with default fallback.
func (m *MergedConfig) GetBasePort() int {
	if m.BasePort <= 0 {
		return 3000
	}
	return m.BasePort
}

// GetMaxPorts returns max ports with default fallback.
func (m *MergedConfig) GetMaxPorts() int {
	if m.MaxPorts <= 0 {
		return 100
	}
	return m.MaxPorts
}

// GetPortsPerFeature returns ports per feature with default fallback.
func (m *MergedConfig) GetPortsPerFeature() int {
	if m.PortsPerFeature <= 0 {
		return 1
	}
	return m.PortsPerFeature
}

// HasPortConfig returns true if port configuration is set.
func (m *MergedConfig) HasPortConfig() bool {
	return m.BasePort > 0 || m.MaxPorts > 0
}
