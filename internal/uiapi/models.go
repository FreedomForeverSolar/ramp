package uiapi

import "time"

// Project represents a Ramp project in the UI
type Project struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Path                string    `json:"path"`
	AddedAt             time.Time `json:"addedAt"`
	Order               int       `json:"order"`
	IsFavorite          bool      `json:"isFavorite"`
	Repos               []Repo    `json:"repos,omitempty"`
	Features            []string  `json:"features"`
	BasePort            int       `json:"basePort,omitempty"`
	DefaultBranchPrefix string    `json:"defaultBranchPrefix,omitempty"`
}

// Repo represents a repository in a project
type Repo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Git         string `json:"git"`
	AutoRefresh bool   `json:"autoRefresh"`
}

// DiffStats holds git diff statistics for uncommitted changes
type DiffStats struct {
	FilesChanged int `json:"filesChanged"`
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`
}

// StatusStats holds git status statistics for working directory
type StatusStats struct {
	UntrackedFiles int `json:"untrackedFiles"`
	StagedFiles    int `json:"stagedFiles"`
	ModifiedFiles  int `json:"modifiedFiles"`
}

// FeatureWorktreeStatus holds detailed status for a single repo worktree
type FeatureWorktreeStatus struct {
	RepoName       string       `json:"repoName"`
	BranchName     string       `json:"branchName"`
	HasUncommitted bool         `json:"hasUncommitted"`
	DiffStats      *DiffStats   `json:"diffStats,omitempty"`
	StatusStats    *StatusStats `json:"statusStats,omitempty"`
	AheadCount     int          `json:"aheadCount"`
	BehindCount    int          `json:"behindCount"`
	IsMerged       bool         `json:"isMerged"`
	Error          string       `json:"error,omitempty"`
}

// Feature represents a feature/worktree in a project
type Feature struct {
	Name                  string                  `json:"name"`
	Repos                 []string                `json:"repos"`
	Created               time.Time               `json:"created,omitempty"`
	HasUncommittedChanges bool                    `json:"hasUncommittedChanges"`
	Category              string                  `json:"category"`                        // "in_flight", "merged", "clean"
	WorktreeStatuses      []FeatureWorktreeStatus `json:"worktreeStatuses,omitempty"`
}

// AppConfig is the UI application configuration stored locally
type AppConfig struct {
	Projects    []ProjectRef `json:"projects"`
	Preferences Preferences  `json:"preferences"`
}

// ProjectRef is a reference to a project stored in app config
type ProjectRef struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	AddedAt    time.Time `json:"addedAt"`
	Order      int       `json:"order"`
	IsFavorite bool      `json:"isFavorite"`
}

// Preferences stores user preferences
type Preferences struct {
	Theme                 string `json:"theme"`
	ShowGitStatus         bool   `json:"showGitStatus"`
	TerminalApp           string `json:"terminalApp"`           // "terminal", "iterm", "warp", or custom command
	LastSelectedProjectID string `json:"lastSelectedProjectId"` // Remember last selected project across launches
}

// API Request/Response types

// AddProjectRequest is the request body for adding a project
type AddProjectRequest struct {
	Path string `json:"path"`
}

// CreateFeatureRequest is the request body for creating a feature
type CreateFeatureRequest struct {
	Name string `json:"name"`

	// Optional - branch configuration (mirrors operations.UpOptions)
	Prefix   string `json:"prefix,omitempty"`   // Branch prefix override (empty = use config default)
	NoPrefix bool   `json:"noPrefix,omitempty"` // Explicitly disable prefix
	Target   string `json:"target,omitempty"`   // Source branch/feature to create from

	// For "From Branch" flow - derives prefix and target from remote branch name
	// When set, parses like CLI --from: derives prefix from branch path, sets target to origin/{fromBranch}
	// If Name is empty, derives feature name from the last segment of the branch
	FromBranch string `json:"fromBranch,omitempty"`

	// Optional - pre-operation behavior
	AutoInstall  bool `json:"autoInstall,omitempty"`  // Auto-install repos if not present
	ForceRefresh bool `json:"forceRefresh,omitempty"` // Force refresh ALL repos (override per-repo config)
	SkipRefresh  bool `json:"skipRefresh,omitempty"`  // Skip refresh for ALL repos (override per-repo config)
}

// ProjectsResponse is the response for listing projects
type ProjectsResponse struct {
	Projects []Project `json:"projects"`
}

// FeaturesResponse is the response for listing features
type FeaturesResponse struct {
	Features []Feature `json:"features"`
}

// ErrorResponse is a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse is a standard success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// WebSocket message types

// WSMessage is a WebSocket message
type WSMessage struct {
	Type       string `json:"type"`                  // "progress", "error", "complete", "output"
	Operation  string `json:"operation"`             // "up", "down", "refresh", "run", etc.
	Message    string `json:"message"`
	Percentage int    `json:"percentage,omitempty"`
	Target     string `json:"target,omitempty"`      // Feature name for filtering
	Command    string `json:"command,omitempty"`     // Command name for run operations
}

// Command represents a custom command defined in ramp.yaml
type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Scope   string `json:"scope,omitempty"` // "source", "feature", or empty (both)
}

// CommandsResponse is the response for listing commands
type CommandsResponse struct {
	Commands []Command `json:"commands"`
}

// RunCommandRequest is the request body for running a command
type RunCommandRequest struct {
	FeatureName string `json:"featureName,omitempty"` // Optional - if empty, runs against source
}

// CancelCommandRequest is the request body for cancelling a command
type CancelCommandRequest struct {
	Target string `json:"target,omitempty"` // "source" or feature name
}

// RunCommandResponse is the response for command execution
type RunCommandResponse struct {
	Success  bool   `json:"success"`
	ExitCode int    `json:"exitCode"`
	Duration int64  `json:"duration"` // milliseconds
	Error    string `json:"error,omitempty"`
}

// Config types for local preferences

// PromptOption represents a single option in a prompt
type PromptOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Prompt represents a configuration prompt defined in ramp.yaml
type Prompt struct {
	Name     string         `json:"name"`
	Question string         `json:"question"`
	Options  []PromptOption `json:"options"`
	Default  string         `json:"default,omitempty"`
}

// ConfigStatusResponse is the response for checking if config is needed
type ConfigStatusResponse struct {
	NeedsConfig bool     `json:"needsConfig"`
	Prompts     []Prompt `json:"prompts,omitempty"`
}

// ConfigResponse is the response for getting current config
type ConfigResponse struct {
	Preferences map[string]string `json:"preferences"`
}

// SaveConfigRequest is the request body for saving config
type SaveConfigRequest struct {
	Preferences map[string]string `json:"preferences"`
}

// ReorderProjectsRequest is the request body for reordering projects
type ReorderProjectsRequest struct {
	ProjectIDs []string `json:"projectIds"`
}

// ToggleFavoriteResponse is the response for toggling favorite status
type ToggleFavoriteResponse struct {
	IsFavorite bool `json:"isFavorite"`
}

// SourceRepoStatus holds git status for a source repository
type SourceRepoStatus struct {
	Name        string `json:"name"`
	Branch      string `json:"branch"`
	AheadCount  int    `json:"aheadCount"`
	BehindCount int    `json:"behindCount"`
	IsInstalled bool   `json:"isInstalled"`
	Error       string `json:"error,omitempty"`
}

// SourceReposResponse is the response for getting source repo status
type SourceReposResponse struct {
	Repos []SourceRepoStatus `json:"repos"`
}

// InstallResponse is the response for installing source repos
type InstallResponse struct {
	ClonedRepos  []string `json:"clonedRepos"`
	SkippedRepos []string `json:"skippedRepos"`
	Message      string   `json:"message"`
}

// OpenTerminalRequest is the request body for opening a terminal
type OpenTerminalRequest struct {
	Path string `json:"path"`
}

// AppSettingsResponse is the response for getting app settings
type AppSettingsResponse struct {
	TerminalApp           string `json:"terminalApp"`
	LastSelectedProjectID string `json:"lastSelectedProjectId"`
	Theme                 string `json:"theme"`
}

// SaveAppSettingsRequest is the request body for saving app settings
type SaveAppSettingsRequest struct {
	TerminalApp           string `json:"terminalApp,omitempty"`
	LastSelectedProjectID string `json:"lastSelectedProjectId,omitempty"`
	Theme                 string `json:"theme,omitempty"`
}

// PruneFailure represents a feature that failed to be pruned
type PruneFailure struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// PruneResponse is the response for pruning merged features
type PruneResponse struct {
	Pruned  []string       `json:"pruned"`
	Failed  []PruneFailure `json:"failed"`
	Message string         `json:"message"`
}
