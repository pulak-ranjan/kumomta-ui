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
	// 1. Config via Env
	dbDir := os.Getenv("DB_DIR")
	if dbDir == "" {
		dbDir = "/var/lib/kumomta-ui"
	}
	dbPath := dbDir + "/panel.db"

	// 2. Setup DB
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		log.Printf("Warning: failed to create db directory %s: %v", dbDir, err)
	}

	log.Printf("Opening database at: %s", dbPath)
	st, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}

	// 3. Initialize Services
	ws := core.NewWebhookService(st)
	srv := api.NewServer(st, ws)

	// 4. Start Background Scheduler (The "Worker")
	go startScheduler(ws)

	// 5. Start HTTP Server
	// Listen only on localhost to force Nginx SSL usage
	addr := "127.0.0.1:9000"
	log.Printf("Kumo UI backend listening on %s (localhost only)\n", addr)

	if err := http.ListenAndServe(addr, srv.Router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func startScheduler(ws *core.WebhookService) {
	log.Println("Starting background scheduler...")

	// Tickers
	dailyTicker := time.NewTicker(24 * time.Hour)
	hourlyTicker := time.NewTicker(1 * time.Hour)
	
	// Run once immediately on startup (optional, good for testing)
	go ws.CheckBounceRates()

	for {
		select {
		case <-dailyTicker.C:
			log.Println("[Scheduler] Running daily tasks...")
			// 1. Daily Stats Summary
			if stats, err := core.GetAllDomainsStats(1); err == nil {
				ws.SendDailySummary(stats)
			}
			// 2. Security Audit
			ws.RunSecurityAudit()

		case <-hourlyTicker.C:
			log.Println("[Scheduler] Running hourly tasks...")
			// 1. Check Blacklists
			ws.CheckBlacklists()
			// 2. Check Bounce Rates
			ws.CheckBounceRates()
		}
	}
}
