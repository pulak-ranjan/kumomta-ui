package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/api"
	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

func main() {
	dbDir := os.Getenv("DB_DIR")
	if dbDir == "" {
		dbDir = "/var/lib/kumomta-ui"
	}
	dbPath := dbDir + "/panel.db"

	// Ensure DB directory exists
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		log.Printf("Warning: failed to create db directory: %v", err)
	}

	// Initialize Store
	st, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}

	// Initialize Core Services
	ws := core.NewWebhookService(st)
	srv := api.NewServer(st, ws)

	// Start Background Scheduler
	go startScheduler(ws)

	// Start HTTP Server
	addr := "127.0.0.1:9000"
	log.Printf("Kumo UI backend listening on %s\n", addr)
	if err := http.ListenAndServe(addr, srv.Router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func startScheduler(ws *core.WebhookService) {
	log.Println("Starting background scheduler...")

	// Run warmup check immediately on startup to catch up
	go func() {
		log.Println("[Scheduler] Running initial warmup check...")
		if err := core.ProcessDailyWarmup(ws.Store); err != nil {
			log.Printf("Warmup error: %v", err)
		}
	}()

	dailyTicker := time.NewTicker(24 * time.Hour)
	hourlyTicker := time.NewTicker(1 * time.Hour)
	warmupTicker := time.NewTicker(30 * time.Minute) // Check every 30 mins

	for {
		select {
		case <-warmupTicker.C:
			// Run frequent checks for warmup progression
			if err := core.ProcessDailyWarmup(ws.Store); err != nil {
				log.Printf("Warmup error: %v", err)
			}

		case <-dailyTicker.C:
			log.Println("[Scheduler] Running daily tasks...")

			// 1. Daily Summary
			if stats, err := core.GetAllDomainsStats(1); err == nil {
				ws.SendDailySummary(stats)
			}
			
			// 3. Security Audit
			ws.RunSecurityAudit()
			
			// 4. Auto Backup
			if err := core.BackupConfig(); err != nil {
				log.Printf("Backup failed: %v", err)
			} else {
				log.Println("Configuration backed up.")
			}

		case <-hourlyTicker.C:
			log.Println("[Scheduler] Running hourly tasks...")
			ws.CheckBlacklists(false) // Silent check
			ws.CheckBounceRates()
		}
	}
}
