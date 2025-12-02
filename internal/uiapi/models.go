package uiapi

import "time"

// Project represents a Ramp project in the UI
type Project struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Path                string    `json:"path"`
	AddedAt             time.Time `json:"addedAt"`
	Repos               []Repo    `json:"repos,omitempty"`
	Features            []string  `json:"features,omitempty"`
	Commands            []Command `json:"commands,omitempty"`
	BasePort            int       `json:"basePort,omitempty"`
	DefaultBranchPrefix string    `json:"defaultBranchPrefix,omitempty"`
	HasSetupScript      bool      `json:"hasSetupScript"`
	HasCleanupScript    bool      `json:"hasCleanupScript"`
}

// Repo represents a repository in a project
type Repo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Git         string `json:"git"`
	AutoRefresh bool   `json:"autoRefresh"`
}

// Command represents a custom command from the project config
type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// Feature represents a feature/worktree in a project
type Feature struct {
	Name                  string    `json:"name"`
	Repos                 []string  `json:"repos"`
	Created               time.Time `json:"created,omitempty"`
	HasUncommittedChanges bool      `json:"hasUncommittedChanges"`
}

// AppConfig is the UI application configuration stored locally
type AppConfig struct {
	Projects    []ProjectRef `json:"projects"`
	Preferences Preferences  `json:"preferences"`
}

// ProjectRef is a reference to a project stored in app config
type ProjectRef struct {
	ID      string    `json:"id"`
	Path    string    `json:"path"`
	AddedAt time.Time `json:"addedAt"`
}

// Preferences stores user preferences
type Preferences struct {
	Theme         string `json:"theme"`
	ShowGitStatus bool   `json:"showGitStatus"`
}

// API Request/Response types

// AddProjectRequest is the request body for adding a project
type AddProjectRequest struct {
	Path string `json:"path"`
}

// CreateFeatureRequest is the request body for creating a feature
type CreateFeatureRequest struct {
	Name string `json:"name"`
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
	Type       string `json:"type"`       // "progress", "error", "complete"
	Operation  string `json:"operation"`  // "up", "down", "refresh", etc.
	Message    string `json:"message"`
	Percentage int    `json:"percentage,omitempty"`
}
