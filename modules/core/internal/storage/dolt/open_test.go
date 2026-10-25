package dolt

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/doltserver"
)

// TestResolveAutoStart verifies all conditions that govern the AutoStart decision.
//
// Each subtest uses t.Setenv for env-var isolation: t.Setenv records the
// original value (including the unset state) and restores it after the test,
// correctly handling cases where a variable was previously unset vs. set to "".
func TestResolveAutoStart(t *testing.T) {
	tests := []struct {
		name             string
		testMode         string // ISSUEGRAPH_TEST_MODE to set; "" leaves it unset/empty
		autoStartEnv     string // ISSUEGRAPH_DOLT_AUTO_START to set; "" leaves it unset/empty
		doltAutoStartCfg string // raw value of "dolt.auto-start" from config.yaml
		currentValue     bool   // AutoStart value supplied by caller
		wantAutoStart    bool
	}{
		{
			name:          "defaults to true for standalone user",
			wantAutoStart: true,
		},
		{
			name:          "disabled when ISSUEGRAPH_TEST_MODE=1",
			testMode:      "1",
			wantAutoStart: false,
		},
		{
			name:          "disabled when ISSUEGRAPH_DOLT_AUTO_START=0",
			autoStartEnv:  "0",
			wantAutoStart: false,
		},
		{
			name:          "enabled when ISSUEGRAPH_DOLT_AUTO_START=1",
			autoStartEnv:  "1",
			wantAutoStart: true,
		},
		{
			name:             "disabled when dolt.auto-start=false in config",
			doltAutoStartCfg: "false",
			wantAutoStart:    false,
		},
		{
			name:             "disabled when dolt.auto-start=0 in config",
			doltAutoStartCfg: "0",
			wantAutoStart:    false,
		},
		{
			name:             "disabled when dolt.auto-start=off in config",
			doltAutoStartCfg: "off",
			wantAutoStart:    false,
		},
		{
			name:          "test mode wins over ISSUEGRAPH_DOLT_AUTO_START=1",
			testMode:      "1",
			autoStartEnv:  "1",
			wantAutoStart: false,
		},
		{
			name:          "caller true preserved when no overrides",
			currentValue:  true,
			wantAutoStart: true,
		},
		{
			// Config opt-out must still win when callers pass current=true.
			name:             "config.yaml opt-out wins over caller true",
			currentValue:     true,
			doltAutoStartCfg: "false",
			wantAutoStart:    false,
		},
		{
			name:          "test mode overrides caller true",
			testMode:      "1",
			currentValue:  true,
			wantAutoStart: false,
		},
		{
			name:          "ISSUEGRAPH_DOLT_AUTO_START=0 overrides caller true",
			autoStartEnv:  "0",
			currentValue:  true,
			wantAutoStart: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ISSUEGRAPH_TEST_MODE", tc.testMode)
			t.Setenv("ISSUEGRAPH_DOLT_AUTO_START", tc.autoStartEnv)

			got := resolveAutoStart(tc.currentValue, tc.doltAutoStartCfg, ServerModeOwned)
			if got != tc.wantAutoStart {
				t.Errorf("resolveAutoStart(current=%v, configVal=%q, mode=Owned) = %v, want %v",
					tc.currentValue, tc.doltAutoStartCfg, got, tc.wantAutoStart)
			}
		})
	}
}

