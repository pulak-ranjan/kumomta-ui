package api

import (
	"encoding/json"
	"net/http"
	"os/exec"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

type Server struct {
	Store *store.Store
}

func NewServer(st *store.Store) *Server {
	return &Server{Store: st}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// TODO: add auth middleware later

	// Routes
	r.Get("/api/status", s.handleStatus)
	r.Get("/api/settings", s.handleGetSettings)
	r.Post("/api/settings", s.handleSaveSettings)

	// TODO: /api/domains, /api/senders, /api/apply, /api/dns, /api/ai-summary

	return r
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// ------------ Status ------------

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"api":      "ok",
		"kumomta":  serviceStatus("kumomta"),
		"dovecot":  serviceStatus("dovecot"),
		"fail2ban": serviceStatus("fail2ban"),
	}
	writeJSON(w, http.StatusOK, resp)
}

func serviceStatus(name string) string {
	cmd := exec.Command("systemctl", "is-active", "--quiet", name)
	if err := cmd.Run(); err != nil {
		return "inactive"
	}
	return "active"
}

// ------------ Settings ------------

// DTO so we don't leak AIAPIKey back in responses
type settingsDTO struct {
	MainHostname string `json:"main_hostname"`
	MainServerIP string `json:"main_server_ip"`
	MailWizzIP   string `json:"mailwizz_ip"`

	AIProvider string `json:"ai_provider"`
	AIAPIKey   string `json:"ai_api_key,omitempty"` // write-only from client
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.Store.GetSettings()
	if err != nil {
		// If not found, just return empty settings object
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusOK, settingsDTO{})
			return
		}
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load settings"})
		return
	}

	dto := settingsDTO{
		MainHostname: st.MainHostname,
		MainServerIP: st.MainServerIP,
		MailWizzIP:   st.MailWizzIP,
		AIProvider:   st.AIProvider,
		// AIAPIKey intentionally not returned
	}

	writeJSON(w, http.StatusOK, dto)
}

func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	var dto settingsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Load existing or create new
	st, err := s.Store.GetSettings()
	if err != nil && err != store.ErrNotFound {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load existing settings"})
		return
	}
	if st == nil {
		st = &models.AppSettings{}
	}

	// Update fields from DTO
	st.MainHostname = dto.MainHostname
	st.MainServerIP = dto.MainServerIP
	st.MailWizzIP = dto.MailWizzIP
	st.AIProvider = dto.AIProvider

	// Only overwrite AIAPIKey if user sent something non-empty
	if dto.AIAPIKey != "" {
		st.AIAPIKey = dto.AIAPIKey
	}

	if err := s.Store.UpsertSettings(st); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save settings"})
		return
	}

	// Return sanitized DTO
	dto.AIAPIKey = ""
	writeJSON(w, http.StatusOK, dto)
}
