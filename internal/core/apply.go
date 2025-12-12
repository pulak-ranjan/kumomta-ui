package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Paths for Kumo policy files.
const (
	KumoPolicyDir           = "/opt/kumomta/etc/policy"
	KumoSourcesPath         = "/opt/kumomta/etc/policy/sources.toml"
	KumoQueuesPath          = "/opt/kumomta/etc/policy/queues.toml"
	KumoListenerDomainsPath = "/opt/kumomta/etc/policy/listener_domains.toml"
	KumoDKIMDataPath        = "/opt/kumomta/etc/policy/dkim_data.toml"
	KumoInitLuaPath         = "/opt/kumomta/etc/policy/init.lua"

	KumoBinary = "/opt/kumomta/sbin/kumod"
)

// ApplyResult captures what happened during apply.
type ApplyResult struct {
	SourcesPath         string `json:"sources_path"`
	QueuesPath          string `json:"queues_path"`
	ListenerDomainsPath string `json:"listener_domains_path"`
	DKIMDataPath        string `json:"dkim_data_path"`
	InitLuaPath         string `json:"init_lua_path"`

	ValidationOK  bool   `json:"validation_ok"`
	ValidationLog string `json:"validation_log"`

	RestartOK  bool   `json:"restart_ok"`
	RestartLog string `json:"restart_log"`
}

// ApplyKumoConfig generates comprehensive configs from the DB.
// It effectively "searches and verifies" that all records in the DB
// are present in the config files.
func ApplyKumoConfig(snap *Snapshot) (*ApplyResult, error) {
	// 1. Ensure Directory Exists
	if err := os.MkdirAll(KumoPolicyDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create policy dir: %w", err)
	}

	// 2. Generate "Perfect" Content from Database
	// These functions (in configgen.go) ensure all DB records are included.
	sources := GenerateSourcesTOML(snap)
	queues := GenerateQueuesTOML(snap)
	listenerDomains := GenerateListenerDomainsTOML(snap)
	dkimData := GenerateDKIMDataTOML(snap, "/opt/kumomta/etc/dkim")
	initLua := GenerateInitLua(snap)

	// Helper: Smart Update
	// Writes the file only if it's missing or different.
	// If different, it creates a .bak backup first.
	applyFile := func(path string, content string) error {
		return smartUpdateFile(path, []byte(content), 0o644)
	}

	// 3. Apply Files (Self-Healing)
	if err := applyFile(KumoSourcesPath, sources); err != nil {
		return nil, fmt.Errorf("failed to apply sources.toml: %w", err)
	}
	if err := applyFile(KumoQueuesPath, queues); err != nil {
		return nil, fmt.Errorf("failed to apply queues.toml: %w", err)
	}
	if err := applyFile(KumoListenerDomainsPath, listenerDomains); err != nil {
		return nil, fmt.Errorf("failed to apply listener_domains.toml: %w", err)
	}
	if err := applyFile(KumoDKIMDataPath, dkimData); err != nil {
		return nil, fmt.Errorf("failed to apply dkim_data.toml: %w", err)
	}
	// This heals init.lua if it's missing the new logic
	if err := applyFile(KumoInitLuaPath, initLua); err != nil {
		return nil, fmt.Errorf("failed to apply init.lua: %w", err)
	}

	res := &ApplyResult{
		SourcesPath:         KumoSourcesPath,
		QueuesPath:          KumoQueuesPath,
		ListenerDomainsPath: KumoListenerDomainsPath,
		DKIMDataPath:        KumoDKIMDataPath,
		InitLuaPath:         KumoInitLuaPath,
	}

	// 4. Validate Configuration
	// We use the real Kumo binary to verify the generated config is valid.
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
		// CRITICAL: Validation failed. We do NOT restart.
		// The .bak files are available if the user needs to revert manually.
		return res, fmt.Errorf("kumod validation failed (check logs): %w", err)
	}
	res.ValidationOK = true

	// 5. Restart Service
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

// smartUpdateFile implements the "Check, Backup, Write" logic
func smartUpdateFile(path string, data []byte, perm os.FileMode) error {
	// 1. Read existing file
	existing, err := os.ReadFile(path)
	if err == nil {
		// 2. Compare content
		if string(existing) == string(data) {
			// IDENTICAL: No action needed. This is safe and efficient.
			return nil
		}

		// 3. DIFFERENT: Create Backup
		backupPath := path + ".bak"
		// We ignore backup errors (e.g. permission) to prioritize system recovery,
		// but typically this succeeds.
		_ = os.WriteFile(backupPath, existing, perm)
	}

	// 4. Write New Content (Atomic: Temp -> Rename)
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, path)
}
