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

// Server wraps dependencies for HTTP handlers.
type Server struct {
	Store *store.Store
}

// NewServer creates a new API server instance.
func NewServer(st *store.Store) *Server {
	return &Server{Store: st}
}

// Router builds the chi router with all routes and middleware.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// TODO: add auth middleware here later

 	// Routes
	r.Get("/api/status", s.handleStatus)
	r.Get("/api/settings", s.handleGetSettings)
	r.Post("/api/settings", s.handleSaveSettings)

	// Config preview and apply
	r.Get("/api/config/preview", s.handlePreviewConfig)
	r.Post("/api/config/apply", s.handleApplyConfig)

	// Domains + Senders
	r.Route("/api/domains", func(r chi.Router) {
		r.Get("/", s.handleListDomains)
		r.Post("/", s.handleSaveDomain)

		r.Route("/{domainID}", func(r chi.Router) {
			r.Get("/", s.handleGetDomain)
			r.Delete("/", s.handleDeleteDomain)

			r.Get("/senders", s.handleListSenders)
			r.Post("/senders", s.handleSaveSender)
		})
	})

	// Delete a sender by ID
	r.Delete("/api/senders/{senderID}", s.handleDeleteSenderByID)



	return r
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// ----------------------
// Status
// ----------------------

// handleStatus returns basic service health info.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"api":      "ok",
		"kumomta":  serviceStatus("kumomta"),
		"dovecot":  serviceStatus("dovecot"),
		"fail2ban": serviceStatus("fail2ban"),
	}
	writeJSON(w, http.StatusOK, resp)
}

// serviceStatus checks systemd status of a service.
func serviceStatus(name string) string {
	cmd := exec.Command("systemctl", "is-active", "--quiet", name)
	if err := cmd.Run(); err != nil {
		return "inactive"
	}
	return "active"
}

// ----------------------
// Settings
// ----------------------

// settingsDTO is what the API exposes to the UI.
// Note: RelayIPs is a generic list of authorized relay IPs (CSV),
// can be MailWizz, any ESP, or custom app IPs.
type settingsDTO struct {
	MainHostname string `json:"main_hostname"`
	MainServerIP string `json:"main_server_ip"`

	// Comma-separated list of relay/allowed IPs
	// Example: "10.0.0.5,192.168.1.10"
	RelayIPs string `json:"relay_ips"`

	AIProvider string `json:"ai_provider"`
	AIAPIKey   string `json:"ai_api_key,omitempty"` // write-only from client
}

// handleGetSettings returns the current app settings (if any).
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.Store.GetSettings()
	if err != nil {
		// If not found, return an empty settings object instead of error.
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
		RelayIPs:     st.MailWizzIP, // internal field name, but semantics = relay IPs
		AIProvider:   st.AIProvider,
		// AIAPIKey intentionally not returned
	}

	writeJSON(w, http.StatusOK, dto)
}

// handleSaveSettings creates or updates app-level settings.
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
	// Even though field name is MailWizzIP, it's actually generic "relay IPs"
	st.MailWizzIP = dto.RelayIPs
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

	// Never echo back the AI key
	dto.AIAPIKey = ""
	writeJSON(w, http.StatusOK, dto)
}
