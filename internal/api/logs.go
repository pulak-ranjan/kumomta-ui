package api

import (
	"net/http"
	"os/exec"
	"strconv"
)

// GET /api/logs/kumomta?lines=100
func (s *Server) handleLogsKumo(w http.ResponseWriter, r *http.Request) {
	s.handleLogsForService(w, r, "kumomta")
}

// GET /api/logs/dovecot?lines=100
func (s *Server) handleLogsDovecot(w http.ResponseWriter, r *http.Request) {
	s.handleLogsForService(w, r, "dovecot")
}

// GET /api/logs/fail2ban?lines=100
func (s *Server) handleLogsFail2ban(w http.ResponseWriter, r *http.Request) {
	s.handleLogsForService(w, r, "fail2ban")
}

func (s *Server) handleLogsForService(w http.ResponseWriter, r *http.Request, service string) {
	linesStr := r.URL.Query().Get("lines")
	if linesStr == "" {
		linesStr = "100"
	}
	n, err := strconv.Atoi(linesStr)
	if err != nil || n <= 0 {
		n = 100
	}

	cmd := exec.Command("journalctl", "-u", service, "-n", strconv.Itoa(n), "--no-pager")
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to read logs",
			"info":  string(out),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"service": service,
		"logs":    string(out),
	})
}
