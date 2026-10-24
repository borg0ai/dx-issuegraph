package main

import (
	"fmt"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage/backends"
)

// validateConfiguredBackend fails closed on metadata that selects a removed or
// unrecognized backend. issuegraphDir is the workspace whose metadata cfg came from;
// the rejection inspects it (read-only) so a workspace that already holds a Dolt
// database is told to fix the stale "backend" value rather than to export and
// reinitialize. Pass "" only where there is genuinely no workspace directory.
func validateConfiguredBackend(cfg *configfile.Config, issuegraphDir string) error {
	if cfg == nil {
		return nil
	}
	if backends.Registered(cfg.Backend) {
		return nil
	}
	switch cfg.Backend {
	case configfile.BackendPostgres, configfile.BackendMySQL, configfile.BackendSQLite:
		return configfile.RemovedBackendErrorAt(cfg.Backend, issuegraphDir, cfg)
	case "", configfile.BackendDolt:
		return nil
	default:
		return configfile.UnknownBackendErrorAt(cfg.Backend, issuegraphDir, cfg)
	}
}

// registeredBackendWorkspaceIsIssueGraphDir reports whether metadata selects a
// backend that has no separately discoverable local database.
func registeredBackendWorkspaceIsIssueGraphDir(cfg *configfile.Config) bool {
	if cfg == nil {
		return false
	}
	return backends.WorkspaceIsIssueGraphDir(cfg.GetBackend())
}

func requireDoltBackend(cfg *configfile.Config, issuegraphDir string) error {
	if err := validateConfiguredBackend(cfg, issuegraphDir); err != nil {
		return err
	}
	if cfg != nil && cfg.GetBackend() != configfile.BackendDolt {
		return fmt.Errorf("not using Dolt backend (configured backend %q)", cfg.GetBackend())
	}
	return nil
}

// normalizeLoadedConfig substitutes the default config for an absent
// metadata.json (cfg == nil) so mode inference still runs: a remote host
// supplied via ISSUEGRAPH_DOLT_SERVER_HOST or config.yaml dolt.host (GH#3545)
// must select server mode even when no metadata.json exists — otherwise
// the CLI silently opens the embedded store against a remote-host
// configuration.
func normalizeLoadedConfig(cfg *configfile.Config) *configfile.Config {
	if cfg == nil {
		return configfile.DefaultConfig()
	}
	return cfg
}

func loadDoltBackendConfig(issuegraphDir string) (*configfile.Config, error) {
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg == nil {
		cfg = configfile.DefaultConfig()
	}
	if err := requireDoltBackend(cfg, issuegraphDir); err != nil {
		return nil, err
	}
	return cfg, nil
}
