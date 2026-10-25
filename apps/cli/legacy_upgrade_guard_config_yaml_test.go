package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/config"
)

// bindSelectedWorkspaceConfig loads issuegraphDir's config.yaml into the process
// config state the way PersistentPreRunE does — prepareSelectedCommandContext
// binds the discovered target before admission precisely so the guard reads the
// selected workspace's config rather than the caller's. The Dolt env vars are
// pinned off so a developer's ambient shell cannot decide a mode for the guard.
func bindSelectedWorkspaceConfig(t *testing.T, issuegraphDir string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ISSUEGRAPH_TEST_IGNORE_REPO_CONFIG", "1")
	t.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "0")
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "0")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_HOST", "")

	config.ResetForTesting()
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize() = %v", err)
	}
	t.Cleanup(config.ResetForTesting)
}

// writeDoltRootWorkspace lays down the ambiguous shape the guard classifies: a
// .issuegraph directory that owns a local Dolt root, plus whatever metadata.json,
// config.yaml and version witness the case needs. An empty string means "do not
// write this file at all".
func writeDoltRootWorkspace(t *testing.T, metadata, configYaml, version string) string {
	t.Helper()
	issuegraphDir := t.TempDir()
	write := func(name, contents string) {
		if contents == "" {
			return
		}
		if err := os.WriteFile(filepath.Join(issuegraphDir, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("metadata.json", metadata)
	write("config.yaml", configYaml)
	if version != "" {
		if err := writeLocalVersion(filepath.Join(issuegraphDir, localVersionFile), version); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(issuegraphDir, "dolt"), 0o700); err != nil {
		t.Fatal(err)
	}
	bindSelectedWorkspaceConfig(t, issuegraphDir)
	return issuegraphDir
}

// configYamlServerMode reproduces the gc-provisioned config.yaml: dolt.mode is
// declared as a flat dotted key beside a nested dolt map, which is how issuegraph's own
// `issuegraph config set` writer emits it.
const configYamlServerMode = "issue_prefix: gc\n" +
	"dolt:\n" +
	"  disable-event-flush: true\n" +
	"dolt.mode: server\n"

// TestLegacyUpgradeGuardAdmitsConfigYamlServerModeWorkspace pins the shape the
// v1.3.0 release candidate refused after creating it itself: dolt.mode declared
// in .issuegraph/config.yaml, a local .issuegraph/dolt root, a current-era version
// witness, and no metadata.json. LoadForDiscovery reads metadata.json only, so
// the guard classified a server workspace as embedded — and an embedded
// workspace owning a .issuegraph/dolt root is exactly what it refuses.
func TestLegacyUpgradeGuardAdmitsConfigYamlServerModeWorkspace(t *testing.T) {
	versions := []string{
		"1.3.0",
		"1.3.0-rc.1",
		"1.1.0",
		"v1.1.1-0.20260805093327-bf97b73749ac",
	}

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			warnings := captureLegacyUpgradeWarnings(t)
			issuegraphDir := writeDoltRootWorkspace(t, "", configYamlServerMode, version)

			if err := guardLegacyUpgradeWorkspace(issuegraphDir); err != nil {
				t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want nil", err)
			}
			if warnings.Len() != 0 {
				t.Fatalf("guard warned about a readable witness: %q", warnings.String())
			}
		})
	}

	t.Run("nested dolt.mode mapping", func(t *testing.T) {
		captureLegacyUpgradeWarnings(t)
		issuegraphDir := writeDoltRootWorkspace(t, "", "dolt:\n  mode: server\n", "1.3.0")

		if err := guardLegacyUpgradeWorkspace(issuegraphDir); err != nil {
			t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want nil", err)
		}
	})

	t.Run("unreadable witness warns and opens", func(t *testing.T) {
		warnings := captureLegacyUpgradeWarnings(t)
		issuegraphDir := writeDoltRootWorkspace(t, "", configYamlServerMode, "not-a-version")

		if err := guardLegacyUpgradeWorkspace(issuegraphDir); err != nil {
			t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want nil", err)
		}
		if warnings.Len() == 0 {
			t.Fatal("guard admitted an unreadable witness without warning")
		}
	})
}

// TestLegacyUpgradeGuardStillRefusesLegacyConfigYamlServerWorkspace proves the
// config.yaml layer classifies the mode without relaxing the era check: a
// workspace that declares server mode there and carries pre-1.0 evidence — or
// no evidence at all — is still refused.
func TestLegacyUpgradeGuardStillRefusesLegacyConfigYamlServerWorkspace(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{name: "historical server era witness", version: "0.62.0"},
		{name: "historical embedded era witness", version: "0.49.6"},
		{name: "four component pre-1.0 witness", version: "0.62.0.1"},
		{name: "no witness at all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captureLegacyUpgradeWarnings(t)
			issuegraphDir := writeDoltRootWorkspace(t, "", configYamlServerMode, tt.version)

			if err := guardLegacyUpgradeWorkspace(issuegraphDir); !isLegacyUpgradeRefusal(err) {
				t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want migration refusal", err)
			}
		})
	}

	t.Run("metadata embedded mode outranks config.yaml server mode", func(t *testing.T) {
		captureLegacyUpgradeWarnings(t)
		// metadata.json's dolt_mode is the highest-priority layer, so this stays
		// an embedded workspace — and the embedded arm deliberately does not
		// trust an unreadable witness.
		issuegraphDir := writeDoltRootWorkspace(t,
			`{"backend":"dolt","dolt_mode":"embedded"}`, configYamlServerMode, "not-a-version")

		if err := guardLegacyUpgradeWorkspace(issuegraphDir); !isLegacyUpgradeRefusal(err) {
			t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want migration refusal", err)
		}
	})
}

