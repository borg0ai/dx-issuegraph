//go:build cgo

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestLegacyUpgradeGuardRefusesBeforeMutatingHistoricalWorkspace(t *testing.T) {
	issuegraph := buildIssueGraphUnderTest(t)

	layouts := []struct {
		name       string
		wantReason string
		setup      func(t *testing.T, issuegraphDir string)
	}{
		{
			name:       "legacy server Dolt",
			wantReason: "legacy Dolt server workspace",
			setup: func(t *testing.T, issuegraphDir string) {
				t.Helper()
				writeFile(t, filepath.Join(issuegraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_mode":"server"}`))
				writeFile(t, filepath.Join(issuegraphDir, localVersionFile), []byte("0.62.0\n"))
				if err := os.Mkdir(filepath.Join(issuegraphDir, "dolt"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:       "metadata-less v0.9.1 SQLite",
			wantReason: "historical SQLite workspace",
			setup: func(t *testing.T, issuegraphDir string) {
				t.Helper()
				writeFile(t, filepath.Join(issuegraphDir, "vc.db"), []byte("SQLite format 3\x00"))
			},
		},
		{
			name:       "blank-mode old Dolt root without version witness",
			wantReason: "legacy Dolt workspace",
			setup: func(t *testing.T, issuegraphDir string) {
				t.Helper()
				writeFile(t, filepath.Join(issuegraphDir, "metadata.json"), []byte(`{"backend":"dolt"}`))
				if err := os.Mkdir(filepath.Join(issuegraphDir, "dolt"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:       "metadata-less old Dolt root",
			wantReason: "legacy Dolt workspace",
			setup: func(t *testing.T, issuegraphDir string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(issuegraphDir, "dolt"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	commands := []struct {
		name     string
		args     []string
		wantOK   bool
		wantJSON bool
	}{
		{name: "list", args: []string{"list", "--json", "--limit", "0", "--all"}},
		{name: "context", args: []string{"context", "--json"}},
		{name: "dolt start", args: []string{"dolt", "start"}},
		{name: "force init", args: []string{"init", "--force", "--quiet", "--non-interactive", "--skip-hooks", "--skip-agents"}},
		{name: "doctor", args: []string{"doctor", "--json"}, wantOK: true, wantJSON: true},
	}

	for _, layout := range layouts {
		t.Run(layout.name, func(t *testing.T) {
			repoDir := t.TempDir()
			initGitRepo(t, repoDir)
			issuegraphDir := filepath.Join(repoDir, ".issuegraph")
			layout.setup(t, issuegraphDir)
			if err := os.Chmod(issuegraphDir, 0o700); err != nil {
				t.Fatal(err)
			}

			for _, tc := range commands {
				t.Run(tc.name, func(t *testing.T) {
					before := legacyUpgradeTreeDigest(t, issuegraphDir)
					commandCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					cmd := exec.CommandContext(commandCtx, issuegraph, tc.args...)
					cmd.Dir = repoDir
					// Pin the mode inputs the guard now reads through
					// IsDoltServerMode: this child inherits os.Environ(), so
					// without them a developer's shell decides which refusal
					// text these historical layouts produce.
					cmd.Env = append(os.Environ(),
						"ISSUEGRAPH_DISABLE_METRICS=1",
						"ISSUEGRAPH_DOLT_AUTO_START=0",
						"ISSUEGRAPH_DOLT_SERVER_MODE=0",
						"ISSUEGRAPH_DOLT_SHARED_SERVER=0",
						"ISSUEGRAPH_DOLT_SERVER_HOST=",
					)
					var stdout, stderr bytes.Buffer
					cmd.Stdout = &stdout
					cmd.Stderr = &stderr
					err := cmd.Run()
					commandContextErr := commandCtx.Err()
					cancel()
					output := stdout.String() + stderr.String()
					if commandContextErr != nil {
						t.Fatalf("issuegraph %s did not refuse before the command deadline: %v\n%s", strings.Join(tc.args, " "), commandContextErr, output)
					}
					if (err == nil) != tc.wantOK {
						t.Fatalf("issuegraph %s error = %v, want success=%v\n%s", strings.Join(tc.args, " "), err, tc.wantOK, output)
					}
					if !strings.Contains(output, layout.wantReason) ||
						!strings.Contains(output, "explicit migration is required") {
						t.Fatalf("issuegraph %s did not report the expected migration refusal:\n%s", strings.Join(tc.args, " "), output)
					}
					if tc.wantJSON {
						var payload map[string]any
						if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
							t.Fatalf("doctor stdout is not valid JSON: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
						}
						if payload["code"] != "legacy_upgrade_required" {
							t.Fatalf("doctor code = %v, want legacy_upgrade_required", payload["code"])
						}
						if payload["guide"] != "docs/getting-started/upgrading.md#cross-era-upgrades" {
							t.Fatalf("doctor guide = %v, want layout-specific cross-era guide", payload["guide"])
						}
						if _, misleading := payload["bridge"]; misleading {
							t.Fatalf("doctor advertised the SQLite bridge for %s", layout.name)
						}
					}
					if after := legacyUpgradeTreeDigest(t, issuegraphDir); after != before {
						t.Fatalf("issuegraph %s mutated legacy source: before=%s after=%s", strings.Join(tc.args, " "), before, after)
					}
				})
			}
		})
	}
}

func TestLegacyNoStoreCommandExemptions(t *testing.T) {
	syntheticRoot := &cobra.Command{Use: "issuegraph"}
	helpCmd := &cobra.Command{Use: "help", Run: func(*cobra.Command, []string) {}}
	completeCmd := &cobra.Command{Use: "__complete", Run: func(*cobra.Command, []string) {}}
	syntheticRoot.AddCommand(helpCmd, completeCmd)
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "root", cmd: rootCmd},
		{name: "version", cmd: versionCmd},
		{name: "schema", cmd: schemaCmd},
		{name: "metrics", cmd: metricsCmd},
		{name: "metrics subcommand", cmd: metricsOffCmd},
		{name: "doctor", cmd: doctorCmd},
		{name: "init", cmd: initCmd},
		{name: "bootstrap", cmd: bootstrapCmd},
		{name: "legacy SQLite reader", cmd: legacySQLiteCmd},
		{name: "help", cmd: helpCmd},
		{name: "shell completion", cmd: completeCmd},
		{name: "non-runnable", cmd: &cobra.Command{Use: "group"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := guardLegacyNoStoreCommand(tt.cmd, t.TempDir()); err != nil {
				t.Fatalf("guardLegacyNoStoreCommand() = %v, want exemption", err)
			}
		})
	}

	issuegraphDir := t.TempDir()
	writeFile(t, filepath.Join(issuegraphDir, "vc.db"), []byte("SQLite format 3\x00"))
	if err := guardLegacyNoStoreCommand(contextCmd, issuegraphDir); !isLegacyUpgradeRefusal(err) {
		t.Fatalf("context admission = %v, want legacy migration refusal", err)
	}
}

func TestVersionLeavesHistoricalWorkspaceUnchanged(t *testing.T) {
	issuegraph := buildIssueGraphUnderTest(t)
	repoDir := t.TempDir()
	initGitRepo(t, repoDir)
	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(issuegraphDir, "vc.db"), []byte("SQLite format 3\x00"))
	before := legacyUpgradeTreeDigest(t, issuegraphDir)

	cmd := exec.Command(issuegraph, "version")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "ISSUEGRAPH_DISABLE_METRICS=1", "ISSUEGRAPH_DOLT_AUTO_START=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("issuegraph version failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "issuegraph version") {
		t.Fatalf("issuegraph version output missing version: %s", output)
	}
	if after := legacyUpgradeTreeDigest(t, issuegraphDir); after != before {
		t.Fatalf("issuegraph version mutated historical source: before=%s after=%s", before, after)
	}
}

func TestLegacyBackendConfigRefusesBeforeMetadataMigration(t *testing.T) {
	issuegraph := buildIssueGraphUnderTest(t)
	commands := [][]string{
		{"list", "--json", "--limit", "0", "--all"},
		{"init", "--force", "--quiet", "--non-interactive", "--skip-hooks", "--skip-agents"},
	}

	for _, backend := range []string{"postgres", "mysql", "mystery"} {
		t.Run(backend, func(t *testing.T) {
			repoDir := t.TempDir()
			initGitRepo(t, repoDir)
			issuegraphDir := filepath.Join(repoDir, ".issuegraph")
			if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
				t.Fatal(err)
			}
			legacyPath := filepath.Join(issuegraphDir, "config.json")
			legacy := []byte("{\n  \"backend\": \"" + backend + "\"\n}\n")
			writeFile(t, legacyPath, legacy)

			for _, args := range commands {
				before := legacyUpgradeTreeDigest(t, issuegraphDir)
				cmd := exec.Command(issuegraph, args...)
				cmd.Dir = repoDir
				cmd.Env = append(os.Environ(), "ISSUEGRAPH_DISABLE_METRICS=1", "ISSUEGRAPH_DOLT_AUTO_START=0")
				output, err := cmd.CombinedOutput()
				if err == nil {
					t.Fatalf("issuegraph %s unexpectedly accepted backend %q:\n%s", strings.Join(args, " "), backend, output)
				}
				if !strings.Contains(string(output), backend) {
					t.Fatalf("issuegraph %s error did not identify backend %q:\n%s", strings.Join(args, " "), backend, output)
				}
				if after := legacyUpgradeTreeDigest(t, issuegraphDir); after != before {
					t.Fatalf("issuegraph %s mutated legacy config: before=%s after=%s", strings.Join(args, " "), before, after)
				}
				after, readErr := os.ReadFile(legacyPath)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if !bytes.Equal(after, legacy) {
					t.Fatalf("issuegraph %s rewrote config.json: got %q, want %q", strings.Join(args, " "), after, legacy)
				}
				if _, statErr := os.Stat(filepath.Join(issuegraphDir, "metadata.json")); !os.IsNotExist(statErr) {
					t.Fatalf("issuegraph %s created metadata.json: %v", strings.Join(args, " "), statErr)
				}
			}
		})
	}
}

func TestBootstrapRefusesLegacyServerBeforeMigratingLegacyConfig(t *testing.T) {
	issuegraph := buildIssueGraphUnderTest(t)
	repoDir := t.TempDir()
	initGitRepo(t, repoDir)
	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
		t.Fatal(err)
	}

	legacyPath := filepath.Join(issuegraphDir, "config.json")
	legacy := []byte("{\n  \"backend\": \"dolt\",\n  \"dolt_mode\": \"server\"\n}\n")
	writeFile(t, legacyPath, legacy)
	writeFile(t, filepath.Join(issuegraphDir, localVersionFile), []byte("0.62.0\n"))
	if err := os.Mkdir(filepath.Join(issuegraphDir, "dolt"), 0o700); err != nil {
		t.Fatal(err)
	}

	before := legacyUpgradeTreeDigest(t, issuegraphDir)
	cmd := exec.Command(issuegraph, "bootstrap", "--dry-run", "--yes")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "ISSUEGRAPH_DISABLE_METRICS=1", "ISSUEGRAPH_DOLT_AUTO_START=0")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("issuegraph bootstrap unexpectedly accepted historical server workspace:\n%s", output)
	}
	if !strings.Contains(string(output), "legacy Dolt server workspace") ||
		!strings.Contains(string(output), "explicit migration is required") {
		t.Fatalf("issuegraph bootstrap did not report the expected migration refusal:\n%s", output)
	}
	if after := legacyUpgradeTreeDigest(t, issuegraphDir); after != before {
		t.Fatalf("issuegraph bootstrap mutated legacy source: before=%s after=%s", before, after)
	}
	after, readErr := os.ReadFile(legacyPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(after, legacy) {
		t.Fatalf("issuegraph bootstrap rewrote config.json: got %q, want %q", after, legacy)
	}
	if _, statErr := os.Stat(filepath.Join(issuegraphDir, "metadata.json")); !os.IsNotExist(statErr) {
		t.Fatalf("issuegraph bootstrap created metadata.json: %v", statErr)
	}
}

func TestBootstrapRefusesLegacyAncestorConfigWithoutMigratingIt(t *testing.T) {
	issuegraph := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	ancestorIssueGraphDir := filepath.Join(root, ".issuegraph")
	if err := os.MkdirAll(ancestorIssueGraphDir, 0o700); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(ancestorIssueGraphDir, "config.json")
	legacy := []byte("{\n  \"backend\": \"dolt\",\n  \"dolt_mode\": \"server\"\n}\n")
	writeFile(t, legacyPath, legacy)
	writeFile(t, filepath.Join(ancestorIssueGraphDir, localVersionFile), []byte("0.62.0\n"))
	if err := os.Mkdir(filepath.Join(ancestorIssueGraphDir, "dolt"), 0o700); err != nil {
		t.Fatal(err)
	}

	selectedIssueGraphDir := filepath.Join(root, "rig", ".issuegraph")
	if err := os.MkdirAll(selectedIssueGraphDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(selectedIssueGraphDir, "dolt"), 0o700); err != nil {
		t.Fatal(err)
	}
	before := legacyUpgradeTreeDigest(t, ancestorIssueGraphDir)
	cmd := exec.Command(issuegraph, "bootstrap", "--dry-run", "--yes")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"ISSUEGRAPH_DISABLE_METRICS=1",
		"ISSUEGRAPH_DOLT_AUTO_START=0",
		"ISSUEGRAPH_DOLT_SHARED_SERVER=1",
		"ISSUEGRAPH_DIR="+selectedIssueGraphDir,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("issuegraph bootstrap unexpectedly accepted historical ancestor workspace:\n%s", output)
	}
	if !strings.Contains(string(output), "legacy Dolt server workspace") ||
		!strings.Contains(string(output), "explicit migration is required") {
		t.Fatalf("issuegraph bootstrap did not report the expected ancestor migration refusal:\n%s", output)
	}
	if after := legacyUpgradeTreeDigest(t, ancestorIssueGraphDir); after != before {
		t.Fatalf("issuegraph bootstrap mutated legacy ancestor source: before=%s after=%s", before, after)
	}
	after, readErr := os.ReadFile(legacyPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(after, legacy) {
		t.Fatalf("issuegraph bootstrap rewrote ancestor config.json: got %q, want %q", after, legacy)
	}
	if _, statErr := os.Stat(filepath.Join(ancestorIssueGraphDir, "metadata.json")); !os.IsNotExist(statErr) {
		t.Fatalf("issuegraph bootstrap created ancestor metadata.json: %v", statErr)
	}
}

func TestInitProxiedServerRefusesHistoricalExternalWorkspaceBeforeMutation(t *testing.T) {
	issuegraph := buildIssueGraphUnderTest(t)
	repoDir := t.TempDir()
	initGitRepo(t, repoDir)
	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
		t.Fatal(err)
	}
	legacyConfig := []byte("{\n  \"backend\": \"dolt\",\n  \"dolt_mode\": \"server\",\n  \"dolt_database\": \"historical\"\n}\n")
	legacyConfigPath := filepath.Join(issuegraphDir, "config.json")
	writeFile(t, legacyConfigPath, legacyConfig)
	writeFile(t, filepath.Join(issuegraphDir, localVersionFile), []byte("0.62.0\n"))
	before := legacyUpgradeTreeDigest(t, issuegraphDir)
	home := t.TempDir()

	cmd := exec.Command(issuegraph, "init", "--proxied-server", "--quiet", "--non-interactive", "--skip-hooks", "--skip-agents")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(),
		"ISSUEGRAPH_DISABLE_METRICS=1",
		"ISSUEGRAPH_DISABLE_EVENT_FLUSH=1",
		"ISSUEGRAPH_DOLT_AUTO_START=0",
		"HOME="+home,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("issuegraph init --proxied-server unexpectedly accepted historical external workspace:\n%s", output)
	}
	if !strings.Contains(string(output), "legacy Dolt server workspace") ||
		!strings.Contains(string(output), "explicit migration is required") {
		t.Fatalf("issuegraph init --proxied-server did not report the expected migration refusal:\n%s", output)
	}
	if after := legacyUpgradeTreeDigest(t, issuegraphDir); after != before {
		t.Fatalf("issuegraph init --proxied-server mutated historical source: before=%s after=%s", before, after)
	}
	afterConfig, err := os.ReadFile(legacyConfigPath)
	if err != nil {
		t.Fatalf("issuegraph init --proxied-server removed historical config.json: %v", err)
	}
	if !bytes.Equal(afterConfig, legacyConfig) {
		t.Fatalf("issuegraph init --proxied-server rewrote config.json: got %q, want %q", afterConfig, legacyConfig)
	}
	if _, err := os.Stat(filepath.Join(issuegraphDir, "metadata.json")); !os.IsNotExist(err) {
		t.Fatalf("issuegraph init --proxied-server created metadata.json before refusing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(issuegraphDir, "dolt")); !os.IsNotExist(err) {
		t.Fatalf("issuegraph init --proxied-server created local Dolt state: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".issuegraph")); !os.IsNotExist(err) {
		t.Fatalf("issuegraph init --proxied-server created client state before refusing: %v", err)
	}
}

func TestLegacyGuardUsesSelectedTargetSharedServerConfig(t *testing.T) {
	issuegraph := buildIssueGraphUnderTest(t)
	selectors := []struct {
		name string
		args func(target, targetIssueGraphDir string) []string
	}{
		{
			name: "explicit db",
			args: func(_, targetIssueGraphDir string) []string {
				return []string{"--db", filepath.Join(targetIssueGraphDir, "dolt"), "list", "--json", "--limit", "0", "--all"}
			},
		},
		{
			name: "change directory",
			args: func(target, _ string) []string {
				return []string{"-C", target, "list", "--json", "--limit", "0", "--all"}
			},
		},
	}
	tests := []struct {
		name              string
		callerShared      bool
		targetShared      bool
		wantLegacyRefusal bool
	}{
		{name: "target shared caller not", targetShared: true},
		{name: "caller shared target not", callerShared: true, wantLegacyRefusal: true},
	}

	for _, selector := range selectors {
		for _, tt := range tests {
			t.Run(selector.name+"/"+tt.name, func(t *testing.T) {
				caller := t.TempDir()
				callerIssueGraphDir := filepath.Join(caller, ".issuegraph")
				if err := os.MkdirAll(callerIssueGraphDir, 0o700); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(callerIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_mode":"embedded"}`))
				writeFile(t, filepath.Join(callerIssueGraphDir, "config.yaml"), []byte("dolt:\n  shared-server: "+strconv.FormatBool(tt.callerShared)+"\n"))

				target := t.TempDir()
				targetIssueGraphDir := filepath.Join(target, ".issuegraph")
				if err := os.MkdirAll(filepath.Join(targetIssueGraphDir, "dolt"), 0o700); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(targetIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt"}`))
				writeFile(t, filepath.Join(targetIssueGraphDir, "config.yaml"), []byte("dolt:\n  shared-server: "+strconv.FormatBool(tt.targetShared)+"\n"))

				cmd := exec.Command(issuegraph, selector.args(target, targetIssueGraphDir)...)
				cmd.Dir = caller
				cmd.Env = append(os.Environ(),
					"ISSUEGRAPH_DISABLE_METRICS=1",
					"ISSUEGRAPH_DISABLE_EVENT_FLUSH=1",
					"ISSUEGRAPH_DOLT_AUTO_START=0",
					"ISSUEGRAPH_DOLT_SERVER_PORT=59999",
					"ISSUEGRAPH_DOLT_SHARED_SERVER=0",
					"HOME="+t.TempDir(),
				)
				output, err := cmd.CombinedOutput()
				if err == nil {
					t.Fatalf("issuegraph %s target list unexpectedly succeeded:\n%s", selector.name, output)
				}
				hasLegacyRefusal := strings.Contains(string(output), "explicit migration is required")
				if hasLegacyRefusal != tt.wantLegacyRefusal {
					t.Fatalf("issuegraph %s target list legacy refusal=%v, want %v:\n%s", selector.name, hasLegacyRefusal, tt.wantLegacyRefusal, output)
				}
				if !tt.wantLegacyRefusal && !strings.Contains(string(output), "Dolt server") {
					t.Fatalf("issuegraph %s target list did not use the target shared-server config:\n%s", selector.name, output)
				}
			})
		}
	}
}

// TestLegacyGuardAdmitsConfigYamlServerWorkspace runs the real binary against
// the workspace shape gc provisions — `dolt.mode: server` in
// .issuegraph/config.yaml, a local .issuegraph/dolt root, a metadata.json that names no
// dolt_mode — and pins that the guard no longer calls it legacy. The in-process
// guard tests bind the workspace config themselves; only a subprocess proves
// that PersistentPreRunE's own binding puts the selected workspace's
// config.yaml in front of the guard.
//
// The command still fails: the workspace points at a Dolt server that is not
// running. That is the point — reaching the connection attempt means admission
// succeeded and the workspace routed as a server workspace rather than being
// refused or opened as a phantom embedded database.
func TestLegacyGuardAdmitsConfigYamlServerWorkspace(t *testing.T) {
	issuegraph := buildIssueGraphUnderTest(t)
	tests := []struct {
		name              string
		version           string
		wantLegacyRefusal bool
	}{
		{name: "current era witness", version: "1.3.0"},
		{name: "release candidate witness", version: "1.3.0-rc.1"},
		{name: "historical witness still refuses", version: "0.62.0", wantLegacyRefusal: true},
		{name: "missing witness still refuses", wantLegacyRefusal: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := t.TempDir()
			initGitRepo(t, repoDir)
			issuegraphDir := filepath.Join(repoDir, ".issuegraph")
			if err := os.MkdirAll(filepath.Join(issuegraphDir, "dolt"), 0o700); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(issuegraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"hq"}`))
			writeFile(t, filepath.Join(issuegraphDir, "config.yaml"), []byte("dolt:\n  mode: server\n  auto-start: false\n"))
			if tt.version != "" {
				writeFile(t, filepath.Join(issuegraphDir, localVersionFile), []byte(tt.version+"\n"))
			}

			commandCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(commandCtx, issuegraph, "list", "--json", "--limit", "0", "--all")
			cmd.Dir = repoDir
			cmd.Env = append(os.Environ(),
				"ISSUEGRAPH_DISABLE_METRICS=1",
				"ISSUEGRAPH_DISABLE_EVENT_FLUSH=1",
				"ISSUEGRAPH_DOLT_AUTO_START=0",
				"ISSUEGRAPH_DOLT_SERVER_PORT=59999",
				"ISSUEGRAPH_DOLT_SHARED_SERVER=0",
				"HOME="+t.TempDir(),
			)
			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("issuegraph list unexpectedly succeeded against an unreachable server:\n%s", output)
			}
			hasLegacyRefusal := strings.Contains(string(output), "explicit migration is required")
			if hasLegacyRefusal != tt.wantLegacyRefusal {
				t.Fatalf("issuegraph list legacy refusal=%v, want %v:\n%s", hasLegacyRefusal, tt.wantLegacyRefusal, output)
			}
			if !tt.wantLegacyRefusal && !strings.Contains(string(output), "Dolt server") {
				t.Fatalf("admitted workspace did not route as a server workspace:\n%s", output)
			}
			if !tt.wantLegacyRefusal {
				if _, err := os.Stat(filepath.Join(issuegraphDir, "embeddeddolt")); !os.IsNotExist(err) {
					t.Fatalf("admitted server workspace created a phantom embedded database: %v", err)
				}
			}
		})
	}
}

func legacyUpgradeTreeDigest(t *testing.T, root string) string {
	t.Helper()
	var entries []string
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		entries = append(entries, rel+"\x00"+info.Mode().String())
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			entries = append(entries, hex.EncodeToString(sum[:]))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	sum := sha256.Sum256([]byte(strings.Join(entries, "\n")))
	return hex.EncodeToString(sum[:])
}
