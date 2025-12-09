package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

type Server struct {
	Store  *store.Store
	Router chi.Router
}

type contextKey string

const adminContextKey contextKey = "admin"

func NewServer(st *store.Store) *Server {
	s := &Server{Store: st}
	s.Router = s.routes()
	return s
}

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Temp-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes
	r.Post("/api/auth/register", s.handleRegister)
	r.Post("/api/auth/login", s.handleLogin)
	r.Post("/api/auth/verify-2fa", s.handleVerify2FA)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		// Auth & Profile
		r.Get("/api/auth/me", s.handleMe)
		r.Post("/api/auth/logout", s.handleLogout)
		r.Post("/api/auth/setup-2fa", s.handleSetup2FA)
		r.Post("/api/auth/enable-2fa", s.handleEnable2FA)
		r.Post("/api/auth/disable-2fa", s.handleDisable2FA)
		r.Post("/api/auth/theme", s.handleSetTheme)
		r.Get("/api/auth/sessions", s.handleListSessions)

		// Dashboard
		r.Get("/api/dashboard/stats", s.handleGetDashboardStats)

		// Settings
		r.Get("/api/settings", s.handleGetSettings)
		r.Post("/api/settings", s.handleSetSettings)

		// Domains
		r.Get("/api/domains", s.handleListDomains)
		r.Post("/api/domains", s.handleCreateDomain)
		r.Get("/api/domains/{id}", s.handleGetDomain)
		r.Put("/api/domains/{id}", s.handleUpdateDomain)
		r.Delete("/api/domains/{id}", s.handleDeleteDomain)

		// Senders
		r.Get("/api/domains/{domainID}/senders", s.handleListSenders)
		r.Post("/api/domains/{domainID}/senders", s.handleCreateSender)
		r.Get("/api/senders/{id}", s.handleGetSender)
		r.Put("/api/senders/{id}", s.handleUpdateSender)
		r.Delete("/api/senders/{id}", s.handleDeleteSender)
		r.Post("/api/domains/{domainID}/senders/{id}/setup", s.handleSetupSender)

		// Bounce Accounts (Aligned with Frontend)
		r.Get("/api/bounces", s.handleListBounce)
		r.Post("/api/bounces", s.handleSaveBounceAccount) // create or update
		r.Delete("/api/bounces/{bounceID}", s.handleDeleteBounceAccount)
		r.Post("/api/bounces/apply", s.handleApplyBounceAccounts)

		// System IPs (Aligned with Frontend)
		r.Get("/api/system/ips", s.handleListIPs)
		r.Post("/api/system/ips", s.handleAddIP) // handles single add if body matches, or create separate if needed
		r.Delete("/api/system/ips/{id}", s.handleDeleteIP)
		// Frontend might use /api/system/ips for add, verify if IPsPage uses specific route
		// Note: Frontend IPsPage uses POST /api/system/ips for CIDR/List adds.
		// We should route that to handleBulkAddIPs or similar if logic differs, 
		// but typically we can check body. For now let's map:
		r.Post("/api/system/ips/bulk", s.handleBulkAddIPs) // if frontend calls this
		r.Post("/api/system/ips/cidr", s.handleAddIPsByCIDR)
		r.Post("/api/system/ips/detect", s.handleDetectIPs)

		// DKIM (Aligned with Frontend)
		r.Get("/api/dkim/records", s.handleListDKIM)
		r.Post("/api/dkim/generate", s.handleGenerateDKIM)

		// DMARC
		r.Get("/api/dmarc/{domainID}", s.handleGetDMARC)
		r.Post("/api/dmarc/{domainID}", s.handleSetDMARC)
		r.Get("/api/dns/{domainID}", s.handleGetAllDNS)

		// Stats
		r.Get("/api/stats/domains", s.handleGetDomainStats)
		r.Get("/api/stats/domains/{domain}", s.handleGetSingleDomainStats)
		r.Get("/api/stats/summary", s.handleGetStatsSummary)
		r.Post("/api/stats/refresh", s.handleRefreshStats)

		// Queue
		r.Get("/api/queue", s.handleGetQueue)
		r.Get("/api/queue/stats", s.handleGetQueueStats)
		r.Delete("/api/queue/{id}", s.handleDeleteQueueMessage)
		r.Post("/api/queue/flush", s.handleFlushQueue)

		// Webhooks
		r.Get("/api/webhooks/settings", s.handleGetWebhookSettings)
		r.Post("/api/webhooks/settings", s.handleSetWebhookSettings)
		r.Post("/api/webhooks/test", s.handleTestWebhook)
		r.Get("/api/webhooks/logs", s.handleGetWebhookLogs)
		r.Post("/api/webhooks/check-bounces", s.handleCheckBounces)

		// Config (Missing in original)
		r.Get("/api/config/preview", s.handlePreviewConfig)
		r.Post("/api/config/apply", s.handleApplyConfig)

		// Logs (Missing in original)
		r.Get("/api/logs/kumomta", s.handleLogsKumo)
		r.Get("/api/logs/dovecot", s.handleLogsDovecot)
		r.Get("/api/logs/fail2ban", s.handleLogsFail2ban)

		// System
		r.Get("/api/system/health", s.handleSystemHealth)
		r.Get("/api/system/services", s.handleSystemServices)
		r.Get("/api/system/ports", s.handleSystemPorts)
		r.Post("/api/system/ai-analyze", s.handleAIAnalyze)

		// Bulk Import
		r.Post("/api/import/csv", s.handleCSVImport)
	})

	return r
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
			return
		}

		admin, err := s.Store.GetAdminBySessionToken(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
			return
		}

		ctx := context.WithValue(r.Context(), adminContextKey, admin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getAdminFromContext(ctx context.Context) *models.AdminUser {
	admin, ok := ctx.Value(adminContextKey).(*models.AdminUser)
	if !ok {
		return nil
	}
	return admin
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// POST /api/auth/logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	s.Store.DeleteSession(token)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}
