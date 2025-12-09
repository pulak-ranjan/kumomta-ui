package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// ----------------------
// IP Management
// ----------------------

type addIPRequest struct {
	CIDR string `json:"cidr"` // e.g. "1.2.3.0/24"
	List string `json:"list"` // newline separated IPs
}

// GET /api/system/ips
// Returns IPs from DB + detected interfaces
func (s *Server) handleGetSystemIPs(w http.ResponseWriter, r *http.Request) {
	// 1. Get DB IPs
	dbIPs, _ := s.Store.ListSystemIPs()

	// 2. Detect Interface IPs (for convenience)
	ifaceIPs := detectInterfaceIPs()

	// Merge unique
	unique := make(map[string]bool)
	for _, ip := range dbIPs {
		unique[ip.Value] = true
	}
	for _, ip := range ifaceIPs {
		unique[ip] = true
	}

	var out []string
	for ip := range unique {
		out = append(out, ip)
	}

	writeJSON(w, http.StatusOK, out)
}

// POST /api/system/ips
func (s *Server) handleAddIPs(w http.ResponseWriter, r *http.Request) {
	var req addIPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	var newIPs []models.SystemIP

	// Process CIDR
	if req.CIDR != "" {
		ip, ipnet, err := net.ParseCIDR(strings.TrimSpace(req.CIDR))
		if err == nil {
			for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
				// Filter standard lengths, skip network/broadcast implies
				// user logic. We just add valid IPs.
				newIPs = append(newIPs, models.SystemIP{
					Value:   ip.String(),
					Netmask: req.CIDR,
				})
			}
		}
	}

	// Process List
	if req.List != "" {
		lines := strings.Split(req.List, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if net.ParseIP(line) != nil {
				newIPs = append(newIPs, models.SystemIP{Value: line})
			}
		}
	}

	if len(newIPs) > 0 {
		if err := s.Store.CreateSystemIPs(newIPs); err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save IPs"})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "added": strconv.Itoa(len(newIPs))})
}

// inc increments an IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func detectInterfaceIPs() []string {
	var ips []string
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips
}

// ----------------------
// Dashboard Stats
// ----------------------

type dashboardStatsDTO struct {
	Domains int64 `json:"domains"`
	Senders int64 `json:"senders"`

	// Server Health
	CPULoad  string `json:"cpu_load"`
	RAMUsage string `json:"ram_usage"` // "Used / Total MB"

	// Kumo Health
	KumoStatus string `json:"kumo_status"`
}

// GET /api/dashboard/stats
func (s *Server) handleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	dCount, _ := s.Store.CountDomains()
	sCount, _ := s.Store.CountSenders()

	stats := dashboardStatsDTO{
		Domains: dCount,
		Senders: sCount,
	}

	// Load Avg (Linux only)
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) > 0 {
			stats.CPULoad = parts[0] + " (1m)"
		}
	} else {
		stats.CPULoad = "N/A"
	}

	// RAM Usage (Linux)
	stats.RAMUsage = getSystemRAM()

	// Check Kumo status
	stats.KumoStatus = "Running"
	// Optional: Check port 8000
	if _, err := net.DialTimeout("tcp", "127.0.0.1:8000", 1*time.Second); err != nil {
		stats.KumoStatus = "Unreachable"
	}

	writeJSON(w, http.StatusOK, stats)
}

func getSystemRAM() string {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "N/A"
	}
	lines := strings.Split(string(data), "\n")
	var total, free, available int
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.Atoi(fields[1]) // kB
		switch fields[0] {
		case "MemTotal:":
			total = val
		case "MemFree:":
			free = val
		case "MemAvailable:":
			available = val
		}
	}
	if available == 0 {
		available = free
	} // fallback
	used := total - available
	return fmt.Sprintf("%d / %d MB", used/1024, total/1024)
}

// ----------------------
// AI Insights
// ----------------------

// POST /api/dashboard/ai
func (s *Server) handleAIInsights(w http.ResponseWriter, r *http.Request) {
	st, err := s.Store.GetSettings()
	if err != nil || st == nil || st.AIAPIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "AI not configured in settings"})
		return
	}

	// Gather context
	dCount, _ := s.Store.CountDomains()
	sCount, _ := s.Store.CountSenders()
	load := "Unknown"
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		load = string(data)
	}

	systemContext := fmt.Sprintf(
		"I am running a KumoMTA server. Stats: %d domains, %d senders. System Load: %s. RAM: %s. Give me a brief health check summary and 2 optimization tips.",
		dCount, sCount, load, getSystemRAM(),
	)

	// Determine URL based on provider
	apiUrl := "https://api.openai.com/v1/chat/completions"
	model := "gpt-3.5-turbo"
	if st.AIProvider == "deepseek" {
		apiUrl = "https://api.deepseek.com/v1/chat/completions"
		model = "deepseek-chat"
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful MTA server administrator assistant."},
			{"role": "user", "content": systemContext},
		},
	})

	aiReq, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(reqBody))
	aiReq.Header.Set("Content-Type", "application/json")
	aiReq.Header.Set("Authorization", "Bearer "+st.AIAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(aiReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to call AI: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "AI Provider Error: " + string(body)})
		return
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Extract text (simplified)
	content := "No content"
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				content = msg["content"].(string)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"insight": content})
}
