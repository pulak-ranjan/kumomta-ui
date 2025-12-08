package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/mail"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// Token validity duration
const TokenValidityDuration = 7 * 24 * time.Hour // 7 days

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// validateEmail checks if the email format is valid
func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// validatePassword checks password strength (min 8 chars, at least 1 letter and 1 number)
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

// POST /api/auth/register
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Validate email format
	if !validateEmail(req.Email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email format"})
		return
	}

	// Validate password strength
	if !validatePassword(req.Password) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters with at least 1 letter and 1 number"})
		return
	}

	// Check if any admin exists
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

	token := generateToken()
	admin := &models.AdminUser{
		Email:        req.Email,
		PasswordHash: string(hash),
		APIToken:     token,
		TokenExpiry:  time.Now().Add(TokenValidityDuration),
	}

	if err := s.Store.CreateAdmin(admin); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create admin"})
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

	// Regenerate token on login with new expiry
	admin.APIToken = generateToken()
	admin.TokenExpiry = time.Now().Add(TokenValidityDuration)
	s.Store.UpdateAdmin(admin)

	writeJSON(w, http.StatusOK, authResponse{Token: admin.APIToken, Email: admin.Email})
}

// GET /api/auth/me
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	admin := getAdminFromContext(r.Context())
	if admin == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"email": admin.Email})
}