// TestApplyCLIAutoStart_RespectsExternalMode verifies that an external-mode
// repo (metadata.json with explicit dolt_server_port) suppresses the CLI
// auto-start path, preventing the shadow-database fallback when the
// configured external server is transiently unreachable.
//
// Regression for the case where ApplyCLIAutoStart hardcoded ServerModeOwned
// and bypassed resolveAutoStart's external-mode check, so any issuegraph command in
// an external-mode repo could spawn a fallback embedded server when the
// configured server didn't respond (e.g. during a service cutover).
func TestApplyCLIAutoStart_RespectsExternalMode(t *testing.T) {
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
	t.Setenv("ISSUEGRAPH_DOLT_AUTO_START", "")
	t.Setenv("ISSUEGRAPH_TEST_MODE", "")

	issuegraphDir := t.TempDir()
	// metadata.json with explicit dolt_server_port → ServerModeExternal.
	cfg := configfile.DefaultConfig()
	cfg.DoltMode = configfile.DoltModeServer
	cfg.DoltServerPort = 3399
	if err := cfg.Save(issuegraphDir); err != nil {
		t.Fatalf("save metadata.json: %v", err)
	}

	if got := doltserver.ResolveServerMode(issuegraphDir); got != doltserver.ServerModeExternal {
		t.Fatalf("ResolveServerMode = %v, want External (preflight)", got)
	}

	// No config.yaml dolt.auto-start set — relies entirely on External-
	// mode suppression. Pre-fix this would return AutoStart=true (default).
	storeCfg := &Config{}
	ApplyCLIAutoStart(issuegraphDir, storeCfg)
	if storeCfg.AutoStart {
		t.Errorf("ApplyCLIAutoStart set AutoStart=true in external-mode repo; want false (shadow database protection)")
	}
}

func TestApplyCLIAutoStart_DisableAutoStartWins(t *testing.T) {
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
	t.Setenv("ISSUEGRAPH_DOLT_AUTO_START", "")
	t.Setenv("ISSUEGRAPH_TEST_MODE", "")

	issuegraphDir := t.TempDir()
	if err := configfile.DefaultConfig().Save(issuegraphDir); err != nil {
		t.Fatalf("save metadata.json: %v", err)
	}

	storeCfg := &Config{DisableAutoStart: true}
	ApplyCLIAutoStart(issuegraphDir, storeCfg)
	if storeCfg.AutoStart {
		t.Fatal("DisableAutoStart should suppress CLI auto-start defaults")
	}
}

func TestConfigConstructorsRejectNonDoltBackends(t *testing.T) {
	for _, backend := range []string{
		configfile.BackendSQLite,
		configfile.BackendPostgres,
		configfile.BackendMySQL,
		"mystery",
	} {
		t.Run(backend, func(t *testing.T) {
			issuegraphDir := t.TempDir()
			if err := (&configfile.Config{Backend: backend}).Save(issuegraphDir); err != nil {
				t.Fatalf("save metadata: %v", err)
			}

			constructors := map[string]func() (*DoltStore, error){
				"standard": func() (*DoltStore, error) {
					return NewFromConfigWithOptions(t.Context(), issuegraphDir, &Config{DisableAutoStart: true})
				},
				"cli": func() (*DoltStore, error) {
					return NewFromConfigWithCLIOptions(t.Context(), issuegraphDir, &Config{DisableAutoStart: true})
				},
			}
			for name, construct := range constructors {
				t.Run(name, func(t *testing.T) {
					store, err := construct()
					if store != nil {
						_ = store.Close()
						t.Fatalf("non-Dolt backend %q unexpectedly opened as Dolt", backend)
					}
					if err == nil || !strings.Contains(err.Error(), "cannot be opened as Dolt") {
						t.Fatalf("constructor error = %v, want backend mismatch", err)
					}
					if backend == configfile.BackendPostgres || backend == configfile.BackendMySQL {
						for _, want := range []string{"no longer supported", "was not opened or modified", "issuegraph help init-safety"} {
							if !strings.Contains(err.Error(), want) {
								t.Errorf("removed-backend constructor error missing %q: %v", want, err)
							}
						}
					}
				})
			}
		})
	}
}

func TestCLIDirUsesSharedDoltRootInSharedServerMode(t *testing.T) {
	sharedRoot := t.TempDir()
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "1")
	t.Setenv("ISSUEGRAPH_SHARED_SERVER_DIR", sharedRoot)

	store := &DoltStore{
		serverMode:    true,
		issuegraphDir: filepath.Join(t.TempDir(), ".issuegraph"),
		dbPath:        filepath.Join(t.TempDir(), ".issuegraph", "dolt"),
		database:      "shared_db",
	}

	want := filepath.Join(sharedRoot, "dolt", "shared_db")
	if got := store.CLIDir(); got != want {
		t.Fatalf("CLIDir() = %q, want %q", got, want)
	}
}

