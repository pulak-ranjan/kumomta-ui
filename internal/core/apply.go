package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Paths for Kumo policy files.
// Later we can move these into settings or env if needed.
const (
	KumoPolicyDir          = "/opt/kumomta/etc/policy"
	KumoSourcesPath        = "/opt/kumomta/etc/policy/sources.toml"
	KumoQueuesPath         = "/opt/kumomta/etc/policy/queues.toml"
	KumoListenerDomainsPath = "/opt/kumomta/etc/policy/listener_domains.toml"
	KumoDKIMDataPath       = "/opt/kumomta/etc/policy/dkim_data.toml"
	KumoInitLuaPath        = "/opt/kumomta/etc/policy/init.lua"

	KumoBinary = "/opt/kumomta/sbin/kumod"
)

// ApplyResult captures what happened during apply.
type ApplyResult struct {
	SourcesPath         string `json:"sources_path"`
	QueuesPath          string `json:"queues_path"`
	ListenerDomainsPath string `json:"listener_domains_path"`
	DKIMDataPath        string `json:"dkim_data_path"`
	InitLuaPath         string `json:"init_lua_path"`

	ValidationOK bool   `json:"validation_ok"`
	ValidationLog string `json:"validation_log"`

	RestartOK bool   `json:"restart_ok"`
	RestartLog string `json:"restart_log"`
}

// ApplyKumoConfig generates and writes Kumo configs,
// validates them, and restarts the Kumo service if validation passes.
func ApplyKumoConfig(snap *Snapshot) (*ApplyResult, error) {
	if err := os.MkdirAll(KumoPolicyDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create policy dir: %w", err)
	}

	// Generate content
	sources := GenerateSourcesTOML(snap)
	queues := GenerateQueuesTOML(snap)
	listenerDomains := GenerateListenerDomainsTOML(snap)
	dkimData := GenerateDKIMDataTOML(snap, "/opt/kumomta/etc/dkim")
	initLua := GenerateInitLua(snap)

	// Write files (0644 so kumod user can read)
	if err := writeFileAtomic(KumoSourcesPath, []byte(sources), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write sources.toml: %w", err)
	}
	if err := writeFileAtomic(KumoQueuesPath, []byte(queues), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write queues.toml: %w", err)
	}
	if err := writeFileAtomic(KumoListenerDomainsPath, []byte(listenerDomains), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write listener_domains.toml: %w", err)
	}
	if err := writeFileAtomic(KumoDKIMDataPath, []byte(dkimData), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write dkim_data.toml: %w", err)
	}
	if err := writeFileAtomic(KumoInitLuaPath, []byte(initLua), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write init.lua: %w", err)
	}

	res := &ApplyResult{
		SourcesPath:         KumoSourcesPath,
		QueuesPath:          KumoQueuesPath,
		ListenerDomainsPath: KumoListenerDomainsPath,
		DKIMDataPath:        KumoDKIMDataPath,
		InitLuaPath:         KumoInitLuaPath,
	}

	// Validate configuration
	validateCmd := exec.Command(
		KumoBinary,
		"--policy", KumoInitLuaPath,
		"--validate",
		"--user", "kumod",
	)
	out, err := validateCmd.CombinedOutput()
	res.ValidationLog = string(out)
	if err != nil {
		res.ValidationOK = false
		// If validation fails, do NOT restart service.
		return res, fmt.Errorf("kumod validation failed: %w", err)
	}
	res.ValidationOK = true

	// Restart Kumo service
	restartCmd := exec.Command("systemctl", "restart", "kumomta")
	restartOut, restartErr := restartCmd.CombinedOutput()
	res.RestartLog = string(restartOut)
	if restartErr != nil {
		res.RestartOK = false
		return res, fmt.Errorf("failed to restart kumomta: %w", restartErr)
	}
	res.RestartOK = true

	return res, nil
}

// writeFileAtomic writes data to a temp file and then renames it,
// to avoid partially written files.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, path)
}
