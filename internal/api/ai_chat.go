package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages []ChatMessage `json:"messages"`
}

// Allowed "Safe" Tools for the Agent
var allowedTools = map[string]string{
	"status":          "Check KumoMTA Service Status",
	"queue":           "Check Queue Summary",
	"version":         "Check KumoMTA Version",
	"dig":             "Perform DNS Lookup (dig)",
	"logs_kumo":       "Get recent KumoMTA Logs (last 20 lines)",
	"logs_error":      "Get recent Error Logs (last 20 lines)",
	"config_bind_ip":  "Update SMTP listener IP (args: IP address)",
	"dovecot_status":  "Check Dovecot Service Status",
	"fail2ban_status": "Check Fail2Ban Service Status",
}

// POST /api/ai/chat
func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	settings, err := s.Store.GetSettings()
	if err != nil || settings == nil || settings.AIAPIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "AI not configured in settings"})
		return
	}

	// 1. Gather System Context (Live Logs snapshot)
	cmd := exec.Command("journalctl", "-u", "kumomta", "-n", "30", "--no-pager")
	logOut, _ := cmd.CombinedOutput()

	// 2. Gather Knowledge Base (Docs)
	docsContext := loadLocalDocs("docs", 6000) 

	// 3. Construct System Persona & Prompt
	toolsDesc := ""
	for k, v := range allowedTools {
		toolsDesc += fmt.Sprintf("- `%s`: %s\n", k, v)
	}

	systemPrompt := fmt.Sprintf(`You are the KumoMTA Resident Expert Agent.
Your goal is to help the admin manage their MTA, debug errors, and configure the server.

[CAPABILITIES]
You can run safe system tasks by outputting a command tag at the END of your response.
Format: <<EXEC: command_name args>>
Allowed Commands:
%s
- Example: "I will check the queue. <<EXEC: queue>>"
- Example: "I will bind the listener to localhost. <<EXEC: config_bind_ip 127.0.0.1>>"

[CURRENT SYSTEM SNAPSHOT]
Last 30 Log Lines:
%s

[DOCUMENTATION SNIPPET]
%s

INSTRUCTIONS:
- Use Markdown to structure your response (Bold, Tables, Lists, Code Blocks).
- If the user asks about an error, CROSS-REFERENCE the logs with the Documentation.
- If the user asks for a sensitive action (like changing config), ASK FOR CONFIRMATION first.
- NEVER suggest destructive commands (rm, kill, stop).
`, toolsDesc, string(logOut), docsContext)

	// Prepend system prompt
	finalMessages := append([]ChatMessage{{Role: "system", Content: systemPrompt}}, req.Messages...)

	// 4. Call AI Provider
	aiKey, err := core.Decrypt(settings.AIAPIKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to decrypt AI key"})
		return
	}

	rawReply, err := s.sendToAI(settings.AIProvider, aiKey, finalMessages)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// 5. Check for Tool Execution Tags
	reply, toolOutput := s.processToolExecution(rawReply)
	if toolOutput != "" {
		reply += fmt.Sprintf("\n\n**System Output:**\n```\n%s\n```", toolOutput)
	}

	writeJSON(w, http.StatusOK, map[string]string{"reply": reply})
}

// processToolExecution looks for <<EXEC: cmd>> patterns
func (s *Server) processToolExecution(response string) (string, string) {
	re := regexp.MustCompile(`<<EXEC:\s*(\w+)(?:\s+(.+))?>>`)
	matches := re.FindStringSubmatch(response)

	if len(matches) > 0 {
		cmdName := matches[1]
		args := ""
		if len(matches) > 2 {
			args = strings.TrimSpace(matches[2])
		}

		cleanReply := strings.Replace(response, matches[0], "", -1)
		output := s.runSafeTool(cmdName, args)
		return cleanReply, output
	}

	return response, ""
}

