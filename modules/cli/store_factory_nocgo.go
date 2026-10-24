//go:build !cgo

package main

import (
	"context"
	"fmt"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/storage/backends"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dbproxy/util"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dolt"
)

func usesSQLServer() bool {
	return true
}

// isEmbeddedMode reports whether the command is using embedded Dolt storage.
func isEmbeddedMode() bool {
	return false
}

func usesProxiedServer() bool {
	if shouldUseGlobals() {
		return proxiedServerMode
	}
	return cmdCtx != nil && cmdCtx.ProxiedServerMode
}

// newRegisteredBackendStore is the non-CGO twin of the CGO build's registry
// factory: identical, since the backend registry does not depend on CGO. It
// exists here too so the root pre-run's registry arm has one activating
// construction path in both builds.
func newRegisteredBackendStore(ctx context.Context, name, issuegraphDir string, readOnly bool) (s storage.DoltStorage, err error) {
	defer func() { s, err = activateEventsJournalStore(issuegraphDir, s, err) }()
	backend, ok := backends.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("storage backend %q is not registered", name)
	}
	if readOnly {
		return backend.OpenReadOnly(ctx, issuegraphDir)
	}
	return backend.Open(ctx, issuegraphDir)
}

// newDoltStore applies events-journal activation for the same reason its CGO
// twin does — see the note at the top of events_journal.go.
func newDoltStore(ctx context.Context, cfg *dolt.Config) (s storage.DoltStorage, err error) {
	defer func() { s, err = activateEventsJournalStore(cfg.IssueGraphDir, s, err) }()
	if cfg.ProxiedServer {
		return nil, errProxiedStoreUnrouted()
	}
	if !cfg.ServerMode {
		return nil, fmt.Errorf("%s", nocgoEmbeddedErrMsg)
	}
	return dolt.New(ctx, cfg)
}

// acquireEmbeddedLock returns a no-op lock in non-CGO builds.
func acquireEmbeddedLock(_ string, _ bool) (util.Unlocker, error) {
	return util.NoopLock{}, nil
}

// newDoltStoreFromConfig creates a SQL-server-backed storage backend from config.
func newDoltStoreFromConfig(ctx context.Context, issuegraphDir string) (s storage.DoltStorage, err error) {
	defer func() { s, err = activateEventsJournalStore(issuegraphDir, s, err) }()
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		// Name the real cause: without this, a present-but-unloadable
		// metadata.json surfaces as the misleading "embedded requires CGO"
		// message below.
		return nil, fmt.Errorf("load %s: %w", configfile.ConfigPath(issuegraphDir), err)
	}
	if err := validateConfiguredBackend(cfg, issuegraphDir); err != nil {
		return nil, err
	}
	cfg = normalizeLoadedConfig(cfg)
	if backend, ok := backends.Lookup(cfg.GetBackend()); ok {
		return backend.Open(ctx, issuegraphDir)
	}
	if cfg != nil && cfg.IsDoltProxiedServerMode() {
		return nil, errProxiedStoreUnrouted()
	}
	if cfg != nil && cfg.IsDoltServerMode() {
		return dolt.NewFromConfig(ctx, issuegraphDir)
	}
	return nil, fmt.Errorf("%s", nocgoEmbeddedErrMsg)
}

// newReadOnlyStoreFromConfig creates a read-only SQL-server-backed storage
// backend. It does not activate the events journal: the store refuses writes,
// so there is no mutation for a journal row to accompany (exemption is recorded
// in the construction guard).
func newReadOnlyStoreFromConfig(ctx context.Context, issuegraphDir string) (storage.DoltStorage, error) {
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", configfile.ConfigPath(issuegraphDir), err)
	}
	if err := validateConfiguredBackend(cfg, issuegraphDir); err != nil {
		return nil, err
	}
	cfg = normalizeLoadedConfig(cfg)
	if backend, ok := backends.Lookup(cfg.GetBackend()); ok {
		return backend.OpenReadOnly(ctx, issuegraphDir)
	}
	if cfg != nil && cfg.IsDoltProxiedServerMode() {
		return nil, errProxiedStoreUnrouted()
	}
	if cfg != nil && cfg.IsDoltServerMode() {
		return dolt.NewFromConfigWithOptions(ctx, issuegraphDir, &dolt.Config{ReadOnly: true})
	}
	return nil, fmt.Errorf("%s", nocgoEmbeddedErrMsg)
}

// newPreviewStoreFromConfig is the non-CGO twin of the CGO build's preview
// factory. The two differ only in how they open the EMBEDDED store, and this
// build has no embedded store at all, so preview and read-only coincide here.
func newPreviewStoreFromConfig(ctx context.Context, issuegraphDir string) (storage.DoltStorage, error) {
	return newReadOnlyStoreFromConfig(ctx, issuegraphDir)
}

const nocgoEmbeddedErrMsg = `embedded Dolt requires a CGO build, but this issuegraph binary was built with CGO_ENABLED=0.

Three options:

  1. Use the proxied dolt sql-server (no external server, no reinstall):
       issuegraph init --proxied-server
     issuegraph spawns a per-workspace proxy + child dolt sql-server under
     .issuegraph/dolt/ and manages their lifecycle for you.

  2. Use external server mode (no reinstall needed):
       issuegraph init --server
     Requires a running 'dolt sql-server'. See docs/architecture/dolt.md.

  3. Reinstall with embedded-mode support:
       brew install issuegraph                              # macOS / Linux
       npm install -g @borg0ai/issuegraph                        # any platform with Node
       curl -fsSL https://raw.githubusercontent.com/borg0ai/issuegraph/main/scripts/install.sh | bash

See docs/getting-started/installation.md for the full comparison.`
