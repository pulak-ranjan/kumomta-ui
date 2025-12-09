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

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// ----------------------
// Settings
// ----------------------

// GET /api/settings
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Store.GetSettings()
	if err != nil {
		// Return empty settings
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"main_hostname":    "",
			"main_server_ip":   "",
			"mailwizz_ip":      "",
			"ai_provider":      "",
			"ai_api_key":       "",
			"webhook_url":      "",
			"webhook_enabled":  false,
			"bounce_alert_pct": 5.0,
		})
		return
	}

	// Mask API key
	maskedKey := ""
	if settings.AIAPIKey != "" {
		if len(settings.AIAPIKey) > 8 {
			maskedKey = settings.AIAPIKey[:4] + "****" + settings.AIAPIKey[len(settings.AIAPIKey)-4:]
		} else {
			maskedKey = "****"
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":               settings.ID,
		"main_hostname":    settings.MainHostname,
		"main_server_ip":   settings.MainServerIP,
		"mailwizz_ip":      settings.MailWizzIP,
		"ai_provider":      settings.AIProvider,
		"ai_api_key":       maskedKey,
		"webhook_url":      settings.WebhookURL,
		"webhook_enabled":  settings.WebhookEnabled,
		"bounce_alert_pct": settings.BounceAlertPct,
	})
}

// POST /api/settings
func (s *Server) handleSetSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MainHostname   string  `json:"main_hostname"`
		MainServerIP   string  `json:"main_server_ip"`
		MailWizzIP     string  `json:"mailwizz_ip"`
		AIProvider     string  `json:"ai_provider"`
		AIAPIKey       string  `json:"ai_api_key"`
		WebhookURL     string  `json:"webhook_url"`
		WebhookEnabled bool    `json:"webhook_enabled"`
		BounceAlertPct float64 `json:"bounce_alert_pct"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	settings, _ := s.Store.GetSettings()
	if settings == nil {
		settings = &models.AppSettings{}
	}

	if req.MainHostname != "" {
		settings.MainHostname = req.MainHostname
	}
	if req.MainServerIP != "" {
		settings.MainServerIP = req.MainServerIP
	}
	settings.MailWizzIP = req.MailWizzIP

	if req.AIProvider != "" {
		settings.AIProvider = req.AIProvider
	}
	// Only update API key if not masked
	if req.AIAPIKey != "" && !strings.Contains(req.AIAPIKey, "****") {
		settings.AIAPIKey = req.AIAPIKey
	}

	settings.WebhookURL = req.WebhookURL
	settings.WebhookEnabled = req.WebhookEnabled
	if req.BounceAlertPct > 0 {
		settings.BounceAlertPct = req.BounceAlertPct
	}

	if err := s.Store.UpsertSettings(settings); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// ----------------------
// System
// ----------------------

// GET /api/system/health
func (s *Server) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{}

	// CPU Load
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			load, _ := strconv.ParseFloat(parts[0], 64)
			health["cpu_load_1m"] = load
		}
		if len(parts) >= 2 {
			load, _ := strconv.ParseFloat(parts[1], 64)
			health["cpu_load_5m"] = load
		}
		if len(parts) >= 3 {
			load, _ := strconv.ParseFloat(parts[2], 64)
			health["cpu_load_15m"] = load
		}
	}

	// RAM
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
		usedKB := totalKB - availKB
		health["ram_total_mb"] = totalKB / 1024
		health["ram_used_mb"] = usedKB / 1024
		health["ram_available_mb"] = availKB / 1024
		if totalKB > 0 {
			health["ram_used_pct"] = float64(usedKB) / float64(totalKB) * 100
		}
	}

	// Disk
	cmd := exec.Command("df", "-B1", "/")
	if out, err := cmd.Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) >= 2 {
			parts := strings.Fields(lines[1])
			if len(parts) >= 4 {
				total, _ := strconv.ParseInt(parts[1], 10, 64)
				used, _ := strconv.ParseInt(parts[2], 10, 64)
				avail, _ := strconv.ParseInt(parts[3], 10, 64)
				health["disk_total_gb"] = total / (1024 * 1024 * 1024)
				health["disk_used_gb"] = used / (1024 * 1024 * 1024)
				health["disk_available_gb"] = avail / (1024 * 1024 * 1024)
				if total > 0 {
					health["disk_used_pct"] = float64(used) / float64(total) * 100
				}
			}
		}
	}

	// Uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			secs, _ := strconv.ParseFloat(parts[0], 64)
			health["uptime_hours"] = secs / 3600
		}
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
		if status == "" {
			status = "unknown"
		}
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
		Type string `json:"type"` // "logs" or "health"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Type = "logs"
	}

	settings, err := s.Store.GetSettings()
	if err != nil || settings == nil || settings.AIAPIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "AI not configured"})
		return
	}

	var prompt string
	var context string

	if req.Type == "health" {
		// Gather health data
		var healthData strings.Builder
		if data, err := os.ReadFile("/proc/loadavg"); err == nil {
			healthData.WriteString("Load: " + string(data))
		}
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines[:10] {
				healthData.WriteString(line + "\n")
			}
		}
		cmd := exec.Command("df", "-h")
		if out, err := cmd.Output(); err == nil {
			healthData.WriteString("\nDisk:\n" + string(out))
		}

		context = healthData.String()
		prompt = "Analyze this server health data and provide insights. Suggest optimizations if needed. Be concise."
	} else {
		// Get recent logs
		cmd := exec.Command("journalctl", "-u", "kumomta", "-n", "100", "--no-pager")
		out, err := cmd.Output()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read logs"})
			return
		}
		context = string(out)
		prompt = "Analyze these KumoMTA email server logs. Identify any delivery issues, bounces, or problems. Summarize the email delivery health. Be concise."
	}

	// Call AI API
	analysis, err := callAIAPI(settings.AIProvider, settings.AIAPIKey, prompt, context)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"analysis": analysis,
		"type":     req.Type,
	})
}

func callAIAPI(provider, apiKey, prompt, context string) (string, error) {
	var url string
	var reqBody map[string]interface{}

	fullPrompt := prompt + "\n\nData:\n" + context

	if provider == "deepseek" {
		url = "https://api.deepseek.com/chat/completions"
		reqBody = map[string]interface{}{
			"model": "deepseek-chat",
			"messages": []map[string]string{
				{"role": "user", "content": fullPrompt},
			},
			"max_tokens": 500,
		}
	} else {
		// Default to OpenAI
		url = "https://api.openai.com/v1/chat/completions"
		reqBody = map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]string{
				{"role": "user", "content": fullPrompt},
			},
			"max_tokens": 500,
		}
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response")
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid message format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("no content in response")
	}

	return content, nil
}
