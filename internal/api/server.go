package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// context key for admin user
type ctxKeyAdmin struct{}

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

	// Global middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)

	// Public routes (no auth)
	r.Get("/api/status", s.handleStatus)

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", s.handleRegister)
		r.Post("/login", s.handleLogin)
	})

	// Protected routes (auth required)
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		r.Get("/api/auth/me", s.handleMe)

		// Settings
		r.Get("/api/settings", s.handleGetSettings)
		r.Post("/api/settings", s.handleSaveSettings)

		// Config preview and apply
		r.Get("/api/config/preview", s.handlePreviewConfig)
		r.Post("/api/config/apply", s.handleApplyConfig)

		// DKIM
		r.Get("/api/dkim/records", s.handleListDKIMRecords)
		r.Post("/api/dkim/generate", s.handleGenerateDKIM)

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

		// Bounce accounts
		r.Get("/api/bounces", s.handleListBounceAccounts)
		r.Post("/api/bounces", s.handleSaveBounceAccount)
		r.Delete("/api/bounces/{bounceID}", s.handleDeleteBounceAccount)
		r.Post("/api/bounces/apply", s.handleApplyBounceAccounts)

		// Logs
		r.Get("/api/logs/kumomta", s.handleLogsKumo)
		r.Get("/api/logs/dovecot", s.handleLogsDovecot)
		r.Get("/api/logs/fail2ban", s.handleLogsFail2ban)
	})

	fileServer := http.FileServer(http.Dir("./web/dist"))
	r.Handle("/*", fileServer)

	return r
}

// securityHeaders adds security headers to all responses
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// ----------------------
// Auth middleware
// ----------------------

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authorization header"})
			return
		}
		token := strings.TrimSpace(parts[1])
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "empty token"})
			return
		}

		admin, err := s.Store.GetAdminByToken(token)
		if err != nil {
			if err == store.ErrNotFound {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to validate token"})
			return
		}

		// Check if token has expired
		if !admin.TokenExpiry.IsZero() && admin.TokenExpiry.Before(time.Now()) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "token expired, please login again"})
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyAdmin{}, admin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getAdminFromContext(ctx context.Context) *models.AdminUser {
	val := ctx.Value(ctxKeyAdmin{})
	if val == nil {
		return nil
	}
	if u, ok := val.(*models.AdminUser); ok {
		return u
	}
	return nil
}

// ----------------------
// Status
// ----------------------

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
