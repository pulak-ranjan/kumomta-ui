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
	"strconv"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

// ----------------------
// IP Management
// ----------------------

type addIPRequest struct {
	CIDR string `json:"cidr"`
	List string `json:"list"`
}

func (s *Server) handleGetSystemIPs(w http.ResponseWriter, r *http.Request) {
	dbIPs, _ := s.Store.ListSystemIPs()
	ifaceIPs := detectInterfaceIPs()
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

func (s *Server) handleAddIPs(w http.ResponseWriter, r *http.Request) {
	var req addIPRequest
	json.NewDecoder(r.Body).Decode(&req)
	var newIPs []models.SystemIP

	if req.CIDR != "" {
		ip, ipnet, err := net.ParseCIDR(strings.TrimSpace(req.CIDR))
		if err == nil {
			for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
				newIPs = append(newIPs, models.SystemIP{Value: ip.String(), Netmask: req.CIDR})
			}
		}
	}
	if req.List != "" {
		for _, line := range strings.Split(req.List, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && net.ParseIP(line) != nil {
				newIPs = append(newIPs, models.SystemIP{Value: line})
			}
		}
	}
	if len(newIPs) > 0 {
		s.Store.CreateSystemIPs(newIPs)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "added": strconv.Itoa(len(newIPs))})
}

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
	Domains       int64  `json:"domains"`
	Senders       int64  `json:"senders"`
	CPULoad       string `json:"cpu_load"`
	RAMUsage      string `json:"ram_usage"`
	KumoStatus    string `json:"kumo_status"`
	DovecotStatus string `json:"dovecot_status"`
	F2BStatus     string `json:"f2b_status"`
	OpenPorts     string `json:"open_ports"`
}

func (s *Server) handleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	dCount, _ := s.Store.CountDomains()
	sCount, _ := s.Store.CountSenders()

	stats := dashboardStatsDTO{
		Domains: dCount,
		Senders: sCount,
	}

	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) > 0 {
			stats.CPULoad = parts[0]
		}
	} else {
		stats.CPULoad = "N/A"
	}

	stats.RAMUsage = getSystemRAM()

	// Port Check
	ports := []string{}
	for _, p := range []string{"25", "587", "9000", "80", "443"} {
		if conn, err := net.DialTimeout("tcp", "127.0.0.1:"+p, 200*time.Millisecond); err == nil {
			conn.Close()
			ports = append(ports, p)
		}
	}
	stats.OpenPorts = strings.Join(ports, ", ")

	// Use the serviceStatus helper from server.go (same package)
	stats.KumoStatus = serviceStatus("kumomta")
	stats.DovecotStatus = serviceStatus("dovecot")
	stats.F2BStatus = serviceStatus("fail2ban")

	writeJSON(w, http.StatusOK, stats)
}

func getSystemRAM() string {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "N/A"
	}
	var total, free, avail int
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		v, _ := strconv.Atoi(f[1])
		switch f[0] {
		case "MemTotal:":
			total = v
		case "MemFree:":
			free = v
		case "MemAvailable:":
			avail = v
		}
	}
	if avail == 0 {
		avail = free
	}
	used := total - avail
	return fmt.Sprintf("%d / %d MB", used/1024, total/1024)
}

// ----------------------
// AI Insights
// ----------------------

func (s *Server) handleAIInsights(w http.ResponseWriter, r *http.Request) {
	st, err := s.Store.GetSettings()
	if err != nil || st == nil || st.AIAPIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "AI not configured"})
		return
	}

	cmd := exec.Command("journalctl", "-u", "kumomta", "-n", "50", "--no-pager")
	out, _ := cmd.CombinedOutput()
	logs := string(out)

	sysContext := fmt.Sprintf("Analyze these KumoMTA logs and summarize delivery health:\n%s", logs)

	apiUrl := "https://api.openai.com/v1/chat/completions"
	model := "gpt-3.5-turbo"
	if st.AIProvider == "deepseek" {
		apiUrl = "https://api.deepseek.com/v1/chat/completions"
		model = "deepseek-chat"
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a mail server analyst."},
			{"role": "user", "content": sysContext},
		},
	})

	aiReq, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(reqBody))
	aiReq.Header.Set("Authorization", "Bearer "+st.AIAPIKey)
	aiReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(aiReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "AI Error: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "AI Provider Error: " + string(body)})
		return
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	content := "No insight."
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if c, ok := msg["content"].(string); ok {
					content = c
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"insight": content})
}
