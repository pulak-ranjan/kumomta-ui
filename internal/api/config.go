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

// configApplyResponse is what we return after applying to the system.
type configApplyResponse struct {
	ApplyResult *core.ApplyResult `json:"apply_result,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// GET /api/config/preview
func (s *Server) handlePreviewConfig(w http.ResponseWriter, r *http.Request) {
	snap, err := core.LoadSnapshot(s.Store)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load snapshot"})
		return
	}

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

// POST /api/config/apply
// This will:
//  - load snapshot
//  - generate configs
//  - write them to real Kumo paths
//  - validate via kumod
//  - restart kumomta if validation passes
func (s *Server) handleApplyConfig(w http.ResponseWriter, r *http.Request) {
	snap, err := core.LoadSnapshot(s.Store)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, configApplyResponse{
			Error: "failed to load snapshot",
		})
		return
	}

	res, applyErr := core.ApplyKumoConfig(snap)
	if applyErr != nil {
		s.Store.LogError(applyErr)
		writeJSON(w, http.StatusInternalServerError, configApplyResponse{
			ApplyResult: res,
			Error:       applyErr.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, configApplyResponse{
		ApplyResult: res,
	})
}