// TestCurrentVersionWitnessVetoesLegacyVerdictInEveryMode pins the general
// safety net: a witness naming issuegraph 1.0 or later can only have been written by a
// current-era binary, and only after this guard already admitted the workspace,
// so it settles the era regardless of which mode the workspace resolves to. The
// escape hatch used to live inside the server-mode arm, where a misclassified
// workspace could never reach it.
func TestCurrentVersionWitnessVetoesLegacyVerdictInEveryMode(t *testing.T) {
	modes := []struct {
		name     string
		metadata string
	}{
		{name: "metadata-less"},
		{name: "blank mode", metadata: `{"backend":"dolt"}`},
		{name: "explicit embedded mode", metadata: `{"backend":"dolt","dolt_mode":"embedded"}`},
		{name: "explicit server mode", metadata: `{"backend":"dolt","dolt_mode":"server"}`},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			captureLegacyUpgradeWarnings(t)
			issuegraphDir := writeDoltRootWorkspace(t, mode.metadata, "", "1.3.0")

			if err := guardLegacyUpgradeWorkspace(issuegraphDir); err != nil {
				t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want nil", err)
			}
		})

		t.Run(mode.name+"/pre-1.0 witness still refuses", func(t *testing.T) {
			captureLegacyUpgradeWarnings(t)
			issuegraphDir := writeDoltRootWorkspace(t, mode.metadata, "", "0.62.21")

			if err := guardLegacyUpgradeWorkspace(issuegraphDir); !isLegacyUpgradeRefusal(err) {
				t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want migration refusal", err)
			}
		})

		t.Run(mode.name+"/missing witness still refuses", func(t *testing.T) {
			captureLegacyUpgradeWarnings(t)
			issuegraphDir := writeDoltRootWorkspace(t, mode.metadata, "", "")

			if err := guardLegacyUpgradeWorkspace(issuegraphDir); !isLegacyUpgradeRefusal(err) {
				t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want migration refusal", err)
			}
		})
	}

	// The unreadable-witness admission is deliberately NOT hoisted: outside
	// server mode a .issuegraph/dolt root is itself strong legacy evidence, and
	// TestLegacyUpgradeGuardRefusesOldDoltRootWithoutTrustingVersionWitness
	// pins that arm. Keep the boundary explicit so a later cleanup cannot widen
	// it by accident.
	t.Run("embedded mode still refuses an unreadable witness", func(t *testing.T) {
		captureLegacyUpgradeWarnings(t)
		issuegraphDir := writeDoltRootWorkspace(t, `{"backend":"dolt"}`, "", "not-a-version")

		if err := guardLegacyUpgradeWorkspace(issuegraphDir); !isLegacyUpgradeRefusal(err) {
			t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want migration refusal", err)
		}
	})
}

// TestLegacyUpgradeGuardSharedServerAdmissionSurvivesServerClassification keeps
// the shared-server escape reachable now that a config.yaml `dolt.mode: server`
// workspace resolves to the server arm. Before the hoist this admission lived
// only in the embedded arm, so classifying the same workspace correctly would
// have turned a workspace issuegraph admits today into a refusal — trading one
// false-refusal class for another.
func TestLegacyUpgradeGuardSharedServerAdmissionSurvivesServerClassification(t *testing.T) {
	tests := []struct {
		name        string
		configYaml  string
		version     string
		wantRefusal bool
	}{
		{name: "config.yaml server mode without witness", configYaml: configYamlServerMode + "dolt.shared-server: true\n"},
		{name: "shared server only", configYaml: "dolt.shared-server: true\n"},
		{name: "config.yaml server mode with historical server witness",
			configYaml: configYamlServerMode + "dolt.shared-server: true\n", version: "0.62.0", wantRefusal: true},
		{name: "not shared and no witness", configYaml: configYamlServerMode, wantRefusal: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captureLegacyUpgradeWarnings(t)
			issuegraphDir := writeDoltRootWorkspace(t, "", tt.configYaml, tt.version)

			err := guardLegacyUpgradeWorkspace(issuegraphDir)
			if tt.wantRefusal && !isLegacyUpgradeRefusal(err) {
				t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want migration refusal", err)
			}
			if !tt.wantRefusal && err != nil {
				t.Fatalf("guardLegacyUpgradeWorkspace() = %v, want nil", err)
			}
		})
	}
}
