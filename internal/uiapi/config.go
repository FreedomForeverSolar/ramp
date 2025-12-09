package uiapi

import (
	"encoding/json"
	"net/http"

	"ramp/internal/config"

	"github.com/gorilla/mux"
)

// GetConfigStatus checks if a project needs configuration
// Returns the prompts if config is needed
func (s *Server) GetConfigStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	// Load project config to get prompts
	cfg, err := config.LoadConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load project config", err.Error())
		return
	}

	// Check if prompts are defined
	if len(cfg.Prompts) == 0 {
		// No prompts defined, config not needed
		writeJSON(w, http.StatusOK, ConfigStatusResponse{
			NeedsConfig: false,
			Prompts:     nil,
		})
		return
	}

	// Convert prompts to API format (always return prompts for editing)
	prompts := make([]Prompt, len(cfg.Prompts))
	for i, p := range cfg.Prompts {
		options := make([]PromptOption, len(p.Options))
		for j, o := range p.Options {
			options[j] = PromptOption{
				Value: o.Value,
				Label: o.Label,
			}
		}
		prompts[i] = Prompt{
			Name:     p.Name,
			Question: p.Question,
			Options:  options,
			Default:  p.Default,
		}
	}

	// Check if local.yaml exists
	localCfg, err := config.LoadLocalConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load local config", err.Error())
		return
	}

	// needsConfig is true only if local config doesn't exist
	// Always return prompts so user can edit existing config
	needsConfig := localCfg == nil

	writeJSON(w, http.StatusOK, ConfigStatusResponse{
		NeedsConfig: needsConfig,
		Prompts:     prompts,
	})
}

// GetConfig returns the current local config for a project
func (s *Server) GetConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	// Load local config
	localCfg, err := config.LoadLocalConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load local config", err.Error())
		return
	}

	preferences := make(map[string]string)
	if localCfg != nil && localCfg.Preferences != nil {
		preferences = localCfg.Preferences
	}

	writeJSON(w, http.StatusOK, ConfigResponse{
		Preferences: preferences,
	})
}

// SaveConfig saves the local config for a project
func (s *Server) SaveConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req SaveConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	// Create and save local config
	localCfg := &config.LocalConfig{
		Preferences: req.Preferences,
	}

	if err := config.SaveLocalConfig(localCfg, ref.Path); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save config", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Configuration saved",
	})
}

// ResetConfig deletes the local config for a project
func (s *Server) ResetConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	// Delete local.yaml by saving nil/empty config
	// Actually, we should delete the file entirely
	if err := config.DeleteLocalConfig(ref.Path); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reset config", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Configuration reset",
	})
}
