package main

import (
	"log"
	"net/http"
	"os"

	"github.com/pulak-ranjan/kumomta-ui/internal/api"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

func main() {
	// Ensure DB directory exists
	dbDir := "/var/lib/kumomta-ui"
	dbPath := dbDir + "/panel.db"

	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		log.Fatalf("failed to create db directory: %v", err)
	}

	st, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}

	srv := api.NewServer(st)

	addr := ":9000"
	log.Printf("Kumo UI backend listening on %s\n", addr)

	if err := http.ListenAndServe(addr, srv.Router()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
