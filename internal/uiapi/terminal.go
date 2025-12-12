package uiapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// OpenTerminal opens a terminal window at the specified path
func (s *Server) OpenTerminal(w http.ResponseWriter, r *http.Request) {
	var req OpenTerminalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate path exists
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		writeError(w, http.StatusBadRequest, "Path does not exist", req.Path)
		return
	}

	// Get terminal preference
	appConfig, err := LoadAppConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load app config", err.Error())
		return
	}

	terminalApp := appConfig.Preferences.TerminalApp
	if terminalApp == "" {
		terminalApp = "terminal" // default
	}

	// Open terminal based on preference
	if err := openTerminalApp(terminalApp, req.Path); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to open terminal", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "Terminal opened"})
}

// openTerminalApp opens a terminal application at the specified path
func openTerminalApp(terminalApp, path string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("terminal opening only supported on macOS")
	}

	var cmd *exec.Cmd

	switch terminalApp {
	case "terminal":
		// macOS Terminal.app
		script := fmt.Sprintf(`tell application "Terminal"
			activate
			do script "cd %q"
		end tell`, path)
		cmd = exec.Command("osascript", "-e", script)

	case "iterm":
		// iTerm2
		script := fmt.Sprintf(`tell application "iTerm"
			activate
			create window with default profile
			tell current session of current window
				write text "cd %q"
			end tell
		end tell`, path)
		cmd = exec.Command("osascript", "-e", script)

	case "warp":
		// Warp terminal - use open command with directory
		cmd = exec.Command("open", "-a", "Warp", path)

	default:
		// Custom command - replace $PATH with the actual path
		if strings.Contains(terminalApp, "$PATH") {
			customCmd := strings.ReplaceAll(terminalApp, "$PATH", path)
			cmd = exec.Command("sh", "-c", customCmd)
		} else {
			// Assume it's an app name, use open -a
			cmd = exec.Command("open", "-a", terminalApp, path)
		}
	}

	// Use Start() instead of Run() - we just want to launch the terminal,
	// not wait for it to exit. Run() would wait and potentially return errors
	// even when the terminal opened successfully.
	return cmd.Start()
}

// GetAppSettings returns the current app settings
func (s *Server) GetAppSettings(w http.ResponseWriter, r *http.Request) {
	appConfig, err := LoadAppConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load app config", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, AppSettingsResponse{
		TerminalApp:           appConfig.Preferences.TerminalApp,
		LastSelectedProjectID: appConfig.Preferences.LastSelectedProjectID,
		Theme:                 appConfig.Preferences.Theme,
	})
}

// SaveAppSettings saves app settings
// Only updates fields that are provided (non-empty) in the request
func (s *Server) SaveAppSettings(w http.ResponseWriter, r *http.Request) {
	var req SaveAppSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	appConfig, err := LoadAppConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load app config", err.Error())
		return
	}

	// Only update fields that are provided
	if req.TerminalApp != "" {
		appConfig.Preferences.TerminalApp = req.TerminalApp
	}
	if req.LastSelectedProjectID != "" {
		appConfig.Preferences.LastSelectedProjectID = req.LastSelectedProjectID
	}
	if req.Theme != "" {
		appConfig.Preferences.Theme = req.Theme
	}

	if err := SaveAppConfig(appConfig); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save app config", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "Settings saved"})
}
