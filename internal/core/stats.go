package core

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// --- CACHING STRUCTURES ---
var (
	cacheLock       sync.RWMutex
	cachedStats     map[string][]DayStats
	cacheExpiry     time.Time
	CACHE_DURATION  = 60 * time.Second // Keep data in RAM for 60s
)

// KumoMTA log entry structure
type KumoLogEntry struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"event_time"`
	Sender    string    `json:"sender"`
}

const KumoLogDir = "/var/log/kumomta"

// --- FILE HELPERS ---

// openLogFile intelligently handles both Zstd compressed logs and plain JSON logs
func openLogFile(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// Peek at the first 4 bytes to check for Zstd magic number (0x28 0xB5 0x2F 0xFD)
	br := bufio.NewReader(f)
	magic, _ := br.Peek(4)

	if len(magic) >= 4 && magic[0] == 0x28 && magic[1] == 0xB5 && magic[2] == 0x2F && magic[3] == 0xFD {
		// It is compressed (Zstd)
		decoder, err := zstd.NewReader(br)
		if err != nil {
			f.Close()
			return nil, err
		}
		return &compressedReader{f: f, d: decoder}, nil
	}

	// It is plain text (JSON)
	return &plainReader{f: f, r: br}, nil
}

// Wrapper to close both the decoder and the file
type compressedReader struct {
	f *os.File
	d *zstd.Decoder
}
func (c *compressedReader) Read(p []byte) (int, error) { return c.d.Read(p) }
func (c *compressedReader) Close() error { 
	c.d.Close()
	return c.f.Close() 
}

// Wrapper for plain text buffered reader
type plainReader struct {
	f *os.File
	r *bufio.Reader
}
func (p *plainReader) Read(b []byte) (int, error) { return p.r.Read(b) }
func (p *plainReader) Close() error { return p.f.Close() }

func extractDomain(email string) string {
	if email == "" { return "" }
	parts := strings.Split(email, "@")
	if len(parts) != 2 { return "" }
	return strings.ToLower(parts[1])
}

// --- PARALLEL PROCESSING ---

// ParseKumoLogs (Database Sync)
func ParseKumoLogs(st *store.Store, hoursBack int) error {
	// Calculate days to fetch (ensure at least 1 day)
	days := hoursBack / 24
	if days < 1 {
		days = 1
	}

	// Re-use the aggregation logic
	stats, err := GetAllDomainsStats(days)
	if err != nil {
		return err
	}

	for domain, daysData := range stats {
		for _, day := range daysData {
			parsedDate, _ := time.Parse("2006-01-02", day.Date)
			dbStat := &models.EmailStats{
				Domain:    domain,
				Date:      parsedDate,
				Sent:      day.Sent,
				Delivered: day.Delivered,
				Bounced:   day.Bounced,
				Deferred:  day.Deferred,
			}
			st.SetEmailStats(dbStat)
		}
	}
	return nil
}

// GetDomainStatsFromLogs (Single Domain)
func GetDomainStatsFromLogs(domain string, days int) ([]DayStats, error) {
	all, err := GetAllDomainsStats(days)
	if err != nil {
		return nil, err
	}
	if d, ok := all[domain]; ok {
		return d, nil
	}
	// Return empty days if not found
	empty := make([]DayStats, days)
	now := time.Now()
	for i := 0; i < days; i++ {
		empty[i] = DayStats{Date: now.AddDate(0, 0, -(days - 1 - i)).Format("2006-01-02")}
	}
	return empty, nil
}

type DayStats struct {
	Date      string `json:"date"`
	Sent      int64  `json:"sent"`
	Delivered int64  `json:"delivered"`
	Bounced   int64  `json:"bounced"`
	Deferred  int64  `json:"deferred"`
}

// GetAllDomainsStats (Aggregated & Parallel)
func GetAllDomainsStats(days int) (map[string][]DayStats, error) {
	// 1. Check Cache
	cacheLock.RLock()
	if time.Now().Before(cacheExpiry) && cachedStats != nil {
		res := cachedStats // copy pointer
		cacheLock.RUnlock()
		return res, nil
	}
	cacheLock.RUnlock()

	// 2. Setup Scanning
	files, _ := filepath.Glob(filepath.Join(KumoLogDir, "*"))
	now := time.Now()
	cutoff := now.AddDate(0, 0, -days)

	// Thread-safe storage for results
	// map[domain]map[date]*DayStats
	var mu sync.Mutex
	tempStats := make(map[string]map[string]*DayStats)

	// Worker Pool Settings
	numWorkers := 50  // Process 50 files simultaneously
	fileChan := make(chan string, len(files))
	var wg sync.WaitGroup

	// 3. Start Workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				processFile(file, cutoff, &mu, tempStats)
			}
		}()
	}

	// 4. Feed Workers
	for _, file := range files {
		// Quick optimization: skip old files without opening
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			if info.ModTime().After(cutoff) {
				fileChan <- file
			}
		}
	}
	close(fileChan)

	// 5. Wait for finish
	wg.Wait()

	// 6. Format Results
	result := make(map[string][]DayStats)
	for domain, dateMap := range tempStats {
		daysList := make([]DayStats, 0, len(dateMap))
		for _, stat := range dateMap {
			daysList = append(daysList, *stat)
		}
		// Sort by date
		for i := 0; i < len(daysList)-1; i++ {
			for j := i + 1; j < len(daysList); j++ {
				if daysList[i].Date > daysList[j].Date {
					daysList[i], daysList[j] = daysList[j], daysList[i]
				}
			}
		}
		result[domain] = daysList
	}

	// 7. Save to Cache
	cacheLock.Lock()
	cachedStats = result
	cacheExpiry = time.Now().Add(CACHE_DURATION)
	cacheLock.Unlock()

	return result, nil
}

// processFile is run by workers
func processFile(file string, cutoff time.Time, mu *sync.Mutex, tempStats map[string]map[string]*DayStats) {
	rc, err := openLogFile(file)
	if err != nil {
		return
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	// Large buffer for potentially long log lines
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 5*1024*1024)

	// Local aggregation to reduce mutex locking
	localStats := make(map[string]map[string]*DayStats)

	for scanner.Scan() {
		line := scanner.Text()
		
		// Find start of JSON object
		if idx := strings.Index(line, "{"); idx >= 0 {
			line = line[idx:]
		} else {
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
		if domain == "" { continue }

		dateKey := entry.Timestamp.Format("2006-01-02")

		if localStats[domain] == nil {
			localStats[domain] = make(map[string]*DayStats)
		}
		if localStats[domain][dateKey] == nil {
			localStats[domain][dateKey] = &DayStats{Date: dateKey}
		}

		s := localStats[domain][dateKey]
		switch entry.Type {
		case "Reception": s.Sent++
		case "Delivery": s.Delivered++
		case "Bounce": s.Bounced++
		case "TransientFailure": s.Deferred++
		}
	}

	// Merge into main stats (Locking only for the merge, not during parsing)
	mu.Lock()
	for dom, days := range localStats {
		if tempStats[dom] == nil {
			tempStats[dom] = make(map[string]*DayStats)
		}
		for date, stat := range days {
			if tempStats[dom][date] == nil {
				tempStats[dom][date] = &DayStats{Date: date}
			}
			target := tempStats[dom][date]
			target.Sent += stat.Sent
			target.Delivered += stat.Delivered
			target.Bounced += stat.Bounced
			target.Deferred += stat.Deferred
		}
	}
	mu.Unlock()
}
