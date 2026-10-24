package dolt

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/storage"
)

func TestResolveLocalActiveDatabaseDir(t *testing.T) {
	clearSizingModeEnv(t)

	t.Run("owned local", func(t *testing.T) {
		clearSizingModeEnv(t)
		issuegraphDir := t.TempDir()
		root := filepath.Join(issuegraphDir, "dolt")
		cfg := &Config{
			Path:          root,
			IssueGraphDir: issuegraphDir,
			Database:      "active",
			ServerHost:    "127.0.0.1",
			AutoStart:     true,
		}
		want := filepath.Join(root, "active")
		if got := resolveLocalActiveDatabaseDir(cfg); got != want {
			t.Fatalf("resolveLocalActiveDatabaseDir = %q, want %q", got, want)
		}
	})

	t.Run("shared local", func(t *testing.T) {
		clearSizingModeEnv(t)
		sharedDir := t.TempDir()
		t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "1")
		t.Setenv("ISSUEGRAPH_SHARED_SERVER_DIR", sharedDir)
		cfg := &Config{
			Path:          filepath.Join(t.TempDir(), "dolt"),
			IssueGraphDir: t.TempDir(),
			Database:      "shared_active",
			ServerHost:    "localhost",
		}
		want := filepath.Join(sharedDir, "dolt", "shared_active")
		if got := resolveLocalActiveDatabaseDir(cfg); got != want {
			t.Fatalf("resolveLocalActiveDatabaseDir = %q, want %q", got, want)
		}
	})

	tests := []struct {
		name  string
		setup func(*testing.T, *Config)
	}{
		{
			name: "owned without auto-start authority",
			setup: func(_ *testing.T, cfg *Config) {
				cfg.AutoStart = false
			},
		},
		{
			name: "external server mode",
			setup: func(t *testing.T, _ *Config) {
				t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "1")
			},
		},
		{
			name: "environment endpoint",
			setup: func(t *testing.T, _ *Config) {
				t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "44001")
			},
		},
		{
			name: "legacy environment endpoint",
			setup: func(t *testing.T, _ *Config) {
				t.Setenv("ISSUEGRAPH_DOLT_PORT", "44001")
			},
		},
		{
			name: "shared mode with explicit endpoint",
			setup: func(t *testing.T, _ *Config) {
				t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "1")
				t.Setenv("ISSUEGRAPH_SHARED_SERVER_DIR", t.TempDir())
				t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "44001")
			},
		},
		{
			name: "remote host",
			setup: func(_ *testing.T, cfg *Config) {
				cfg.ServerHost = "db.example.test"
			},
		},
		{
			name: "socket",
			setup: func(_ *testing.T, cfg *Config) {
				cfg.ServerSocket = "/tmp/dolt.sock"
			},
		},
		{
			name: "TLS",
			setup: func(_ *testing.T, cfg *Config) {
				cfg.ServerTLS = true
			},
		},
		{
			name: "gateway",
			setup: func(_ *testing.T, cfg *Config) {
				cfg.Gateway = true
			},
		},
		{
			name: "proxied",
			setup: func(_ *testing.T, cfg *Config) {
				cfg.ProxiedServer = true
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			clearSizingModeEnv(t)
			issuegraphDir := t.TempDir()
			cfg := &Config{
				Path:          filepath.Join(issuegraphDir, "dolt"),
				IssueGraphDir: issuegraphDir,
				Database:      "active",
				ServerHost:    "127.0.0.1",
				AutoStart:     true,
			}
			tc.setup(t, cfg)
			if got := resolveLocalActiveDatabaseDir(cfg); got != "" {
				t.Fatalf("resolveLocalActiveDatabaseDir = %q, want unsupported", got)
			}
		})
	}
}

