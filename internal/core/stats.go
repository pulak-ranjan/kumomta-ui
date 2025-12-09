package core

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// KumoMTA log entry structure (simplified)
type KumoLogEntry struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"ts"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Queue     string    `json:"queue"`
	Status    string    `json:"status"`
	Bounce    struct {
		Category string `json:"category"`
	} `json:"bounce"`
}

const KumoLogDir = "/var/log/kumomta"

// ParseKumoLogs reads KumoMTA JSON logs and aggregates stats by domain
func ParseKumoLogs(st *store.Store, hoursBack int) error {
	files, err := filepath.Glob(filepath.Join(KumoLogDir, "*.log"))
	if err != nil {
		return err
	}

	// Also check for .json files
	jsonFiles, _ := filepath.Glob(filepath.Join(KumoLogDir, "*.json"))
	files = append(files, jsonFiles...)

	cutoff := time.Now().Add(-time.Duration(hoursBack) * time.Hour)
	
	// Aggregate stats per domain per day
	stats := make(map[string]map[string]*models.EmailStats) // domain -> date -> stats

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		// Increase buffer size for large log lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var entry KumoLogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}

			// Skip old entries
			if entry.Timestamp.Before(cutoff) {
				continue
			}

			// Extract domain from sender
			domain := extractDomain(entry.Sender)
			if domain == "" {
				continue
			}

			dateKey := entry.Timestamp.Format("2006-01-02")

			if stats[domain] == nil {
				stats[domain] = make(map[string]*models.EmailStats)
			}
			if stats[domain][dateKey] == nil {
				date, _ := time.Parse("2006-01-02", dateKey)
				stats[domain][dateKey] = &models.EmailStats{
					Domain: domain,
					Date:   date,
				}
			}

			s := stats[domain][dateKey]

			switch entry.Type {
			case "Reception":
				s.Sent++
			case "Delivery":
				s.Delivered++
			case "Bounce":
				s.Bounced++
			case "TransientFailure":
				s.Deferred++
			}
		}
		f.Close()
	}

	// Save all stats to DB
	for _, domainStats := range stats {
		for _, stat := range domainStats {
			if err := st.SetEmailStats(stat); err != nil {
				st.LogError(err)
			}
		}
	}

	return nil
}

func extractDomain(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// GetDomainStatsFromLogs reads logs directly for real-time stats
func GetDomainStatsFromLogs(domain string, days int) ([]DayStats, error) {
	result := make([]DayStats, days)
	now := time.Now()

	// Initialize dates
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -(days-1-i))
		result[i] = DayStats{
			Date: date.Format("2006-01-02"),
		}
	}

	files, _ := filepath.Glob(filepath.Join(KumoLogDir, "*.log"))
	jsonFiles, _ := filepath.Glob(filepath.Join(KumoLogDir, "*.json"))
	files = append(files, jsonFiles...)

	cutoff := now.AddDate(0, 0, -days)

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var entry KumoLogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}

			if entry.Timestamp.Before(cutoff) {
				continue
			}

			entryDomain := extractDomain(entry.Sender)
			if domain != "" && entryDomain != domain {
				continue
			}

			dateKey := entry.Timestamp.Format("2006-01-02")

			// Find matching day in result
			for i := range result {
				if result[i].Date == dateKey {
					switch entry.Type {
					case "Reception":
						result[i].Sent++
					case "Delivery":
						result[i].Delivered++
					case "Bounce":
						result[i].Bounced++
					case "TransientFailure":
						result[i].Deferred++
					}
					break
				}
			}
		}
		f.Close()
	}

	return result, nil
}

type DayStats struct {
	Date      string `json:"date"`
	Sent      int64  `json:"sent"`
	Delivered int64  `json:"delivered"`
	Bounced   int64  `json:"bounced"`
	Deferred  int64  `json:"deferred"`
}

// GetAllDomainsStats aggregates stats across all domains
func GetAllDomainsStats(days int) (map[string][]DayStats, error) {
	result := make(map[string][]DayStats)
	now := time.Now()

	files, _ := filepath.Glob(filepath.Join(KumoLogDir, "*.log"))
	jsonFiles, _ := filepath.Glob(filepath.Join(KumoLogDir, "*.json"))
	files = append(files, jsonFiles...)

	cutoff := now.AddDate(0, 0, -days)

	// Temp storage: domain -> date -> stats
	temp := make(map[string]map[string]*DayStats)

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var entry KumoLogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}

			if entry.Timestamp.Before(cutoff) {
				continue
			}

			domain := extractDomain(entry.Sender)
			if domain == "" {
				continue
			}

			dateKey := entry.Timestamp.Format("2006-01-02")

			if temp[domain] == nil {
				temp[domain] = make(map[string]*DayStats)
			}
			if temp[domain][dateKey] == nil {
				temp[domain][dateKey] = &DayStats{Date: dateKey}
			}

			s := temp[domain][dateKey]
			switch entry.Type {
			case "Reception":
				s.Sent++
			case "Delivery":
				s.Delivered++
			case "Bounce":
				s.Bounced++
			case "TransientFailure":
				s.Deferred++
			}
		}
		f.Close()
	}

	// Convert to sorted arrays
	for domain, dateMap := range temp {
		days := make([]DayStats, 0, len(dateMap))
		for _, stat := range dateMap {
			days = append(days, *stat)
		}
		// Sort by date
		for i := 0; i < len(days)-1; i++ {
			for j := i + 1; j < len(days); j++ {
				if days[i].Date > days[j].Date {
					days[i], days[j] = days[j], days[i]
				}
			}
		}
		result[domain] = days
	}

	return result, nil
}
