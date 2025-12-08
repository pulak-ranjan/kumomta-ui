package api

import (
	"net/http"

	"github.com/pulak-ranjan/kumomta-ui/internal/core"
)

// configPreviewDTO is what we return for previewing generated configs.
type configPreviewDTO struct {
	SourcesTOML         string `json:"sources_toml"`
	QueuesTOML          string `json:"queues_toml"`
	ListenerDomainsTOML string `json:"listener_domains_toml"`
	DKIMDataTOML        string `json:"dkim_data_toml"`
	InitLua             string `json:"init_lua"`
}

// GET /api/config/preview
func (s *Server) handlePreviewConfig(w http.ResponseWriter, r *http.Request) {
	snap, err := core.LoadSnapshot(s.Store)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load snapshot"})
		return
	}

	// Even if settings or domains are empty, we still generate something.
	// Later the UI can show "no domains configured yet" on top.
	const dkimBasePath = "/opt/kumomta/etc/dkim"

	out := configPreviewDTO{
		SourcesTOML:         core.GenerateSourcesTOML(snap),
		QueuesTOML:          core.GenerateQueuesTOML(snap),
		ListenerDomainsTOML: core.GenerateListenerDomainsTOML(snap),
		DKIMDataTOML:        core.GenerateDKIMDataTOML(snap, dkimBasePath),
		InitLua:             core.GenerateInitLua(snap),
	}

	writeJSON(w, http.StatusOK, out)
}
