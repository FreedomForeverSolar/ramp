package uiapi

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"ramp/internal/config"

	"github.com/gorilla/mux"
)

// RunCommandRequest is the request body for running a command
type RunCommandRequest struct {
	FeatureName string `json:"featureName,omitempty"` // Optional: run in feature context
}

// ListCommands returns all custom commands for a project
func (s *Server) ListCommands(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	cfg, err := config.LoadConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load project config", err.Error())
		return
	}

	commands := make([]Command, 0, len(cfg.Commands))
	for _, cmd := range cfg.Commands {
		commands = append(commands, Command{
			Name:    cmd.Name,
			Command: cmd.Command,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"commands": commands,
	})
}

// RunCommand executes a custom command and streams output via WebSocket
func (s *Server) RunCommand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	cmdName := vars["name"]

	var req RunCommandRequest
	json.NewDecoder(r.Body).Decode(&req) // Optional body

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	cfg, err := config.LoadConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load project config", err.Error())
		return
	}

	// Find the command
	var cmdConfig *config.Command
	for _, cmd := range cfg.Commands {
		if cmd.Name == cmdName {
			cmdConfig = cmd
			break
		}
	}

	if cmdConfig == nil {
		writeError(w, http.StatusNotFound, "Command not found", cmdName)
		return
	}

	// Determine working directory
	workDir := ref.Path
	if req.FeatureName != "" {
		workDir = filepath.Join(ref.Path, "trees", req.FeatureName)
	}

	s.broadcast(WSMessage{
		Type:      "progress",
		Operation: "command",
		Message:   "Running command: " + cmdName,
	})

	// Build the command path
	cmdPath := cmdConfig.Command
	if !filepath.IsAbs(cmdPath) {
		cmdPath = filepath.Join(ref.Path, cmdPath)
	}

	// Execute the command
	cmd := exec.Command(cmdPath)
	cmd.Dir = workDir

	// Set up environment
	cmd.Env = append(os.Environ(),
		"RAMP_PROJECT_DIR="+ref.Path,
		"RAMP_TREES_DIR="+filepath.Join(ref.Path, "trees"),
	)
	if req.FeatureName != "" {
		cmd.Env = append(cmd.Env, "RAMP_WORKTREE_NAME="+req.FeatureName)
	}

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.broadcast(WSMessage{
			Type:      "error",
			Operation: "command",
			Message:   "Failed to capture stdout: " + err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "Failed to run command", err.Error())
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.broadcast(WSMessage{
			Type:      "error",
			Operation: "command",
			Message:   "Failed to capture stderr: " + err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "Failed to run command", err.Error())
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		s.broadcast(WSMessage{
			Type:      "error",
			Operation: "command",
			Message:   "Failed to start command: " + err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "Failed to run command", err.Error())
		return
	}

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			s.broadcast(WSMessage{
				Type:      "output",
				Operation: "command",
				Message:   scanner.Text(),
			})
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			s.broadcast(WSMessage{
				Type:      "output",
				Operation: "command",
				Message:   "[stderr] " + scanner.Text(),
			})
		}
	}()

	// Wait for command to finish
	err = cmd.Wait()
	if err != nil {
		s.broadcast(WSMessage{
			Type:      "error",
			Operation: "command",
			Message:   "Command failed: " + err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "Command failed", err.Error())
		return
	}

	s.broadcast(WSMessage{
		Type:      "complete",
		Operation: "command",
		Message:   "Command completed successfully",
	})

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "Command completed"})
}
