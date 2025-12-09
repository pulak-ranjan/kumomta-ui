package main

import (
	"log"
	"net/http"
	"os"

	"github.com/pulak-ranjan/kumomta-ui/internal/api"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

func main() {
	// 1. Allow configuration via Environment Variable (Good for Rocky 9 + Dev)
	dbDir := os.Getenv("DB_DIR")
	if dbDir == "" {
		// Default standard path for Rocky Linux / RHEL
		dbDir = "/var/lib/kumomta-ui"
	}

	dbPath := dbDir + "/panel.db"

	// 2. Create the directory if it doesn't exist
	// Note: If running as a non-root user, ensure this user has permissions for dbDir
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		log.Printf("Warning: failed to create db directory %s: %v", dbDir, err)
	}

	log.Printf("Opening database at: %s", dbPath)
	st, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}

	srv := api.NewServer(st)

	addr := ":9000"
	log.Printf("Kumo UI backend listening on %s\n", addr)

	// FIX: srv.Router is a field, not a method. Remove "()"
	if err := http.ListenAndServe(addr, srv.Router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