func TestCLIDirUsesDbPathOutsideSharedServerMode(t *testing.T) {
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "0")

	dbPath := filepath.Join(t.TempDir(), ".issuegraph", "dolt")
	store := &DoltStore{
		serverMode:    true,
		issuegraphDir: filepath.Join(t.TempDir(), ".issuegraph"),
		dbPath:        dbPath,
		database:      "local_db",
	}

	want := filepath.Join(dbPath, "local_db")
	if got := store.CLIDir(); got != want {
		t.Fatalf("CLIDir() = %q, want %q", got, want)
	}
}

func TestApplyResolvedConfig(t *testing.T) {
	t.Run("fills server config for legacy metadata without dolt_mode", func(t *testing.T) {
		issuegraphDir := t.TempDir()
		fileCfg := &configfile.Config{
			Backend:      configfile.BackendDolt,
			DoltDatabase: "issuegraph_codex",
			ProjectID:    "proj-123",
		}
		cfg := &Config{}

		if err := applyResolvedConfig(context.Background(), issuegraphDir, fileCfg, cfg); err != nil {
			t.Fatalf("applyResolvedConfig: %v", err)
		}

		if cfg.IssueGraphDir != issuegraphDir {
			t.Fatalf("IssueGraphDir = %q, want %q", cfg.IssueGraphDir, issuegraphDir)
		}
		if cfg.Path != fileCfg.DatabasePath(issuegraphDir) {
			t.Fatalf("Path = %q, want %q", cfg.Path, fileCfg.DatabasePath(issuegraphDir))
		}
		if cfg.Database != "issuegraph_codex" {
			t.Fatalf("Database = %q, want issuegraph_codex", cfg.Database)
		}
		if cfg.ServerHost != fileCfg.GetDoltServerHost() {
			t.Fatalf("ServerHost = %q, want %q", cfg.ServerHost, fileCfg.GetDoltServerHost())
		}
		if cfg.ServerUser != fileCfg.GetDoltServerUser() {
			t.Fatalf("ServerUser = %q, want %q", cfg.ServerUser, fileCfg.GetDoltServerUser())
		}
		wantPort := doltserver.DefaultConfig(issuegraphDir).Port
		if cfg.ServerPort != wantPort {
			t.Fatalf("ServerPort = %d, want %d", cfg.ServerPort, wantPort)
		}
	})

	t.Run("warns when data-dir set in server mode (GH#2438)", func(t *testing.T) {
		issuegraphDir := t.TempDir()
		fileCfg := &configfile.Config{
			Backend:      configfile.BackendDolt,
			DoltMode:     configfile.DoltModeServer,
			DoltDatabase: "issuegraph_CodeWriter7",
			DoltDataDir:  "/some/stale/path",
		}
		cfg := &Config{}

		// Capture stderr to verify warning
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		if err := applyResolvedConfig(context.Background(), issuegraphDir, fileCfg, cfg); err != nil {
			t.Fatalf("applyResolvedConfig: %v", err)
		}

		w.Close()
		os.Stderr = oldStderr
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)

		output := buf.String()
		if !strings.Contains(output, "dolt_data_dir is set") {
			t.Errorf("expected data-dir warning in server mode, got: %q", output)
		}
		if !strings.Contains(output, "server mode") {
			t.Errorf("expected 'server mode' in warning, got: %q", output)
		}
	})

	t.Run("no warning when data-dir empty in server mode", func(t *testing.T) {
		issuegraphDir := t.TempDir()
		fileCfg := &configfile.Config{
			Backend:      configfile.BackendDolt,
			DoltMode:     configfile.DoltModeServer,
			DoltDatabase: "issuegraph_CodeWriter7",
		}
		cfg := &Config{}

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		if err := applyResolvedConfig(context.Background(), issuegraphDir, fileCfg, cfg); err != nil {
			t.Fatalf("applyResolvedConfig: %v", err)
		}

		w.Close()
		os.Stderr = oldStderr
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)

		if buf.Len() > 0 {
			t.Errorf("expected no warning when data-dir is empty, got: %q", buf.String())
		}
	})

	t.Run("preserves caller overrides", func(t *testing.T) {
		issuegraphDir := t.TempDir()
		fileCfg := &configfile.Config{
			Backend:      configfile.BackendDolt,
			DoltDatabase: "issuegraph_codex",
		}
		cfg := &Config{
			IssueGraphDir: "/override/.issuegraph",
			Database:      "caller_db",
			ServerHost:    "10.0.0.9",
			ServerPort:    15432,
			ServerUser:    "custom",
		}

		if err := applyResolvedConfig(context.Background(), issuegraphDir, fileCfg, cfg); err != nil {
			t.Fatalf("applyResolvedConfig: %v", err)
		}

		if cfg.IssueGraphDir != "/override/.issuegraph" {
			t.Fatalf("IssueGraphDir override lost: %q", cfg.IssueGraphDir)
		}
		if cfg.Database != "caller_db" {
			t.Fatalf("Database override lost: %q", cfg.Database)
		}
		if cfg.ServerHost != "10.0.0.9" {
			t.Fatalf("ServerHost override lost: %q", cfg.ServerHost)
		}
		if cfg.ServerPort != 15432 {
			t.Fatalf("ServerPort override lost: %d", cfg.ServerPort)
		}
		if cfg.ServerUser != "custom" {
			t.Fatalf("ServerUser override lost: %q", cfg.ServerUser)
		}
	})

	t.Run("pool timeouts populate from env vars (bd-vz0y9)", func(t *testing.T) {
		t.Setenv("ISSUEGRAPH_DOLT_POOL_READ_TIMEOUT", "90s")
		t.Setenv("ISSUEGRAPH_DOLT_POOL_WRITE_TIMEOUT", "45")
		cfg := &Config{}

		if err := applyResolvedConfig(context.Background(), t.TempDir(), &configfile.Config{Backend: configfile.BackendDolt}, cfg); err != nil {
			t.Fatalf("applyResolvedConfig: %v", err)
		}

		if cfg.PoolReadTimeout != 90*time.Second {
			t.Fatalf("PoolReadTimeout = %v, want 90s", cfg.PoolReadTimeout)
		}
		if cfg.PoolWriteTimeout != 45*time.Second {
			t.Fatalf("PoolWriteTimeout = %v, want 45s (bare number = seconds)", cfg.PoolWriteTimeout)
		}
	})

	t.Run("caller-set pool timeouts win over env vars", func(t *testing.T) {
		t.Setenv("ISSUEGRAPH_DOLT_POOL_READ_TIMEOUT", "90s")
		cfg := &Config{PoolReadTimeout: 2 * time.Minute}

		if err := applyResolvedConfig(context.Background(), t.TempDir(), &configfile.Config{Backend: configfile.BackendDolt}, cfg); err != nil {
			t.Fatalf("applyResolvedConfig: %v", err)
		}

		if cfg.PoolReadTimeout != 2*time.Minute {
			t.Fatalf("PoolReadTimeout = %v, want caller's 2m", cfg.PoolReadTimeout)
		}
	})

	t.Run("propagates ServerPortSource and ServerPortSharedServer alongside ServerPort (be-9tju)", func(t *testing.T) {
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
		issuegraphDir := t.TempDir()
		if err := doltserver.EnsurePortFile(issuegraphDir, 14567); err != nil {
			t.Fatalf("EnsurePortFile: %v", err)
		}
		fileCfg := &configfile.Config{Backend: configfile.BackendDolt}
		cfg := &Config{}

		if err := applyResolvedConfig(context.Background(), issuegraphDir, fileCfg, cfg); err != nil {
			t.Fatalf("applyResolvedConfig: %v", err)
		}

		if cfg.ServerPort != 14567 {
			t.Fatalf("ServerPort = %d, want 14567", cfg.ServerPort)
		}
		if cfg.ServerPortSource != doltserver.PortSourcePortFile {
			t.Fatalf("ServerPortSource = %q, want %q (applyResolvedConfig must propagate doltserver.DefaultConfig's PortSource, not just Port)", cfg.ServerPortSource, doltserver.PortSourcePortFile)
		}
		if cfg.ServerPortSharedServer != false {
			t.Fatalf("ServerPortSharedServer = %v, want false", cfg.ServerPortSharedServer)
		}
	})

	t.Run("chained through applyConfigDefaults: port-file resolution is never mislabeled caller-explicit (be-9tju)", func(t *testing.T) {
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
		issuegraphDir := t.TempDir()
		if err := doltserver.EnsurePortFile(issuegraphDir, 14567); err != nil {
			t.Fatalf("EnsurePortFile: %v", err)
		}
		fileCfg := &configfile.Config{Backend: configfile.BackendDolt}
		cfg := &Config{}

		if err := applyResolvedConfig(context.Background(), issuegraphDir, fileCfg, cfg); err != nil {
			t.Fatalf("applyResolvedConfig: %v", err)
		}
		applyConfigDefaults(cfg)

		if cfg.ServerPortSource != doltserver.PortSourcePortFile {
			t.Fatalf("ServerPortSource = %q, want %q (a port-file-resolved port must not be mislabeled caller_explicit — GH#4052 auto-start fail-closed logic depends on this)", cfg.ServerPortSource, doltserver.PortSourcePortFile)
		}
		if cfg.ServerPort != 14567 {
			t.Fatalf("ServerPort = %d, want 14567 (unchanged)", cfg.ServerPort)
		}
	})

	t.Run("legacy ISSUEGRAPH_DOLT_PORT redirects away from a non-authoritative port-file port (be-9tju)", func(t *testing.T) {
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_PORT", "43211")
		t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
		issuegraphDir := t.TempDir()
		if err := doltserver.EnsurePortFile(issuegraphDir, 14567); err != nil {
			t.Fatalf("EnsurePortFile: %v", err)
		}
		fileCfg := &configfile.Config{Backend: configfile.BackendDolt}
		cfg := &Config{}

		if err := applyResolvedConfig(context.Background(), issuegraphDir, fileCfg, cfg); err != nil {
			t.Fatalf("applyResolvedConfig: %v", err)
		}
		applyConfigDefaults(cfg)

		if cfg.ServerPort != 43211 {
			t.Fatalf("ServerPort = %d, want 43211 (legacy ISSUEGRAPH_DOLT_PORT must override a non-authoritative port-file-resolved port — regression of hq-27t bug class)", cfg.ServerPort)
		}
		if cfg.ServerPortSource != doltserver.PortSourceEnv {
			t.Fatalf("ServerPortSource = %q, want %q", cfg.ServerPortSource, doltserver.PortSourceEnv)
		}
	})

	t.Run("genuinely caller-explicit ServerPort still outranks ambient ISSUEGRAPH_DOLT_SERVER_PORT env (be-wf9a.1 regression safety)", func(t *testing.T) {
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "43211")
		t.Setenv("ISSUEGRAPH_DOLT_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
		issuegraphDir := t.TempDir()
		fileCfg := &configfile.Config{Backend: configfile.BackendDolt}
		cfg := &Config{ServerPort: 15000}

		if err := applyResolvedConfig(context.Background(), issuegraphDir, fileCfg, cfg); err != nil {
			t.Fatalf("applyResolvedConfig: %v", err)
		}
		applyConfigDefaults(cfg)

		if cfg.ServerPort != 15000 {
			t.Fatalf("ServerPort = %d, want 15000 (caller-explicit preset must outrank ambient env)", cfg.ServerPort)
		}
		if cfg.ServerPortSource != doltserver.PortSourceCallerExplicit {
			t.Fatalf("ServerPortSource = %q, want %q", cfg.ServerPortSource, doltserver.PortSourceCallerExplicit)
		}
	})
}
