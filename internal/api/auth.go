package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
	"github.com/pulak-ranjan/kumomta-ui/internal/validation"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code,omitempty"`
}

type authResponse struct {
	Token        string `json:"token,omitempty"`
	Email        string `json:"email"`
	Requires2FA  bool   `json:"requires_2fa,omitempty"`
	TempToken    string `json:"temp_token,omitempty"`
}

type setup2FARequest struct {
	Password string `json:"password"`
}

type setup2FAResponse struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

type verify2FARequest struct {
	Code string `json:"code"`
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func validatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasLetter := false
	hasNumber := false
	for _, c := range password {
		if unicode.IsLetter(c) {
			hasLetter = true
		}
		if unicode.IsDigit(c) {
			hasNumber = true
		}
	}
	return hasLetter && hasNumber
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}
	ip = r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// POST /api/auth/register
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	v := validation.New()
	v.Required("email", req.Email).Email("email", req.Email)
	v.Required("password", req.Password).MinLength("password", req.Password, 8)

	if !v.Valid() {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"errors": v.Errors()})
		return
	}

	if !validatePassword(req.Password) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must have at least 1 letter and 1 number"})
		return
	}

	count, _ := s.Store.AdminCount()
	if count > 0 {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin already exists"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	admin := &models.AdminUser{
		Email:        req.Email,
		PasswordHash: string(hash),
		Theme:        "dark", // Default theme
	}

	if err := s.Store.CreateAdmin(admin); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create admin"})
		return
	}

	token := generateToken()
	ip := getClientIP(r)
	userAgent := r.UserAgent()

	if err := s.Store.CreateSession(admin.ID, token, ip, userAgent, 7*24*time.Hour); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, Email: admin.Email})
}

// POST /api/auth/login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	v := validation.New()
	v.Required("email", req.Email).Email("email", req.Email)
	v.Required("password", req.Password)

	if !v.Valid() {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"errors": v.Errors()})
		return
	}

	admin, err := s.Store.GetAdminByEmail(req.Email)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to find user"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	// Check if 2FA is enabled
	if admin.TwoFactorEnabled && admin.TwoFactorSecret != "" {
		// If no TOTP code provided, return that 2FA is required
		if req.TOTPCode == "" {
			tempToken := generateToken()
			// Store temp token with short expiry (5 minutes)
			s.Store.CreateSession(admin.ID, "2fa:"+tempToken, getClientIP(r), r.UserAgent(), 5*time.Minute)
			writeJSON(w, http.StatusOK, authResponse{
				Email:       admin.Email,
				Requires2FA: true,
				TempToken:   tempToken,
			})
			return
		}

		// Validate TOTP code
		if !core.ValidateTOTP(admin.TwoFactorSecret, req.TOTPCode) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid 2FA code"})
			return
		}
	}

	// Create full session
	token := generateToken()
	ip := getClientIP(r)
	userAgent := r.UserAgent()

	if err := s.Store.CreateSession(admin.ID, token, ip, userAgent, 7*24*time.Hour); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, Email: admin.Email})
}

// POST /api/auth/verify-2fa
func (s *Server) handleVerify2FA(w http.ResponseWriter, r *http.Request) {
	tempToken := r.Header.Get("X-Temp-Token")
	if tempToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing temp token"})
		return
	}

	var req verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Find admin by temp token
	admin, err := s.Store.GetAdminBySessionToken("2fa:" + tempToken)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired temp token"})
		return
	}

	// Validate TOTP
	if !core.ValidateTOTP(admin.TwoFactorSecret, req.Code) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid 2FA code"})
		return
	}

	// Delete temp session
	s.Store.DeleteSession("2fa:" + tempToken)

	// Create full session
	token := generateToken()
	ip := getClientIP(r)
	userAgent := r.UserAgent()

	if err := s.Store.CreateSession(admin.ID, token, ip, userAgent, 7*24*time.Hour); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, Email: admin.Email})
}

// GET /api/auth/me
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	admin := getAdminFromContext(r.Context())
	if admin == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"email":       admin.Email,
		"theme":       admin.Theme,
		"has_2fa":     admin.TwoFactorEnabled,
	})
}

// POST /api/auth/setup-2fa
func (s *Server) handleSetup2FA(w http.ResponseWriter, r *http.Request) {
	admin := getAdminFromContext(r.Context())
	if admin == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	var req setup2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid password"})
		return
	}

	// Generate new secret
	secret, err := core.GenerateTOTPSecret()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate secret"})
		return
	}

	// Store secret (but don't enable yet)
	admin.TwoFactorSecret = secret
	if err := s.Store.UpdateAdmin(admin); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save secret"})
		return
	}

	uri := core.GenerateTOTPURI(secret, admin.Email)

	writeJSON(w, http.StatusOK, setup2FAResponse{
		Secret: secret,
		URI:    uri,
	})
}

// POST /api/auth/enable-2fa
func (s *Server) handleEnable2FA(w http.ResponseWriter, r *http.Request) {
	admin := getAdminFromContext(r.Context())
	if admin == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	var req verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if admin.TwoFactorSecret == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "2FA not set up"})
		return
	}

	// Validate code
	if !core.ValidateTOTP(admin.TwoFactorSecret, req.Code) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid code"})
		return
	}

	// Enable 2FA
	admin.TwoFactorEnabled = true
	if err := s.Store.UpdateAdmin(admin); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to enable 2FA"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "2fa enabled"})
}

// POST /api/auth/disable-2fa
func (s *Server) handleDisable2FA(w http.ResponseWriter, r *http.Request) {
	admin := getAdminFromContext(r.Context())
	if admin == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	var req struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid password"})
		return
	}

	// Verify TOTP
	if admin.TwoFactorEnabled && !core.ValidateTOTP(admin.TwoFactorSecret, req.Code) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid 2FA code"})
		return
	}

	// Disable 2FA
	admin.TwoFactorEnabled = false
	admin.TwoFactorSecret = ""
	if err := s.Store.UpdateAdmin(admin); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to disable 2FA"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "2fa disabled"})
}

// POST /api/auth/theme
func (s *Server) handleSetTheme(w http.ResponseWriter, r *http.Request) {
	admin := getAdminFromContext(r.Context())
	if admin == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	var req struct {
		Theme string `json:"theme"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.Theme != "dark" && req.Theme != "light" && req.Theme != "system" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid theme"})
		return
	}

	admin.Theme = req.Theme
	if err := s.Store.UpdateAdmin(admin); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save theme"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "theme": req.Theme})
}

// GET /api/auth/sessions
func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	admin := getAdminFromContext(r.Context())
	if admin == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	sessions, err := s.Store.ListSessionsByAdmin(admin.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list sessions"})
		return
	}

	type sessionDTO struct {
		ID        uint      `json:"id"`
		DeviceIP  string    `json:"device_ip"`
		UserAgent string    `json:"user_agent"`
		CreatedAt time.Time `json:"created_at"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	out := make([]sessionDTO, 0, len(sessions))
	for _, sess := range sessions {
		// Don't expose temp 2FA sessions
		if strings.HasPrefix(sess.Token, "2fa:") {
			continue
		}
		out = append(out, sessionDTO{
			ID:        sess.ID,
			DeviceIP:  sess.DeviceIP,
			UserAgent: sess.UserAgent,
			CreatedAt: sess.CreatedAt,
			ExpiresAt: sess.ExpiresAt,
		})
	}

	writeJSON(w, http.StatusOK, out)
}
