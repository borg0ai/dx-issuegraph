package dolt

import (
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/doltserver"
)

// TestApplyResolvedServerPort covers the helper that every hand-built
// dolt.Config now uses to resolve its port. The assertion that matters is not
// the port — it is that the port arrives with the source that produced it, and
// that issuegraph's own bookkeeping never arrives labeled authoritative. Both consumers
// of the label (the env-override gate in applyConfigDefaults, and the
// auto-start fail-closed check in newServerMode) branch on IsAuthoritative, so
// that is asserted directly rather than inferred from the source string.
func TestApplyResolvedServerPort(t *testing.T) {
	t.Run("port file resolves non-authoritative (be-9tju)", func(t *testing.T) {
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
		issuegraphDir := t.TempDir()
		if err := doltserver.EnsurePortFile(issuegraphDir, 14567); err != nil {
			t.Fatalf("EnsurePortFile: %v", err)
		}

		cfg := &Config{}
		ApplyResolvedServerPort(issuegraphDir, cfg)

		if cfg.ServerPort != 14567 {
			t.Fatalf("ServerPort = %d, want 14567", cfg.ServerPort)
		}
		if cfg.ServerPortSource != doltserver.PortSourcePortFile {
			t.Fatalf("ServerPortSource = %q, want %q", cfg.ServerPortSource, doltserver.PortSourcePortFile)
		}
		if cfg.ServerPortSource.IsAuthoritative() {
			t.Fatal("port-file port reported authoritative: the legacy ISSUEGRAPH_DOLT_PORT override would become a silent no-op and auto-start would refuse to retarget a port the user never chose (GH#4052)")
		}
		if cfg.ServerPortSharedServer {
			t.Fatal("ServerPortSharedServer = true in per-project mode")
		}
	})

	t.Run("shared-server default resolves non-authoritative but shared (GH#4052)", func(t *testing.T) {
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
		t.Setenv("HOME", t.TempDir())
		t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "1")

		cfg := &Config{}
		ApplyResolvedServerPort(t.TempDir(), cfg)

		if cfg.ServerPort != doltserver.DefaultSharedServerPort {
			t.Fatalf("ServerPort = %d, want %d", cfg.ServerPort, doltserver.DefaultSharedServerPort)
		}
		if cfg.ServerPortSource.IsAuthoritative() {
			t.Fatalf("ServerPortSource = %q reported authoritative: issuegraph chose 3308 on the user's behalf, so auto-start must not treat it as a configured port", cfg.ServerPortSource)
		}
		// Shared mode still fails closed on retarget — for the orthogonal
		// reason that a repo-local auto-start is a different database.
		if !cfg.ServerPortSharedServer {
			t.Fatal("ServerPortSharedServer = false in shared-server mode: newServerMode would silently retarget to a repo-local server")
		}
	})

	t.Run("env wins and is authoritative", func(t *testing.T) {
		t.Setenv("ISSUEGRAPH_DOLT_PORT", "")
		t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "43211")
		issuegraphDir := t.TempDir()
		if err := doltserver.EnsurePortFile(issuegraphDir, 14567); err != nil {
			t.Fatalf("EnsurePortFile: %v", err)
		}

		cfg := &Config{}
		ApplyResolvedServerPort(issuegraphDir, cfg)

		if cfg.ServerPort != 43211 {
			t.Fatalf("ServerPort = %d, want 43211 (env outranks the port file)", cfg.ServerPort)
		}
		if !cfg.ServerPortSource.IsAuthoritative() {
			t.Fatalf("ServerPortSource = %q, want an authoritative source for an env-set port", cfg.ServerPortSource)
		}
	})
}

// TestApplyResolvedServerPortChainsThroughApplyConfigDefaults is the end-to-end
// statement of the bug class: a port that issuegraph resolved for itself must stay
// overridable by the legacy env spelling. Without the source travelling with
// the port, applyConfigDefaults' "nonzero + unset ⇒ caller_explicit" inference
// promotes the port file to authoritative and the env read below is skipped.
func TestApplyResolvedServerPortChainsThroughApplyConfigDefaults(t *testing.T) {
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
	t.Setenv("ISSUEGRAPH_TEST_MODE", "")
	issuegraphDir := t.TempDir()
	if err := doltserver.EnsurePortFile(issuegraphDir, 14567); err != nil {
		t.Fatalf("EnsurePortFile: %v", err)
	}

	cfg := &Config{IssueGraphDir: issuegraphDir}
	ApplyResolvedServerPort(issuegraphDir, cfg)
	t.Setenv("ISSUEGRAPH_DOLT_PORT", "43211")
	applyConfigDefaults(cfg)

	if cfg.ServerPort != 43211 {
		t.Fatalf("ServerPort = %d, want 43211: legacy ISSUEGRAPH_DOLT_PORT must still override a port-file-resolved port", cfg.ServerPort)
	}
	if cfg.ServerPortSource != doltserver.PortSourceEnv {
		t.Fatalf("ServerPortSource = %q, want %q", cfg.ServerPortSource, doltserver.PortSourceEnv)
	}
}
