package api

import (
	"encoding/json"
	"net/http"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

type settingsDTO struct {
	MainHostname string `json:"main_hostname"`
	MainServerIP string `json:"main_server_ip"`
	RelayIPs     string `json:"relay_ips"`
	AIProvider   string `json:"ai_provider"`
	AIAPIKey     string `json:"ai_api_key,omitempty"`
}

// GET /api/settings
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.Store.GetSettings()
	if err != nil && err != store.ErrNotFound {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load settings"})
		return
	}

	if st == nil {
		writeJSON(w, http.StatusOK, settingsDTO{})
		return
	}

	writeJSON(w, http.StatusOK, settingsDTO{
		MainHostname: st.MainHostname,
		MainServerIP: st.MainServerIP,
		RelayIPs:     st.MailWizzIP,
		AIProvider:   st.AIProvider,
		// AIAPIKey intentionally omitted - write-only
	})
}

// POST /api/settings
func (s *Server) handleSetSettings(w http.ResponseWriter, r *http.Request) {
	var dto settingsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	existing, err := s.Store.GetSettings()
	if err != nil && err != store.ErrNotFound {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load settings"})
		return
	}

	if existing == nil {
		existing = &models.AppSettings{}
	}

	existing.MainHostname = dto.MainHostname
	existing.MainServerIP = dto.MainServerIP
	existing.MailWizzIP = dto.RelayIPs
	existing.AIProvider = dto.AIProvider

	if dto.AIAPIKey != "" {
		enc, err := core.Encrypt(dto.AIAPIKey)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encrypt key"})
			return
		}
		existing.AIAPIKey = enc
	}

	if err := s.Store.UpsertSettings(existing); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
