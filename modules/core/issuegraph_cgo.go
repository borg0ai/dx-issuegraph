//go:build cgo

package issuegraph

import (
	"context"
	"fmt"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage/backends"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dolt"
	"github.com/borg0ai/dx-issuegraph/internal/storage/embeddeddolt"
)

// OpenBestAvailable opens a issuegraph database using the best available backend
// for the given .issuegraph directory. It reads metadata.json to determine the
// configured mode:
//
//   - Embedded Dolt (default): Opens via the CGo embedded Dolt engine.
//   - Dolt server: Connects to a dolt sql-server via OpenFromConfig.
//
// The returned Storage must be closed when no longer needed.
//
// issuegraphDir is the path to the .issuegraph directory.
func OpenBestAvailable(ctx context.Context, issuegraphDir string) (Storage, error) {
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		return nil, fmt.Errorf("loading storage metadata: %w", err)
	}
	if cfg == nil {
		cfg = configfile.DefaultConfig()
	}
	if !configfile.IsSupportedBackend(cfg.Backend) {
		return nil, configuredBackendUnavailable(cfg.Backend, issuegraphDir, cfg)
	}

	// Dispatch to a registered extension backend before any Dolt path, mirroring
	// the CLI store factories so SDK callers get the backend they registered
	// instead of a silently-opened embedded Dolt store.
	if backend, ok := backends.Lookup(cfg.GetBackend()); ok {
		return backend.Open(ctx, issuegraphDir)
	}

	if cfg.IsDoltServerMode() {
		store, err := dolt.NewFromConfig(ctx, issuegraphDir)
		if err != nil {
			return nil, err
		}
		return store, nil
	}

	database := configfile.DefaultDoltDatabase
	if cfg != nil {
		database = cfg.GetDoltDatabase()
	}
	store, err := embeddeddolt.Open(ctx, issuegraphDir, database, "main")
	if err != nil {
		return nil, err
	}
	return store, nil
}