// runSafeTool executes only allowlisted logic
func (s *Server) runSafeTool(cmdName, args string) string {
	switch cmdName {
	case "status":
		out, _ := exec.Command("systemctl", "status", "kumomta").CombinedOutput()
		return string(out)

	case "queue":
		stats, err := core.GetQueueStats()
		if err != nil {
			return "Error reading queue stats: " + err.Error()
		}
		return fmt.Sprintf("Total: %d, Queued: %d, Deferred: %d", stats.Total, stats.Queued, stats.Deferred)

	case "version":
		out, _ := exec.Command("rpm", "-q", "kumomta").CombinedOutput()
		return string(out)

	case "dig":
		validDomain := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
		if !validDomain.MatchString(args) {
			return "Invalid domain format."
		}
		out, _ := exec.Command("dig", "+short", "MX", args).CombinedOutput()
		return fmt.Sprintf("MX Records for %s:\n%s", args, string(out))

	case "logs_kumo":
		out, _ := exec.Command("journalctl", "-u", "kumomta", "-n", "20", "--no-pager").CombinedOutput()
		return string(out)
		
	case "logs_error":
		out, _ := exec.Command("journalctl", "-u", "kumomta", "-p", "err", "-n", "20", "--no-pager").CombinedOutput()
		return string(out)

	case "dovecot_status":
		out, _ := exec.Command("systemctl", "status", "dovecot").CombinedOutput()
		return string(out)

	case "fail2ban_status":
		out, _ := exec.Command("fail2ban-client", "status").CombinedOutput()
		return string(out)

	case "config_bind_ip":
		ip := strings.TrimSpace(args)
		if net.ParseIP(ip) == nil {
			return "Error: Invalid IP address provided."
		}
		
		// Update DB
		settings, err := s.Store.GetSettings()
		if err != nil {
			settings = &models.AppSettings{} 
		}
		settings.SMTPListenAddr = ip + ":25"
		if err := s.Store.UpsertSettings(settings); err != nil {
			return "Database Error: " + err.Error()
		}

		// Apply & Restart
		snap, err := core.LoadSnapshot(s.Store)
		if err != nil { return "Snapshot Error: " + err.Error() }
		
		res, err := core.ApplyKumoConfig(snap)
		if err != nil {
			return fmt.Sprintf("Apply Failed: %v\nValidation Output: %s", err, res.ValidationLog)
		}
		return fmt.Sprintf("Success! Listener updated to %s. Service restarted.", settings.SMTPListenAddr)

	default:
		return "Command not allowed or unknown."
	}
}

// Helper to send chat context to OpenAI/DeepSeek
func (s *Server) sendToAI(provider, key string, messages []ChatMessage) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	model := "gpt-3.5-turbo"
	if provider == "deepseek" {
		url = "https://api.deepseek.com/chat/completions"
		model = "deepseek-chat"
	}

	payloadMsgs := make([]map[string]string, len(messages))
	for i, m := range messages {
		payloadMsgs[i] = map[string]string{"role": m.Role, "content": m.Content}
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":    model,
		"messages": payloadMsgs,
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("AI API Error (%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	choices, _ := result["choices"].([]interface{})
	if len(choices) > 0 {
		if c, ok := choices[0].(map[string]interface{}); ok {
			if m, ok := c["message"].(map[string]interface{}); ok {
				return m["content"].(string), nil
			}
		}
	}
	return "No response.", nil
}

// loadLocalDocs reads .md files from ./docs/
func loadLocalDocs(dir string, limit int) string {
	var sb strings.Builder
	totalLen := 0

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, _ := os.ReadFile(path)
			if totalLen+len(content) > limit {
				return filepath.SkipAll
			}
			sb.WriteString(fmt.Sprintf("\n--- DOC: %s ---\n", info.Name()))
			text := string(content)
			text = strings.ReplaceAll(text, "\n\n", "\n") 
			excerptLen := 1500
			if len(text) < excerptLen {
				excerptLen = len(text)
			}
			sb.WriteString(text[:excerptLen])
			totalLen += excerptLen
		}
		return nil
	})
	
	if totalLen == 0 {
		return "No local documentation found."
	}
	return sb.String()
}
