package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ----------------------
// Dashboard
// ----------------------

// GET /api/dashboard/stats
func (s *Server) handleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})

	// 1. Database Counts
	dCount, _ := s.Store.CountDomains()
	sCount, _ := s.Store.CountSenders()
	stats["domains"] = dCount
	stats["senders"] = sCount

	// 2. CPU Load
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			stats["cpu_load"] = parts[0]
		}
	} else {
		stats["cpu_load"] = "0.00"
	}

	// 3. RAM Usage
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		memInfo := make(map[string]int64)
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				key := strings.TrimSuffix(parts[0], ":")
				val, _ := strconv.ParseInt(parts[1], 10, 64)
				memInfo[key] = val
			}
		}
		totalKB := memInfo["MemTotal"]
		availKB := memInfo["MemAvailable"]
		if totalKB > 0 {
			usedKB := totalKB - availKB
			pct := float64(usedKB) / float64(totalKB) * 100
			stats["ram_usage"] = fmt.Sprintf("%.1f%%", pct)
		} else {
			stats["ram_usage"] = "0%"
		}
	} else {
		stats["ram_usage"] = "N/A"
	}

	// 4. Service Status
	services := map[string]string{
		"kumomta":  "kumo_status",
		"dovecot":  "dovecot_status",
		"fail2ban": "f2b_status",
	}
	for svc, key := range services {
		cmd := exec.Command("systemctl", "is-active", svc)
		out, _ := cmd.Output()
		status := strings.TrimSpace(string(out))
		if status == "" {
			status = "unknown"
		}
		stats[key] = status
	}

	// 5. Open Ports (Simplified scan)
	ports := []int{25, 587, 465, 80, 443, 9000, 993}
	var openPorts []string
	for _, port := range ports {
		// Use ss to check if port is listening
		cmd := exec.Command("sh", "-c", fmt.Sprintf("ss -tlnp | grep ':%d '", port))
		if out, _ := cmd.Output(); len(out) > 0 {
			openPorts = append(openPorts, strconv.Itoa(port))
		}
	}
	stats["open_ports"] = strings.Join(openPorts, ", ")

	writeJSON(w, http.StatusOK, stats)
}

// ----------------------
// Settings (Moved here from settings.go to keep it consolidated if preferred, 
// OR keep in settings.go. Since we removed duplicates earlier, we leave them in settings.go
// and only implement system health/AI here.)
// ----------------------

// GET /api/system/health
func (s *Server) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{}
	// ... (Same implementation as previous, used for dedicated health checks)
	// CPU
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 { health["cpu_load_1m"], _ = strconv.ParseFloat(parts[0], 64) }
	}
	// RAM
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		mem := make(map[string]int64)
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 { mem[strings.TrimSuffix(parts[0], ":")], _ = strconv.ParseInt(parts[1], 10, 64) }
		}
		health["ram_total_mb"] = mem["MemTotal"] / 1024
		health["ram_available_mb"] = mem["MemAvailable"] / 1024
	}
	writeJSON(w, http.StatusOK, health)
}

// GET /api/system/services
func (s *Server) handleSystemServices(w http.ResponseWriter, r *http.Request) {
	services := []string{"kumomta", "dovecot", "fail2ban", "nginx", "postfix"}
	result := make(map[string]string)
	for _, svc := range services {
		cmd := exec.Command("systemctl", "is-active", svc)
		out, _ := cmd.Output()
		status := strings.TrimSpace(string(out))
		if status == "" { status = "unknown" }
		result[svc] = status
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /api/system/ports
func (s *Server) handleSystemPorts(w http.ResponseWriter, r *http.Request) {
	ports := []int{25, 587, 465, 80, 443, 9000, 993, 110}
	result := make(map[string]bool)
	for _, port := range ports {
		cmd := exec.Command("sh", "-c", fmt.Sprintf("ss -tlnp | grep ':%d '", port))
		out, _ := cmd.Output()
		result[strconv.Itoa(port)] = len(out) > 0
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/system/ai-analyze
func (s *Server) handleAIAnalyze(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type string `json:"type"` 
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Type = "logs"
	}

	settings, err := s.Store.GetSettings()
	if err != nil || settings == nil || settings.AIAPIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "AI not configured"})
		return
	}

	var prompt, context string
	if req.Type == "health" {
		cmd := exec.Command("df", "-h")
		out, _ := cmd.Output()
		context = "Disk Usage:\n" + string(out)
		prompt = "Analyze this server health data and provide insights. Be concise."
	} else {
		cmd := exec.Command("journalctl", "-u", "kumomta", "-n", "100", "--no-pager")
		out, _ := cmd.Output()
		context = string(out)
		prompt = "Analyze these KumoMTA logs. Identify issues. Be concise."
	}

	analysis, err := callAIAPI(settings.AIProvider, settings.AIAPIKey, prompt, context)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"analysis": analysis, "type": req.Type})
}

func callAIAPI(provider, apiKey, prompt, context string) (string, error) {
	// ... (Same helper as before)
	url := "https://api.openai.com/v1/chat/completions"
	model := "gpt-3.5-turbo"
	if provider == "deepseek" {
		url = "https://api.deepseek.com/chat/completions"
		model = "deepseek-chat"
	}

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt + "\n\nData:\n" + context},
		},
		"max_tokens": 500,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 { return "", fmt.Errorf("API error: %s", string(body)) }

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil { return "", err }
	
	choices, _ := result["choices"].([]interface{})
	if len(choices) > 0 {
		if c, ok := choices[0].(map[string]interface{}); ok {
			if m, ok := c["message"].(map[string]interface{}); ok {
				return m["content"].(string), nil
			}
		}
	}
	return "", fmt.Errorf("no content")
}
