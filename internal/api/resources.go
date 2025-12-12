package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// ----------------------
// Bounce Accounts
// ----------------------

// GET /api/bounce
func (s *Server) handleListBounce(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.Store.ListBounceAccounts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list bounce accounts"})
		return
	}
	if accounts == nil {
		accounts = []models.BounceAccount{}
	}

	// Filter by domain if requested
	domain := r.URL.Query().Get("domain")
	if domain != "" {
		filtered := make([]models.BounceAccount, 0)
		for _, a := range accounts {
			if a.Domain == domain {
				filtered = append(filtered, a)
			}
		}
		accounts = filtered
	}

	writeJSON(w, http.StatusOK, accounts)
}

// POST /api/bounce
func (s *Server) handleCreateBounce(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Domain   string `json:"domain"`
		Notes    string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.Username == "" || req.Domain == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and domain required"})
		return
	}

	if err := core.CreateBounceAccountWithPassword(req.Username, req.Domain, req.Password, req.Notes, s.Store); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"status":   "created",
		"username": req.Username,
		"email":    req.Username + "@" + req.Domain,
	})
}

// DELETE /api/bounce/{id}
func (s *Server) handleDeleteBounce(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := s.Store.DeleteBounceAccount(uint(id)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete bounce account"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ----------------------
// System IPs
// ----------------------

// GET /api/system/ips
func (s *Server) handleListIPs(w http.ResponseWriter, r *http.Request) {
	ips, err := s.Store.ListSystemIPs()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list IPs"})
		return
	}
	if ips == nil {
		ips = []models.SystemIP{}
	}

	// Check status against OS
	activeMap := core.GetActiveIPsMap()
	for i := range ips {
		if activeMap[ips[i].Value] {
			ips[i].IsActive = true
		}
	}

	writeJSON(w, http.StatusOK, ips)
}

// POST /api/system/ips/configure
// Executes 'ip addr add' on the server
func (s *Server) handleConfigureIP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IP        string `json:"ip"`
		Netmask   string `json:"netmask"`
		Interface string `json:"interface"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.IP == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ip required"})
		return
	}

	if err := core.ConfigureSystemIP(req.IP, req.Netmask, req.Interface); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Log action
	go s.WS.SendAuditLog("Configure IP", "Added IP to interface: "+req.IP, s.getUser(r))

	writeJSON(w, http.StatusOK, map[string]string{"status": "configured"})
}

// POST /api/system/ips
func (s *Server) handleAddIP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value     string `json:"value"`
		Netmask   string `json:"netmask"`
		Interface string `json:"interface"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.Value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "value required"})
		return
	}

	ip := &models.SystemIP{
		Value:     req.Value,
		Netmask:   req.Netmask,
		Interface: req.Interface,
		CreatedAt: time.Now(),
	}

	if err := s.Store.CreateSystemIP(ip); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create IP"})
		return
	}

	writeJSON(w, http.StatusCreated, ip)
}

// POST /api/system/ips/bulk
func (s *Server) handleBulkAddIPs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IPs []string `json:"ips"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	ips := make([]models.SystemIP, 0, len(req.IPs))
	for _, ipVal := range req.IPs {
		if ipVal != "" {
			ips = append(ips, models.SystemIP{
				Value:     strings.TrimSpace(ipVal),
				CreatedAt: time.Now(),
			})
		}
	}

	if err := s.Store.CreateSystemIPs(ips); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to add IPs"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int{"added": len(ips)})
}

// POST /api/system/ips/cidr
func (s *Server) handleAddIPsByCIDR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CIDR string `json:"cidr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	ipList, err := core.ExpandCIDR(req.CIDR)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	ips := make([]models.SystemIP, 0, len(ipList))
	for _, ipVal := range ipList {
		ips = append(ips, models.SystemIP{
			Value:     ipVal,
			CreatedAt: time.Now(),
		})
	}

	if err := s.Store.CreateSystemIPs(ips); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to add IPs"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"added": len(ips),
		"ips":   ipList,
	})
}

// POST /api/system/ips/detect
func (s *Server) handleDetectIPs(w http.ResponseWriter, r *http.Request) {
	detected := core.DetectServerIPs()

	ips := make([]models.SystemIP, 0, len(detected))
	for _, d := range detected {
		ips = append(ips, models.SystemIP{
			Value:     d.IP,
			Interface: d.Interface,
			CreatedAt: time.Now(),
		})
	}

	if err := s.Store.CreateSystemIPs(ips); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save IPs"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"detected": detected,
		"added":    len(ips),
	})
}

// DELETE /api/system/ips/{id}
func (s *Server) handleDeleteIP(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := s.Store.DeleteSystemIP(uint(id)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete IP"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
