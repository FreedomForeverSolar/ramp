package uiapi

import (
	"encoding/json"
	"net/http"

	"ramp/internal/config"
	"ramp/internal/operations"

	"github.com/gorilla/mux"
)

// ListCommands returns all commands defined in the project config
func (s *Server) ListCommands(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	// Load project config
	cfg, err := config.LoadConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load project config", err.Error())
		return
	}

	// Convert config commands to API commands
	configCommands := cfg.Commands
	commands := make([]Command, 0, len(configCommands))
	for _, cmd := range configCommands {
		commands = append(commands, Command{
			Name:    cmd.Name,
			Command: cmd.Command,
			Scope:   cmd.Scope,
		})
	}

	writeJSON(w, http.StatusOK, CommandsResponse{Commands: commands})
}

// RunCommand executes a custom command
func (s *Server) RunCommand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	commandName := vars["commandName"]

	var req RunCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Body might be empty, which is fine (runs against source)
		req = RunCommandRequest{}
	}

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	// Load project config
	cfg, err := config.LoadConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load project config", err.Error())
		return
	}

	// Validate command exists
	command := cfg.GetCommand(commandName)
	if command == nil {
		writeError(w, http.StatusNotFound, "Command not found", commandName)
		return
	}

	// Determine target for message filtering
	// Use feature name if provided, otherwise use "source" as identifier
	target := req.FeatureName
	if target == "" {
		target = "source"
	}

	// Create progress reporter with command context
	progress := operations.NewWSProgressReporterWithCommand("run", target, commandName, func(msg interface{}) {
		s.broadcast(msg)
	})

	// Create output streamer with context for filtering
	output := operations.NewWSOutputStreamerWithContext("run", target, commandName, func(msg interface{}) {
		s.broadcast(msg)
	})

	// Execute the command
	result, err := operations.RunCommand(operations.RunOptions{
		ProjectDir:  ref.Path,
		Config:      cfg,
		CommandName: commandName,
		FeatureName: req.FeatureName,
		Progress:    progress,
		Output:      output,
	})

	// Always return a response (even on error, for exit code)
	if result != nil {
		response := RunCommandResponse{
			Success:  err == nil && result.ExitCode == 0,
			ExitCode: result.ExitCode,
			Duration: result.Duration.Milliseconds(),
		}
		if err != nil {
			response.Error = err.Error()
		}
		writeJSON(w, http.StatusOK, response)
		return
	}

	// Handle case where result is nil (shouldn't happen, but be safe)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to run command", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, RunCommandResponse{Success: true})
}
