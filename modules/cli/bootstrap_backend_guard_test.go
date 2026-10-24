package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
)

func TestBootstrapRejectsRemovedBackendsBeforeWorkspaceWrites(t *testing.T) {
	issuegraph := buildIssueGraphForInitTests(t)

	for _, backend := range []string{configfile.BackendPostgres, configfile.BackendMySQL, configfile.BackendSQLite} {
		backend := backend
		for _, args := range [][]string{{"bootstrap", "--dry-run"}, {"bootstrap", "--yes"}} {
			args := args
			mode := "execute"
			if len(args) > 1 && args[1] == "--dry-run" {
				mode = "dry-run"
			}

			t.Run(backend+"/"+mode, func(t *testing.T) {
				root := t.TempDir()
				issuegraphDir := filepath.Join(root, ".issuegraph")
				if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
					t.Fatalf("create .issuegraph: %v", err)
				}

				metadata := []byte(fmt.Sprintf("{\n  \"database\": \"legacy.db\",\n  \"backend\": %q,\n  \"project_id\": \"legacy-project\"\n}\n", backend))
				metadataPath := filepath.Join(issuegraphDir, configfile.ConfigFileName)
				if err := os.WriteFile(metadataPath, metadata, 0o600); err != nil {
					t.Fatalf("write metadata.json: %v", err)
				}

				cmd := exec.Command(issuegraph, args...)
				cmd.Dir = root
				cmd.Env = bootstrapBackendGuardEnv(root, issuegraphDir)
				out, err := cmd.CombinedOutput()
				if err == nil {
					t.Errorf("issuegraph %s unexpectedly succeeded for removed backend %q:\n%s", strings.Join(args, " "), backend, out)
				}

				message := strings.ToLower(string(out))
				for _, want := range removedBackendWantSubstrings(backend) {
					if !strings.Contains(message, want) {
						t.Errorf("issuegraph %s error for %q missing %q:\n%s", strings.Join(args, " "), backend, want, message)
					}
				}

				after, readErr := os.ReadFile(metadataPath)
				if readErr != nil {
					t.Fatalf("read metadata.json after rejected bootstrap: %v", readErr)
				}
				if !bytes.Equal(after, metadata) {
					t.Errorf("issuegraph %s rewrote metadata.json for removed backend %q:\nbefore:\n%s\nafter:\n%s", strings.Join(args, " "), backend, metadata, after)
				}
				assertNoBootstrapStorageArtifacts(t, issuegraphDir)
			})
		}
	}
}

func TestBootstrapRejectsUnknownBackendBeforeWorkspaceWrites(t *testing.T) {
	issuegraph := buildIssueGraphForInitTests(t)

	for _, args := range [][]string{{"bootstrap", "--dry-run"}, {"bootstrap", "--yes"}} {
		mode := "execute"
		if args[1] == "--dry-run" {
			mode = "dry-run"
		}

		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			issuegraphDir := filepath.Join(root, ".issuegraph")
			if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
				t.Fatalf("create .issuegraph: %v", err)
			}

			metadata := []byte("{\n  \"backend\": \"mystery\",\n  \"project_id\": \"legacy-project\"\n}\n")
			metadataPath := filepath.Join(issuegraphDir, configfile.ConfigFileName)
			if err := os.WriteFile(metadataPath, metadata, 0o600); err != nil {
				t.Fatalf("write metadata.json: %v", err)
			}

			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = root
			cmd.Env = bootstrapBackendGuardEnv(root, issuegraphDir)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("issuegraph %s unexpectedly accepted an unknown backend:\n%s", strings.Join(args, " "), out)
			}
			message := string(out)
			for _, want := range []string{"not recognized", "no storage database was opened or modified", "dolt"} {
				if !strings.Contains(message, want) {
					t.Errorf("issuegraph %s error missing %q:\n%s", strings.Join(args, " "), want, message)
				}
			}

			after, readErr := os.ReadFile(metadataPath)
			if readErr != nil {
				t.Fatalf("read metadata.json after rejected bootstrap: %v", readErr)
			}
			if !bytes.Equal(after, metadata) {
				t.Errorf("issuegraph %s rewrote unknown-backend metadata:\nbefore:\n%s\nafter:\n%s", strings.Join(args, " "), metadata, after)
			}
			assertNoBootstrapStorageArtifacts(t, issuegraphDir)
		})
	}
}

func TestBootstrapRejectsCorruptMetadataBeforeWorkspaceWrites(t *testing.T) {
	issuegraph := buildIssueGraphForInitTests(t)

	for _, args := range [][]string{{"bootstrap", "--dry-run"}, {"bootstrap", "--yes"}} {
		mode := "execute"
		if args[1] == "--dry-run" {
			mode = "dry-run"
		}

		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			issuegraphDir := filepath.Join(root, ".issuegraph")
			if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
				t.Fatalf("create .issuegraph: %v", err)
			}

			metadata := []byte("{\n")
			metadataPath := filepath.Join(issuegraphDir, configfile.ConfigFileName)
			if err := os.WriteFile(metadataPath, metadata, 0o600); err != nil {
				t.Fatalf("write corrupt metadata.json: %v", err)
			}

			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = root
			cmd.Env = bootstrapBackendGuardEnv(root, issuegraphDir)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("issuegraph %s unexpectedly ignored corrupt metadata:\n%s", strings.Join(args, " "), out)
			}
			message := strings.ToLower(string(out))
			for _, want := range []string{"metadata.json", "no storage database was opened or modified"} {
				if !strings.Contains(message, want) {
					t.Errorf("issuegraph %s error missing %q:\n%s", strings.Join(args, " "), want, out)
				}
			}

			after, readErr := os.ReadFile(metadataPath)
			if readErr != nil {
				t.Fatalf("read corrupt metadata after rejected bootstrap: %v", readErr)
			}
			if !bytes.Equal(after, metadata) {
				t.Errorf("issuegraph %s rewrote corrupt metadata:\nbefore: %q\nafter:  %q", strings.Join(args, " "), metadata, after)
			}
			assertNoBootstrapStorageArtifacts(t, issuegraphDir)
		})
	}
}

