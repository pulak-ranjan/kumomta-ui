package core

import (
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// Snapshot represents a consistent view of configuration
// used to generate KumoMTA/Dovecot/firewall configs.
type Snapshot struct {
	Settings *models.AppSettings
	Domains  []models.Domain
}

// LoadSnapshot collects app settings + all domains (+ senders).
func LoadSnapshot(st *store.Store) (*Snapshot, error) {
	settings, err := st.GetSettings()
	if err != nil && err != store.ErrNotFound {
		return nil, err
	}
	// settings can be nil if not configured yet

	domains, err := st.ListDomains()
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		Settings: settings,
		Domains:  domains,
	}, nil
}
