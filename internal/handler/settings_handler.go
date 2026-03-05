package handler

import (
	"net/http"

	"github.com/ivmm/rpmmanager/internal/repository"
)

type SettingsHandler struct {
	repo *repository.SettingsRepo
}

func NewSettingsHandler(repo *repository.SettingsRepo) *SettingsHandler {
	return &SettingsHandler{repo: repo}
}

func (h *SettingsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.GetAll()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

// allowedSettingKeys defines which setting keys can be modified via the API.
var allowedSettingKeys = map[string]bool{
	"github_token":    true,
	"monitor_enabled": true,
	"monitor_interval": true,
	"notification_url": true,
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var updates map[string]string
	if err := decodeJSON(r, &updates); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	for k, v := range updates {
		if !allowedSettingKeys[k] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown setting key: " + k})
			return
		}
		if err := h.repo.Set(k, v); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update settings"})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