func TestBootstrapDoesNotConvertExistingSQLiteWorkspace(t *testing.T) {
	issuegraph := buildIssueGraphForInitTests(t)

	for _, args := range [][]string{{"bootstrap", "--dry-run"}, {"bootstrap", "--yes"}} {
		args := args
		mode := "execute"
		if len(args) > 1 && args[1] == "--dry-run" {
			mode = "dry-run"
		}

		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			for _, gitArgs := range [][]string{
				{"init", "-q"},
				{"config", "user.email", "test@test.com"},
				{"config", "user.name", "Test"},
				{"config", "core.hooksPath", ".git/hooks"},
			} {
				runGitForBootstrapTest(t, root, gitArgs...)
			}

			// Hand-seed a workspace created by an earlier build's SQLite backend:
			// metadata selecting the removed backend plus its database file. The
			// guard must reject it without reading or rewriting either.
			issuegraphDir := filepath.Join(root, ".issuegraph")
			if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
				t.Fatalf("create .issuegraph: %v", err)
			}
			metadataPath := filepath.Join(issuegraphDir, configfile.ConfigFileName)
			metadataBefore := []byte("{\n  \"database\": \"issuegraph.db\",\n  \"backend\": \"sqlite\",\n  \"sqlite_path\": \"issuegraph.db\",\n  \"project_id\": \"sqlite-guard\"\n}\n")
			if err := os.WriteFile(metadataPath, metadataBefore, 0o600); err != nil {
				t.Fatalf("write metadata.json: %v", err)
			}
			databasePath := filepath.Join(issuegraphDir, "issuegraph.db")
			databaseContent := []byte("sqlite-guard database placeholder")
			if err := os.WriteFile(databasePath, databaseContent, 0o600); err != nil {
				t.Fatalf("write issuegraph.db: %v", err)
			}
			databaseBefore := sha256.Sum256(databaseContent)
			localVersionBefore, hadLocalVersion := readOptionalBootstrapGuardFile(t, filepath.Join(issuegraphDir, ".local_version"))

			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = root
			cmd.Env = bootstrapBackendGuardEnv(root, issuegraphDir)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("issuegraph %s unexpectedly accepted an existing SQLite workspace:\n%s", strings.Join(args, " "), out)
			}

			message := strings.ToLower(string(out))
			for _, want := range []string{
				"historical sqlite workspace",
				"explicit migration is required",
				"preserve .issuegraph unchanged",
				"cross-era-upgrades",
			} {
				if !strings.Contains(message, want) {
					t.Errorf("issuegraph %s error missing %q:\n%s", strings.Join(args, " "), want, out)
				}
			}

			metadataAfter := readBootstrapGuardFile(t, metadataPath)
			if !bytes.Equal(metadataAfter, metadataBefore) {
				t.Errorf("issuegraph %s rewrote SQLite metadata.json:\nbefore:\n%s\nafter:\n%s", strings.Join(args, " "), metadataBefore, metadataAfter)
			}
			databaseAfter := sha256.Sum256(readBootstrapGuardFile(t, databasePath))
			if databaseAfter != databaseBefore {
				t.Errorf("issuegraph %s modified the existing SQLite database", strings.Join(args, " "))
			}
			localVersionAfter, hasLocalVersion := readOptionalBootstrapGuardFile(t, filepath.Join(issuegraphDir, ".local_version"))
			if hasLocalVersion != hadLocalVersion || !bytes.Equal(localVersionAfter, localVersionBefore) {
				t.Errorf("issuegraph %s created or modified SQLite workspace version tracking", strings.Join(args, " "))
			}
			for _, name := range []string{"embeddeddolt", "dolt"} {
				path := filepath.Join(issuegraphDir, name)
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Errorf("issuegraph %s converted SQLite by creating Dolt state at %s (stat error: %v)", strings.Join(args, " "), path, statErr)
				}
			}
		})
	}
}

func bootstrapBackendGuardEnv(home, issuegraphDir string) []string {
	env := make([]string, 0, len(os.Environ())+7)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "ISSUEGRAPH_") || strings.HasPrefix(entry, "ISSUEGRAPH_") || strings.HasPrefix(entry, "HOME=") {
			continue
		}
		env = append(env, entry)
	}
	return append(env,
		"HOME="+home,
		"ISSUEGRAPH_DIR="+issuegraphDir,
		"ISSUEGRAPH_DOLT_AUTO_START=0",
		"ISSUEGRAPH_NO_DAEMON=1",
		"ISSUEGRAPH_TEST_IGNORE_REPO_CONFIG=1",
		"ISSUEGRAPH_DISABLE_METRICS=1",
		"ISSUEGRAPH_DISABLE_EVENT_FLUSH=1",
		"ISSUEGRAPH_NON_INTERACTIVE=1",
	)
}

func assertNoBootstrapStorageArtifacts(t *testing.T, issuegraphDir string) {
	t.Helper()
	for _, name := range []string{"embeddeddolt", "dolt", "issuegraph.db", ".local_version"} {
		path := filepath.Join(issuegraphDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("rejected bootstrap created local state at %s (stat error: %v)", path, err)
		}
	}
}

func readBootstrapGuardFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func readOptionalBootstrapGuardFile(t *testing.T, path string) ([]byte, bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data, true
}
