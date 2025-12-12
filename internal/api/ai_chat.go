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
	Messages []ChatMessage `json:"messages"` // Optional now, usually just the last one is needed if using DB
	NewMsg   string        `json:"new_msg"`  // The new message from user
}

// Allowed "Safe" Tools for the Agent
var allowedTools = map[string]string{
	"status":          "Check KumoMTA Service Status",
	"queue":           "Check Queue Summary",
	"logs_kumo":       "Get recent KumoMTA Logs (last 30 lines)",
	"logs_error":      "Get recent Error Logs (last 30 lines)",
	"config_bind_ip":  "Update SMTP listener IP (PORT 25 ONLY). Args: IP address",
	"block_ip":        "Ban an IP address using Firewall (Guardian). Args: IP",
	"backup_config":   "Create a backup of /opt/kumomta/etc.",
	"dig":             "Perform DNS Lookup (dig)",
}

// GET /api/ai/history
func (s *Server) handleGetChatHistory(w http.ResponseWriter, r *http.Request) {
	logs, err := s.Store.GetChatHistory(50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch history"})
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

// POST /api/ai/chat
func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Handle Legacy UI call format (just messages array)
	userContent := req.NewMsg
	if userContent == "" && len(req.Messages) > 0 {
		userContent = req.Messages[len(req.Messages)-1].Content
	}

	if userContent == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "empty message"})
		return
	}

	// Save User Message
	s.Store.SaveChatLog("user", userContent)

	settings, err := s.Store.GetSettings()
	if err != nil || settings == nil || settings.AIAPIKey == "" {
		reply := "AI is not configured. Please add an API Key in Settings."
		s.Store.SaveChatLog("assistant", reply)
		writeJSON(w, http.StatusOK, map[string]string{"reply": reply})
		return
	}

	// 1. Gather Context
	cmd := exec.Command("journalctl", "-u", "kumomta", "-n", "30", "--no-pager")
	logOut, _ := cmd.CombinedOutput()

	// 2. Load History from DB (Memory)
	historyLogs, _ := s.Store.GetChatHistory(10) // Last 10 exchanges for context
	var contextMsgs []ChatMessage
	for _, l := range historyLogs {
		contextMsgs = append(contextMsgs, ChatMessage{Role: l.Role, Content: l.Content})
	}

	// 3. Construct System Prompt
	toolsDesc := ""
	for k, v := range allowedTools {
		toolsDesc += fmt.Sprintf("- `%s`: %s\n", k, v)
	}

	systemPrompt := fmt.Sprintf(`You are the KumoMTA Guardian. 
Your role is to Secure, Configure, and Monitor this MTA server.

[TOOLS]
You can run system tasks by outputting a command tag at the END of your response.
Format: <<EXEC: command_name args>>
Allowed Commands:
%s

[BEHAVIORAL RULES]
1. **Be Structured:** Use Headers (##) and Bullet points. No conversational fluff.
2. **Be Suspicious:** If logs show "auth failed" or "relay denied" > 5 times from an IP, suggest 'block_ip'.
3. **Be Safe:** ALWAYS run 'backup_config' before 'config_bind_ip'.
4. **Binding Info:** 'config_bind_ip' ONLY changes Port 25. Ports 587/465 remain on 0.0.0.0. Tell the user this.
5. **Confirmation:** If the user asks for a sensitive action (change config, block IP), ASK FOR CONFIRMATION first.

[CURRENT SYSTEM LOGS]
%s
`, toolsDesc, string(logOut))

	// Construct final payload
	finalMessages := []ChatMessage{{Role: "system", Content: systemPrompt}}
	finalMessages = append(finalMessages, contextMsgs...)

	// 4. Call AI Provider
	rawReply, err := s.sendToAI(settings.AIProvider, settings.AIAPIKey, finalMessages)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// 5. Process Tools
	reply, toolOutput := s.processToolExecution(rawReply)
	if toolOutput != "" {
		reply += fmt.Sprintf("\n\n**System Output:**\n```\n%s\n```", toolOutput)
	}

	// Save Assistant Reply
	s.Store.SaveChatLog("assistant", reply)

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

	case "logs_kumo":
		out, _ := exec.Command("journalctl", "-u", "kumomta", "-n", "30", "--no-pager").CombinedOutput()
		return string(out)
		
	case "logs_error":
		out, _ := exec.Command("journalctl", "-u", "kumomta", "-p", "err", "-n", "30", "--no-pager").CombinedOutput()
		return string(out)

	case "dig":
		validDomain := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
		if !validDomain.MatchString(args) {
			return "Invalid domain format."
		}
		out, _ := exec.Command("dig", "+short", "MX", args).CombinedOutput()
		return fmt.Sprintf("MX Records for %s:\n%s", args, string(out))

	case "block_ip":
		if err := core.BlockIP(args); err != nil {
			return "Block Failed: " + err.Error()
		}
		return fmt.Sprintf("✅ IP %s has been blocked in firewall.", args)

	case "backup_config":
		if err := core.BackupConfig(); err != nil {
			return "Backup Failed: " + err.Error()
		}
		return "✅ Configuration backed up successfully."

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
		return fmt.Sprintf("Success! Port 25 Listener updated to %s. Service restarted.", settings.SMTPListenAddr)

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