func TestDoltStoreActiveDatabaseSizeKeepsPerInstanceAuthorityAcrossModeChurn(t *testing.T) {
	clearSizingModeEnv(t)

	newSizingStore := func(cfg *Config) *DoltStore {
		return &DoltStore{
			dbPath:                 cfg.Path,
			issuegraphDir:          cfg.IssueGraphDir,
			database:               cfg.Database,
			localActiveDatabaseDir: resolveLocalActiveDatabaseDir(cfg),
			serverMode:             true,
		}
	}
	requireSize := func(store *DoltStore, want int64) {
		t.Helper()
		got, err := store.ActiveDatabaseSize(t.Context())
		if err != nil {
			t.Fatalf("ActiveDatabaseSize: %v", err)
		}
		if got != want {
			t.Fatalf("ActiveDatabaseSize = %d, want %d", got, want)
		}
	}
	requireUnsupported := func(store *DoltStore) {
		t.Helper()
		_, err := store.ActiveDatabaseSize(t.Context())
		var unsupported *storage.ErrUnsupported
		if !errors.As(err, &unsupported) {
			t.Fatalf("ActiveDatabaseSize error = %v, want *storage.ErrUnsupported", err)
		}
	}

	ownedIssueGraphDir := t.TempDir()
	ownedConfig := &Config{
		Path:          filepath.Join(ownedIssueGraphDir, "dolt"),
		IssueGraphDir: ownedIssueGraphDir,
		Database:      "owned_active",
		ServerHost:    "127.0.0.1",
		AutoStart:     true,
	}
	ownedDatabaseDir := filepath.Join(ownedConfig.Path, ownedConfig.Database)
	if err := os.MkdirAll(ownedDatabaseDir, 0o700); err != nil {
		t.Fatal(err)
	}
	const ownedPayload = "owned database"
	if err := os.WriteFile(filepath.Join(ownedDatabaseDir, "data"), []byte(ownedPayload), 0o600); err != nil {
		t.Fatal(err)
	}
	ownedStore := newSizingStore(ownedConfig)
	if ownedStore.localActiveDatabaseDir != ownedDatabaseDir {
		t.Fatalf("owned localActiveDatabaseDir = %q, want %q", ownedStore.localActiveDatabaseDir, ownedDatabaseDir)
	}

	t.Setenv("ISSUEGRAPH_DOLT_PORT", "44001")
	externalIssueGraphDir := t.TempDir()
	externalConfig := &Config{
		Path:          filepath.Join(externalIssueGraphDir, "dolt"),
		IssueGraphDir: externalIssueGraphDir,
		Database:      "external_active",
		ServerHost:    "127.0.0.1",
		AutoStart:     true,
	}
	externalStore := newSizingStore(externalConfig)
	if externalStore.localActiveDatabaseDir != "" {
		t.Fatalf("external localActiveDatabaseDir = %q, want unsupported", externalStore.localActiveDatabaseDir)
	}
	requireSize(ownedStore, int64(len(ownedPayload)))
	requireUnsupported(externalStore)

	t.Setenv("ISSUEGRAPH_DOLT_PORT", "")
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "1")
	t.Setenv("ISSUEGRAPH_SHARED_SERVER_DIR", t.TempDir())
	requireSize(ownedStore, int64(len(ownedPayload)))
	requireUnsupported(externalStore)

	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "44002")
	sharedConfig := &Config{
		Path:          filepath.Join(t.TempDir(), "dolt"),
		IssueGraphDir: t.TempDir(),
		Database:      "shared_explicit",
		ServerHost:    "localhost",
	}
	if got := resolveLocalActiveDatabaseDir(sharedConfig); got != "" {
		t.Fatalf("shared explicit-endpoint path = %q, want unsupported", got)
	}
	requireSize(ownedStore, int64(len(ownedPayload)))
	requireUnsupported(externalStore)
}

func TestDoltStoreActiveDatabaseSizeScopesToActiveDatabase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	active := filepath.Join(root, "active")
	sibling := filepath.Join(root, "sibling")
	if err := os.Mkdir(active, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(sibling, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(active, "data"), []byte("active"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sibling, "data"), []byte("much larger sibling data"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := &DoltStore{localActiveDatabaseDir: active}
	got, err := store.ActiveDatabaseSize(t.Context())
	if err != nil {
		t.Fatalf("ActiveDatabaseSize: %v", err)
	}
	if got != int64(len("active")) {
		t.Fatalf("ActiveDatabaseSize = %d, want %d", got, len("active"))
	}
}

func TestDoltStoreActiveDatabaseSizeUnsupportedIgnoresStaleLocalPath(t *testing.T) {
	t.Parallel()

	staleRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(staleRoot, "unrelated"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &DoltStore{dbPath: staleRoot}

	_, err := store.ActiveDatabaseSize(t.Context())
	var unsupported *storage.ErrUnsupported
	if !errors.As(err, &unsupported) {
		t.Fatalf("ActiveDatabaseSize error = %v, want *storage.ErrUnsupported", err)
	}
}

func TestDoltStoreActiveDatabaseSizeFailsForDeclaredLocalMissingPath(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing")
	store := &DoltStore{localActiveDatabaseDir: missing}
	_, err := store.ActiveDatabaseSize(t.Context())
	if err == nil {
		t.Fatal("ActiveDatabaseSize succeeded, want missing-path error")
	}
	var unsupported *storage.ErrUnsupported
	if errors.As(err, &unsupported) {
		t.Fatalf("ActiveDatabaseSize error = %v, want declared-local measurement failure", err)
	}
}

func clearSizingModeEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"ISSUEGRAPH_DOLT_SERVER_MODE",
		"ISSUEGRAPH_DOLT_SHARED_SERVER",
		"ISSUEGRAPH_SHARED_SERVER_DIR",
		"ISSUEGRAPH_DOLT_SERVER_PORT",
		"ISSUEGRAPH_DOLT_PORT",
		"ISSUEGRAPH_DOLT_SERVER_SOCKET",
		"ISSUEGRAPH_DOLT_SERVER_HOST",
	} {
		t.Setenv(name, "")
	}
}
