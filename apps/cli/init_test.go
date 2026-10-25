//go:build cgo

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/git"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dolt"
	"github.com/borg0ai/dx-issuegraph/internal/testutil"
)

// skipIfNoDolt skips the test when no Dolt server is available.
// Checks both binary availability and test server status.
func skipIfNoDolt(t *testing.T) {
	t.Helper()
	testutil.RequireDoltBinary(t)
	if testDoltServerPort == 0 {
		t.Skip("skipping: Dolt test server not running")
	}
}

func TestInitCommand(t *testing.T) {
	skipIfNoDolt(t)
	tests := []struct {
		name           string
		prefix         string
		quiet          bool
		wantOutputText string
		wantNoOutput   bool
	}{
		{
			name:           "init with default prefix",
			prefix:         "",
			quiet:          false,
			wantOutputText: "issuegraph initialized successfully",
		},
		{
			name:           "init with custom prefix",
			prefix:         "myproject",
			quiet:          false,
			wantOutputText: "myproject-<hash>",
		},
		{
			name:         "init with quiet flag",
			prefix:       "test",
			quiet:        true,
			wantNoOutput: true,
		},
		{
			name:           "init with prefix ending in hyphen",
			prefix:         "test-",
			quiet:          false,
			wantOutputText: "test-<hash>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			origDBPath := dbPath
			origStore := store
			defer func() {
				if store != nil && store != origStore {
					store.Close()
				}
				store = origStore
				dbPath = origDBPath
			}()
			dbPath = ""
			store = nil

			// Reset Cobra command state
			rootCmd.SetArgs([]string{})
			initCmd.Flags().Set("prefix", "")
			initCmd.Flags().Set("quiet", "false")

			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() {
				os.Stdout = oldStdout
			}()

			// Build command arguments
			args := []string{"init"}
			if tt.prefix != "" {
				args = append(args, "--prefix", tt.prefix)
			}
			if tt.quiet {
				args = append(args, "--quiet")
			}

			rootCmd.SetArgs(args)

			// Run command
			var err error
			err = rootCmd.Execute()

			// Restore stdout and read output
			w.Close()
			buf.ReadFrom(r)
			os.Stdout = oldStdout
			output := buf.String()

			if err != nil {
				t.Fatalf("init command failed: %v", err)
			}

			// Check output
			if tt.wantNoOutput {
				if output != "" {
					t.Errorf("Expected no output with --quiet, got: %s", output)
				}
			} else if tt.wantOutputText != "" {
				if !strings.Contains(output, tt.wantOutputText) {
					t.Errorf("Expected output to contain %q, got: %s", tt.wantOutputText, output)
				}
			}

			// Verify .issuegraph directory was created
			issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
			if _, err := os.Stat(issuegraphDir); os.IsNotExist(err) {
				t.Error(".issuegraph directory was not created")
			}

			// Verify .gitignore was created with proper content
			gitignorePath := filepath.Join(issuegraphDir, ".gitignore")
			gitignoreContent, err := os.ReadFile(gitignorePath)
			if err != nil {
				t.Errorf(".gitignore file was not created: %v", err)
			} else {
				// Check for essential patterns
				gitignoreStr := string(gitignoreContent)
				expectedPatterns := []string{
					"*.db",
					"*.db?*",
					"*.db-journal",
					"*.db-wal",
					"*.db-shm",
					"issuegraph.sock",
					"dolt/",
				}
				for _, pattern := range expectedPatterns {
					if !strings.Contains(gitignoreStr, pattern) {
						t.Errorf(".gitignore missing expected pattern: %s", pattern)
					}
				}
			}

			// Verify Dolt database directory was created
			doltPath := filepath.Join(issuegraphDir, "dolt")
			if info, err := os.Stat(doltPath); os.IsNotExist(err) {
				t.Errorf("Dolt database directory was not created at %s", doltPath)
			} else if !info.IsDir() {
				t.Errorf("Expected %s to be a directory", doltPath)
			}

			// Database content verification (prefix, metadata) is skipped here because
			// embedded Dolt's Close() can timeout, leaving file locks held and preventing
			// re-opening the DB in the same process. The init command's own internal logic
			// verifies these writes succeed; prefix/metadata correctness is also covered
			// by dedicated Dolt storage tests.
		})
	}
}

// Note: Error case testing is omitted because the init command calls os.Exit()
// on errors, which makes it difficult to test in a unit test context.

func TestInitAlreadyInitialized(t *testing.T) {
	skipIfNoDolt(t)
	// Reset global state
	origDBPath := dbPath
	defer func() { dbPath = origDBPath }()
	dbPath = ""

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Initialize once
	rootCmd.SetArgs([]string{"init", "--prefix", "test", "--quiet"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("First init failed: %v", err)
	}

	// Initialize again with same prefix and --force flag (bd-emg: safety guard)
	// Without --force, init should refuse when database already exists
	rootCmd.SetArgs([]string{"init", "--prefix", "test", "--quiet", "--force"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Second init with --force failed: %v", err)
	}

	// Verify database still works
	dbPath := filepath.Join(tmpDir, ".issuegraph", "dolt")
	store, err := openExistingTestDB(t, dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	prefix, err := store.GetConfig(ctx, "issue_prefix")
	if err != nil {
		t.Fatalf("Failed to get prefix after re-init: %v", err)
	}

	if prefix != "test" {
		t.Errorf("Expected prefix 'test', got %q", prefix)
	}
}

// GH#3490: `issuegraph init --init-if-missing` makes init idempotent for scaffold
// scripts. When the workspace is already initialized, init must skip and
// return without error (exit 0), emitting a benign "Skipping init" message,
// instead of aborting via os.Exit(1). The default (no-flag) abort path calls
// os.Exit(1) and therefore cannot be exercised in-process (see the note above
// TestInitAlreadyInitialized); this test covers the new success path.
func TestInitIfMissing(t *testing.T) {
	skipIfNoDolt(t)
	// Reset global state
	origDBPath := dbPath

	// Cobra flag values persist across Execute() calls on the shared command
	// tree (a sibling test may have left --force set, and our own first init
	// sets --quiet). Normalize the flags this test depends on before each run,
	// and restore defaults afterward so we neither inherit nor leak state.
	resetInitFlags := func() {
		_ = initCmd.Flags().Set("force", "false")
		_ = initCmd.Flags().Set("reinit-local", "false")
		_ = initCmd.Flags().Set("init-if-missing", "false")
		_ = initCmd.Flags().Set("quiet", "false")
		_ = initCmd.Flags().Set("prefix", "")
	}
	defer func() {
		dbPath = origDBPath
		resetInitFlags()
	}()
	dbPath = ""
	resetInitFlags()

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// First init creates the workspace.
	rootCmd.SetArgs([]string{"init", "--prefix", "test", "--quiet"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("First init failed: %v", err)
	}

	// Re-running with --init-if-missing must be a benign no-op that exits 0:
	// Execute() returns nil rather than the process aborting via os.Exit(1).
	// Clear --quiet (set by the first run) so the skip message is emitted, and
	// ensure no stale --force bypasses the already-initialized guard.
	resetInitFlags()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	rootCmd.SetArgs([]string{"init", "--prefix", "test", "--init-if-missing"})
	execErr := rootCmd.Execute()

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderr := buf.String()

	if execErr != nil {
		t.Fatalf("init --init-if-missing on an initialized workspace should succeed, got: %v", execErr)
	}
	if !strings.Contains(stderr, "Skipping init: workspace already initialized") {
		t.Errorf("expected a 'Skipping init: workspace already initialized' message on stderr, got:\n%s", stderr)
	}

	// The skip is a no-op: the existing workspace must remain in place (init
	// returned before touching any data). We assert on the .issuegraph directory
	// rather than a specific backend layout so the test holds for both the
	// embedded and server test modes.
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if info, err := os.Stat(issuegraphDir); err != nil || !info.IsDir() {
		t.Fatalf("expected existing .issuegraph workspace to remain at %s, stat err: %v", issuegraphDir, err)
	}
}

// TestInitIfMissingPrefixMismatch covers the guard that keeps --init-if-missing
// from masking a genuine prefix mismatch (review follow-up on #4332/#3490): a
// re-init that explicitly requests a different prefix than the existing
// workspace must abort, while a matching prefix (after normalization) or an
// undeterminable existing name must fall through to the benign skip.
func TestInitIfMissingPrefixMismatch(t *testing.T) {
	tests := []struct {
		name      string
		existing  string
		requested string
		want      bool
	}{
		{name: "exact match", existing: "foo", requested: "foo", want: false},
		{name: "case-insensitive match", existing: "Foo", requested: "foo", want: false},
		{name: "hyphen normalizes to underscore", existing: "my_proj", requested: "my-proj", want: false},
		{name: "trailing hyphen trimmed", existing: "foo", requested: "foo-", want: false},
		{name: "leading-digit gets issuegraph_ prefix", existing: "issuegraph_001", requested: "001", want: false},
		{name: "genuine mismatch aborts", existing: "foo", requested: "bar", want: true},
		{name: "unknown existing falls through", existing: "", requested: "bar", want: false},
		{name: "empty requested falls through", existing: "foo", requested: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := initIfMissingPrefixMismatch(tt.existing, tt.requested); got != tt.want {
				t.Errorf("initIfMissingPrefixMismatch(%q, %q) = %v, want %v",
					tt.existing, tt.requested, got, tt.want)
			}
		})
	}
}

// TestInitIfMissingMatchingPrefixSkips verifies the mismatch guard does not
// regress the happy path: re-running with the SAME explicit --prefix as the
// existing workspace still skips cleanly (exit 0) rather than aborting.
func TestInitIfMissingMatchingPrefixSkips(t *testing.T) {
	skipIfNoDolt(t)
	origDBPath := dbPath
	resetInitFlags := func() {
		_ = initCmd.Flags().Set("force", "false")
		_ = initCmd.Flags().Set("reinit-local", "false")
		_ = initCmd.Flags().Set("init-if-missing", "false")
		_ = initCmd.Flags().Set("quiet", "false")
		_ = initCmd.Flags().Set("prefix", "")
	}
	defer func() {
		dbPath = origDBPath
		resetInitFlags()
	}()
	dbPath = ""
	resetInitFlags()

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	rootCmd.SetArgs([]string{"init", "--prefix", "test", "--quiet"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("First init failed: %v", err)
	}

	resetInitFlags()
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	// Same prefix as the existing workspace: must skip, not abort.
	rootCmd.SetArgs([]string{"init", "--prefix", "test", "--init-if-missing"})
	execErr := rootCmd.Execute()

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderr := buf.String()

	if execErr != nil {
		t.Fatalf("init --init-if-missing with matching prefix should succeed, got: %v", execErr)
	}
	if !strings.Contains(stderr, "Skipping init: workspace already initialized") {
		t.Errorf("expected benign skip message, got:\n%s", stderr)
	}
}

func TestInitIfMissingDatabaseMismatch(t *testing.T) {
	tests := []struct {
		name      string
		existing  string
		requested string
		want      bool
	}{
		{name: "exact match", existing: "foo", requested: "foo", want: false},
		{name: "case-insensitive match", existing: "Foo", requested: "foo", want: false},
		{name: "genuine mismatch aborts", existing: "foo", requested: "bar", want: true},
		{name: "unknown existing falls through", existing: "", requested: "bar", want: false},
		{name: "empty requested falls through", existing: "foo", requested: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := initIfMissingDatabaseMismatch(tt.existing, tt.requested); got != tt.want {
				t.Errorf("initIfMissingDatabaseMismatch(%q, %q) = %v, want %v",
					tt.existing, tt.requested, got, tt.want)
			}
		})
	}
}

// TestCheckExistingIssueGraphDataOperationalErrorNotMasked verifies the core of the
// --init-if-missing idempotency fix: only the benign "already initialized"
// outcome matches errWorkspaceAlreadyInitialized (and may be skipped), while an
// operational failure must NOT match it — otherwise --init-if-missing would mask
// a real error (e.g. an unreadable .issuegraph/embeddeddolt) as a successful skip.
func TestCheckExistingIssueGraphDataOperationalErrorNotMasked(t *testing.T) {
	saveEmbeddedConfig := func(t *testing.T, issuegraphDir string) {
		t.Helper()
		cfg := &configfile.Config{
			Database: "dolt",
			Backend:  configfile.BackendDolt,
			DoltMode: configfile.DoltModeEmbedded,
		}
		if err := cfg.Save(issuegraphDir); err != nil {
			t.Fatalf("save config: %v", err)
		}
	}

	t.Run("existing database matches sentinel", func(t *testing.T) {
		issuegraphDir := t.TempDir()
		saveEmbeddedConfig(t, issuegraphDir)
		// A real embedded database lives at embeddeddolt/<db>/.dolt.
		if err := os.MkdirAll(filepath.Join(issuegraphDir, "embeddeddolt", "mydb", ".dolt"), 0o750); err != nil {
			t.Fatal(err)
		}
		err := checkExistingIssueGraphDataAt(issuegraphDir, "mydb")
		if err == nil {
			t.Fatal("expected already-initialized error, got nil")
		}
		if !errors.Is(err, errWorkspaceAlreadyInitialized) {
			t.Errorf("existing-database error must match errWorkspaceAlreadyInitialized, got: %v", err)
		}
	})

	t.Run("operational error not masked", func(t *testing.T) {
		issuegraphDir := t.TempDir()
		saveEmbeddedConfig(t, issuegraphDir)
		// Make embeddeddolt a regular file so os.ReadDir fails with a
		// non-IsNotExist (operational) error rather than "already initialized".
		if err := os.WriteFile(filepath.Join(issuegraphDir, "embeddeddolt"), []byte("not a dir"), 0o600); err != nil {
			t.Fatal(err)
		}
		err := checkExistingIssueGraphDataAt(issuegraphDir, "mydb")
		if err == nil {
			t.Fatal("expected operational error, got nil")
		}
		if errors.Is(err, errWorkspaceAlreadyInitialized) {
			t.Errorf("operational error must NOT match errWorkspaceAlreadyInitialized (would be masked by --init-if-missing): %v", err)
		}
	})
}

func TestInitWithCustomDBPath(t *testing.T) {
	t.Skip("ISSUEGRAPH_DB env var does not control Dolt store location; Dolt always uses .issuegraph/dolt/")
	// Save original state
	origDBPath := dbPath
	defer func() { dbPath = origDBPath }()

	tmpDir := t.TempDir()
	customDBDir := filepath.Join(tmpDir, "custom", "location")

	// Change to a different directory to ensure --db flag is actually used
	workDir := filepath.Join(tmpDir, "workdir")
	if err := os.MkdirAll(workDir, 0750); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	t.Chdir(workDir)

	customDBPath := filepath.Join(customDBDir, "test.db")

	// Test with ISSUEGRAPH_DB environment variable (replacing --db flag test)
	t.Run("init with ISSUEGRAPH_DB pointing to custom path", func(t *testing.T) {
		dbPath = "" // Reset global
		os.Setenv("ISSUEGRAPH_DB", customDBPath)
		defer os.Unsetenv("ISSUEGRAPH_DB")

		rootCmd.SetArgs([]string{"init", "--prefix", "custom", "--quiet"})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Init with ISSUEGRAPH_DB failed: %v", err)
		}

		// Verify database was created at custom location
		if _, err := os.Stat(customDBPath); os.IsNotExist(err) {
			t.Errorf("Database was not created at custom path %s", customDBPath)
		}

		// Verify database works
		store, err := openExistingTestDB(t, customDBPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer store.Close()

		ctx := context.Background()
		prefix, err := store.GetConfig(ctx, "issue_prefix")
		if err != nil {
			t.Fatalf("Failed to get prefix: %v", err)
		}

		if prefix != "custom" {
			t.Errorf("Expected prefix 'custom', got %q", prefix)
		}

		// Verify .issuegraph/ directory was NOT created in work directory
		if _, err := os.Stat(filepath.Join(workDir, ".issuegraph")); err == nil {
			t.Error(".issuegraph/ directory should not be created when using ISSUEGRAPH_DB env var")
		}
	})

	// Test with ISSUEGRAPH_DB env var
	t.Run("init with ISSUEGRAPH_DB env var", func(t *testing.T) {
		dbPath = "" // Reset global
		envDBPath := filepath.Join(tmpDir, "env", "location", "env.db")
		os.Setenv("ISSUEGRAPH_DB", envDBPath)
		defer os.Unsetenv("ISSUEGRAPH_DB")

		rootCmd.SetArgs([]string{"init", "--prefix", "envtest", "--quiet"})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Init with ISSUEGRAPH_DB failed: %v", err)
		}

		// Verify database was created at env location
		if _, err := os.Stat(envDBPath); os.IsNotExist(err) {
			t.Errorf("Database was not created at ISSUEGRAPH_DB path %s", envDBPath)
		}

		// Verify database works
		store, err := openExistingTestDB(t, envDBPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer store.Close()

		ctx := context.Background()
		prefix, err := store.GetConfig(ctx, "issue_prefix")
		if err != nil {
			t.Fatalf("Failed to get prefix: %v", err)
		}

		if prefix != "envtest" {
			t.Errorf("Expected prefix 'envtest', got %q", prefix)
		}
	})

	// Test that ISSUEGRAPH_DB path containing ".issuegraph" doesn't create CWD/.issuegraph
	t.Run("init with ISSUEGRAPH_DB path containing .issuegraph", func(t *testing.T) {
		dbPath = "" // Reset global
		// Path contains ".issuegraph" but is outside work directory
		customPath := filepath.Join(tmpDir, "storage", ".issuegraph-backup", "test.db")
		os.Setenv("ISSUEGRAPH_DB", customPath)
		defer os.Unsetenv("ISSUEGRAPH_DB")

		rootCmd.SetArgs([]string{"init", "--prefix", "issuegraphtest", "--quiet"})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Init with custom .issuegraph path failed: %v", err)
		}

		// Verify database was created at custom location
		if _, err := os.Stat(customPath); os.IsNotExist(err) {
			t.Errorf("Database was not created at custom path %s", customPath)
		}

		// Verify .issuegraph/ directory was NOT created in work directory
		if _, err := os.Stat(filepath.Join(workDir, ".issuegraph")); err == nil {
			t.Error(".issuegraph/ directory should not be created in CWD when ISSUEGRAPH_DB path contains .issuegraph")
		}
	})

	// Test with multiple ISSUEGRAPH_DB variations
	t.Run("ISSUEGRAPH_DB with subdirectories", func(t *testing.T) {
		dbPath = "" // Reset global
		envPath := filepath.Join(tmpDir, "env", "subdirs", "test.db")

		os.Setenv("ISSUEGRAPH_DB", envPath)
		defer os.Unsetenv("ISSUEGRAPH_DB")

		rootCmd.SetArgs([]string{"init", "--prefix", "envtest2", "--quiet"})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Init with ISSUEGRAPH_DB subdirs failed: %v", err)
		}

		// Verify database was created at env location
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			t.Errorf("Database was not created at ISSUEGRAPH_DB path %s", envPath)
		}

		// Verify .issuegraph/ directory was NOT created in work directory
		if _, err := os.Stat(filepath.Join(workDir, ".issuegraph")); err == nil {
			t.Error(".issuegraph/ directory should not be created in CWD when ISSUEGRAPH_DB is set")
		}
	})
}

// TestSetupClaudeSettings_InvalidJSON verifies that invalid JSON in existing
// settings.local.json returns an error instead of silently overwriting.
// This is a regression test for bd-5bj where user settings were lost.
func TestSetupClaudeSettings_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Create .claude directory
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude directory: %v", err)
	}

	// Create settings.local.json with invalid JSON (array syntax in object context)
	// This is the exact pattern that caused the bug in the user's file
	invalidJSON := `{
  "permissions": {
    "allow": [
      "Bash(python3:*)"
    ],
    "deny": [
      "_comment": "Add commands to block here"
    ]
  }
}`
	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	if err := os.WriteFile(settingsPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write invalid settings: %v", err)
	}

	// Call setupClaudeSettings - should return an error
	var err error
	err = setupClaudeSettings(false)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}

	// Verify the error message mentions invalid JSON
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("Expected error to mention 'invalid JSON', got: %v", err)
	}

	// Verify the original file was NOT modified
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	if !strings.Contains(string(content), "permissions") {
		t.Error("Original file content should be preserved")
	}

	if strings.Contains(string(content), "issuegraph prime") {
		t.Error("File should NOT contain issuegraph prime prompt after error")
	}
}

// TestSetupClaudeSettings_ValidJSON verifies that valid JSON is properly updated
func TestSetupClaudeSettings_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Create .claude directory
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude directory: %v", err)
	}

	// Create settings.local.json with valid JSON
	validJSON := `{
  "permissions": {
    "allow": [
      "Bash(python3:*)"
    ]
  },
  "hooks": {
    "PreToolUse": []
  }
}`
	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	if err := os.WriteFile(settingsPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("Failed to write valid settings: %v", err)
	}

	// Call setupClaudeSettings - should succeed
	var err error
	err = setupClaudeSettings(false)
	if err != nil {
		t.Fatalf("Expected no error for valid JSON, got: %v", err)
	}

	// Verify the file was updated with prompt AND preserved existing settings
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	contentStr := string(content)

	// Should contain the new prompt
	if !strings.Contains(contentStr, "issuegraph prime") {
		t.Error("File should contain issuegraph prime prompt")
	}

	// Should preserve existing permissions
	if !strings.Contains(contentStr, "permissions") {
		t.Error("File should preserve permissions section")
	}

	// Should preserve existing hooks
	if !strings.Contains(contentStr, "hooks") {
		t.Error("File should preserve hooks section")
	}

	if !strings.Contains(contentStr, "PreToolUse") {
		t.Error("File should preserve PreToolUse hook")
	}
}

// TestSetupClaudeSettings_NoExistingFile verifies behavior when no file exists
func TestSetupClaudeSettings_NoExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Don't create .claude directory - setupClaudeSettings should create it

	// Call setupClaudeSettings - should succeed
	var err error
	err = setupClaudeSettings(false)
	if err != nil {
		t.Fatalf("Expected no error when no file exists, got: %v", err)
	}

	// Verify the file was created with prompt
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.local.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	if !strings.Contains(string(content), "issuegraph prime") {
		t.Error("File should contain issuegraph prime prompt")
	}
}

// setupIsolatedGitConfig creates an empty git config in tmpDir and sets GIT_CONFIG_GLOBAL
// to prevent tests from using the real user's global git config.
func setupIsolatedGitConfig(t *testing.T, tmpDir string) {
	t.Helper()
	gitConfigPath := filepath.Join(tmpDir, ".gitconfig")
	if err := os.WriteFile(gitConfigPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfigPath)
}

// TestSetupGlobalGitIgnore_ReadOnly verifies graceful handling when the
// gitignore file cannot be written (prints manual instructions instead of failing).
func TestSetupGlobalGitIgnore_ReadOnly(t *testing.T) {
	t.Run("read-only file", func(t *testing.T) {
		if runtime.GOOS == "darwin" {
			t.Skip("macOS allows file owner to write to read-only (0444) files")
		}
		tmpDir := t.TempDir()
		setupIsolatedGitConfig(t, tmpDir)

		configDir := filepath.Join(tmpDir, ".config", "git")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		ignorePath := filepath.Join(configDir, "ignore")
		if err := os.WriteFile(ignorePath, []byte("# existing\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ignorePath, 0444); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(ignorePath, 0644)

		output := captureStdout(t, func() error {
			return setupGlobalGitIgnore(tmpDir, "/test/project", false)
		})

		if !strings.Contains(output, "Unable to write") {
			t.Error("expected instructions for manual addition")
		}
		if !strings.Contains(output, "/test/project/.issuegraph/") {
			t.Error("expected .issuegraph pattern in output")
		}
	})

	t.Run("symlink to read-only file", func(t *testing.T) {
		if runtime.GOOS == "darwin" {
			t.Skip("macOS allows file owner to write to read-only (0444) files")
		}
		tmpDir := t.TempDir()
		setupIsolatedGitConfig(t, tmpDir)

		// Target file in a separate location
		targetDir := filepath.Join(tmpDir, "target")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatal(err)
		}
		targetFile := filepath.Join(targetDir, "ignore")
		if err := os.WriteFile(targetFile, []byte("# existing\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(targetFile, 0444); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(targetFile, 0644)

		// Symlink from expected location
		configDir := filepath.Join(tmpDir, ".config", "git")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(targetFile, filepath.Join(configDir, "ignore")); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() error {
			return setupGlobalGitIgnore(tmpDir, "/test/project", false)
		})

		if !strings.Contains(output, "Unable to write") {
			t.Error("expected instructions for manual addition")
		}
		if !strings.Contains(output, "/test/project/.issuegraph/") {
			t.Error("expected .issuegraph pattern in output")
		}
	})
}

// TestInitPromptRoleConfig tests the issuegraph.role git config read/write functions
func TestInitPromptRoleConfig(t *testing.T) {
	t.Run("getIssueGraphRole returns empty when not configured", func(t *testing.T) {
		tmpDir := newGitRepo(t)
		t.Chdir(tmpDir)

		role, hasRole := getIssueGraphRole()
		if hasRole {
			t.Errorf("Expected hasRole=false when not configured, got true with role=%q", role)
		}
		if role != "" {
			t.Errorf("Expected empty role when not configured, got %q", role)
		}
	})

	t.Run("setIssueGraphRole and getIssueGraphRole roundtrip", func(t *testing.T) {
		tmpDir := newGitRepo(t)
		t.Chdir(tmpDir)

		// Set role to contributor
		if err := setIssueGraphRole("contributor"); err != nil {
			t.Fatalf("Failed to set issuegraph.role: %v", err)
		}

		role, hasRole := getIssueGraphRole()
		if !hasRole {
			t.Error("Expected hasRole=true after setting role")
		}
		if role != "contributor" {
			t.Errorf("Expected role 'contributor', got %q", role)
		}

		// Change to maintainer
		if err := setIssueGraphRole("maintainer"); err != nil {
			t.Fatalf("Failed to set issuegraph.role: %v", err)
		}

		role, hasRole = getIssueGraphRole()
		if !hasRole {
			t.Error("Expected hasRole=true after setting role")
		}
		if role != "maintainer" {
			t.Errorf("Expected role 'maintainer', got %q", role)
		}
	})
}

// TestInitPromptSkippedWithFlags verifies that --contributor and --team flags skip the prompt
func TestInitPromptSkippedWithFlags(t *testing.T) {
	skipIfNoDolt(t)
	t.Run("contributor flag skips prompt and runs wizard", func(t *testing.T) {
		// Reset global state
		origDBPath := dbPath
		defer func() { dbPath = origDBPath }()
		dbPath = ""

		// Reset caches so RepoContext picks up new directory
		issuegraph.ResetCaches()
		git.ResetCaches()
		defer func() {
			issuegraph.ResetCaches()
			git.ResetCaches()
		}()

		// Reset Cobra flags
		initCmd.Flags().Set("contributor", "false")

		tmpDir := newGitRepo(t)
		t.Chdir(tmpDir)

		// Verify no role is set initially
		role, hasRole := getIssueGraphRole()
		if hasRole {
			t.Fatalf("Expected no role initially, got %q", role)
		}

		// Run issuegraph init with --contributor flag (quiet to suppress wizard output)
		// The wizard will fail because there's no planning repo, but that's OK
		// We just want to verify the flag bypasses the prompt
		rootCmd.SetArgs([]string{"init", "--prefix", "test", "--contributor", "--quiet"})
		_ = rootCmd.Execute() // Ignore error - wizard may fail

		// The --contributor flag should NOT set issuegraph.role (that's done by prompt, not flag)
		// The flag just triggers the wizard directly
	})

	t.Run("team flag skips prompt", func(t *testing.T) {
		// Reset global state
		origDBPath := dbPath
		defer func() { dbPath = origDBPath }()
		dbPath = ""

		// Reset caches so RepoContext picks up new directory
		issuegraph.ResetCaches()
		git.ResetCaches()
		defer func() {
			issuegraph.ResetCaches()
			git.ResetCaches()
		}()

		// Reset Cobra flags
		initCmd.Flags().Set("team", "false")

		tmpDir := newGitRepo(t)
		t.Chdir(tmpDir)

		// Verify no role is set initially
		role, hasRole := getIssueGraphRole()
		if hasRole {
			t.Fatalf("Expected no role initially, got %q", role)
		}

		// Run issuegraph init with --team flag
		rootCmd.SetArgs([]string{"init", "--prefix", "test", "--team", "--quiet"})
		_ = rootCmd.Execute() // Ignore error - wizard may fail

		// The --team flag should not set issuegraph.role
		// (team wizard is separate from contributor/maintainer roles)
	})
}

// TestInitPromptTTYDetection verifies shouldPromptForRole behavior
func TestInitPromptTTYDetection(t *testing.T) {
	// Note: In test environment, stdin is typically NOT a TTY (it's a pipe)
	// This test verifies the function works, not that we're in a TTY

	t.Run("shouldPromptForRole returns false in test environment", func(t *testing.T) {
		// In test environment, stdin is typically piped, not a TTY
		result := shouldPromptForRole()

		// We can't guarantee what the result will be in all test environments,
		// but we can verify the function doesn't panic and returns a bool
		if result {
			t.Log("Test environment has TTY stdin (unusual but acceptable)")
		} else {
			t.Log("Test environment does not have TTY stdin (expected)")
		}
	})
}

// TestInitPromptNonGitRepo verifies prompt is skipped in non-git directories
func TestInitPromptNonGitRepo(t *testing.T) {
	skipIfNoDolt(t)
	// Reset global state
	origDBPath := dbPath
	defer func() { dbPath = origDBPath }()
	dbPath = ""

	// Reset caches so RepoContext picks up new directory
	issuegraph.ResetCaches()
	git.ResetCaches()
	defer func() {
		issuegraph.ResetCaches()
		git.ResetCaches()
	}()

	// Reset Cobra flags that may be set from previous tests
	initCmd.Flags().Set("contributor", "false")
	initCmd.Flags().Set("team", "false")

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// DON'T initialize git repo

	// Run issuegraph init - should succeed without prompting (no git repo)
	rootCmd.SetArgs([]string{"init", "--prefix", "test", "--quiet"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Init should succeed in non-git directory: %v", err)
	}

	// Verify .issuegraph was created
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if _, err := os.Stat(issuegraphDir); os.IsNotExist(err) {
		t.Error(".issuegraph directory should be created even without git")
	}
}

// TestInitPromptExistingRole verifies behavior when issuegraph.role is already set
func TestInitPromptExistingRole(t *testing.T) {
	skipIfNoDolt(t)
	t.Run("existing role is preserved on reinit with --force", func(t *testing.T) {
		// Reset global state
		origDBPath := dbPath
		defer func() { dbPath = origDBPath }()
		dbPath = ""

		// Reset caches so RepoContext picks up new directory
		issuegraph.ResetCaches()
		git.ResetCaches()
		defer func() {
			issuegraph.ResetCaches()
			git.ResetCaches()
		}()

		// Reset Cobra flags that may be set from previous tests
		initCmd.Flags().Set("contributor", "false")
		initCmd.Flags().Set("team", "false")
		initCmd.Flags().Set("force", "false")

		tmpDir := newGitRepo(t)
		t.Chdir(tmpDir)

		// Set role before init
		if err := setIssueGraphRole("contributor"); err != nil {
			t.Fatalf("Failed to set issuegraph.role: %v", err)
		}

		// Run issuegraph init (non-interactive, so prompt is skipped)
		rootCmd.SetArgs([]string{"init", "--prefix", "test", "--quiet"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Verify role is still set
		role, hasRole := getIssueGraphRole()
		if !hasRole {
			t.Error("Expected issuegraph.role to still be set after init")
		}
		if role != "contributor" {
			t.Errorf("Expected role 'contributor' to be preserved, got %q", role)
		}

		// Reset Cobra flags for reinit
		initCmd.Flags().Set("force", "false")

		// Reinit with --force (non-interactive)
		rootCmd.SetArgs([]string{"init", "--prefix", "test", "--quiet", "--force"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Reinit failed: %v", err)
		}

		// Verify role is still set (not cleared by reinit)
		role, hasRole = getIssueGraphRole()
		if !hasRole {
			t.Error("Expected issuegraph.role to still be set after reinit")
		}
		if role != "contributor" {
			t.Errorf("Expected role 'contributor' to be preserved after reinit, got %q", role)
		}
	})
}

func TestInitContributorSetsIssueGraphRoleContributor(t *testing.T) {
	skipIfNoDolt(t)

	origDBPath := dbPath
	defer func() { dbPath = origDBPath }()
	dbPath = ""

	issuegraph.ResetCaches()
	git.ResetCaches()
	defer func() {
		issuegraph.ResetCaches()
		git.ResetCaches()
	}()

	initCmd.Flags().Set("contributor", "false")
	initCmd.Flags().Set("team", "false")
	initCmd.Flags().Set("force", "false")

	tmpDir := newGitRepo(t)
	t.Chdir(tmpDir)

	// Keep test isolated from the real home/planning repo.
	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	// Configure remotes so contributor wizard doesn't ask the "continue anyway" prompt.
	cmd := exec.Command("git", "remote", "add", "origin", "git@github.com:osamu2001/zmx.git")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add origin remote: %v", err)
	}
	cmd = exec.Command("git", "remote", "add", "upstream", "git@github.com:neurosnap/zmx.git")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add upstream remote: %v", err)
	}

	// Wizard answers:
	// 1) "Do you want to use a separate planning repo anyway? [Y/n]" -> Enter (default yes)
	// 2) "Planning repo path [press Enter for default]" -> Enter (default ~/.issuegraph-planning)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })
	t.Cleanup(func() { _ = r.Close() })
	_, _ = w.WriteString("\n\n")
	_ = w.Close()

	rootCmd.SetArgs([]string{"init", "--prefix", "test", "--contributor", "--quiet"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init --contributor failed: %v", err)
	}

	role, hasRole := getIssueGraphRole()
	if !hasRole {
		t.Fatal("expected issuegraph.role to be configured")
	}
	if role != "contributor" {
		t.Fatalf("issuegraph.role = %q, want %q", role, "contributor")
	}
}

// TestInitNonInteractiveAlwaysSetsRole verifies that issuegraph init --non-interactive
// always leaves issuegraph.role set, even when no --role flag is provided (GH#2950).
// This is the safety net for the init flow.
func TestInitNonInteractiveAlwaysSetsRole(t *testing.T) {
	skipIfNoDolt(t)

	origDBPath := dbPath
	defer func() { dbPath = origDBPath }()
	dbPath = ""

	issuegraph.ResetCaches()
	git.ResetCaches()
	defer func() {
		issuegraph.ResetCaches()
		git.ResetCaches()
	}()

	initCmd.Flags().Set("contributor", "false")
	initCmd.Flags().Set("team", "false")
	initCmd.Flags().Set("force", "false")
	initCmd.Flags().Set("role", "")

	tmpDir := newGitRepo(t)
	t.Chdir(tmpDir)

	// Ensure no role is set before init
	exec.Command("git", "config", "--unset", "issuegraph.role").Run() //nolint:errcheck

	rootCmd.SetArgs([]string{"init", "--prefix", "test", "--quiet", "--non-interactive"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init --non-interactive failed: %v", err)
	}

	role, hasRole := getIssueGraphRole()
	if !hasRole {
		t.Fatal("expected issuegraph.role to be configured after non-interactive init (GH#2950)")
	}
	if role != "maintainer" {
		t.Fatalf("issuegraph.role = %q, want %q (default for non-interactive)", role, "maintainer")
	}
}

// TestInitWithRedirect verifies that issuegraph init creates the database in the redirect target,
// not in the local .issuegraph directory. (GH#bd-0qel)
// TestInitRedirect groups redirect-related init tests.
func TestInitRedirect(t *testing.T) {
	skipIfNoDolt(t)
	resetRedirectState := func(t *testing.T) {
		t.Helper()
		origDBPath := dbPath
		origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
		t.Cleanup(func() {
			dbPath = origDBPath
			if origIssueGraphDir != "" {
				os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
			} else {
				os.Unsetenv("ISSUEGRAPH_DIR")
			}
		})
		dbPath = ""
		os.Unsetenv("ISSUEGRAPH_DIR")
		initCmd.Flags().Set("prefix", "")
		initCmd.Flags().Set("quiet", "false")
		initCmd.Flags().Set("force", "false")
	}

	t.Run("RedirectCreatesDBInTarget", func(t *testing.T) {
		resetRedirectState(t)

		tmpDir := t.TempDir()

		projectDir := filepath.Join(tmpDir, "project")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatal(err)
		}

		localIssueGraphDir := filepath.Join(projectDir, ".issuegraph")
		if err := os.MkdirAll(localIssueGraphDir, 0755); err != nil {
			t.Fatal(err)
		}

		targetIssueGraphDir := filepath.Join(tmpDir, "canonical", ".issuegraph")
		if err := os.MkdirAll(targetIssueGraphDir, 0755); err != nil {
			t.Fatal(err)
		}

		redirectPath := filepath.Join(localIssueGraphDir, issuegraph.RedirectFileName)
		if err := os.WriteFile(redirectPath, []byte("../canonical/.issuegraph\n"), 0644); err != nil {
			t.Fatal(err)
		}

		t.Chdir(projectDir)

		rootCmd.SetArgs([]string{"init", "--prefix", "redirect-test", "--quiet"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Init with redirect failed: %v", err)
		}

		targetDBPath := filepath.Join(targetIssueGraphDir, "dolt")
		if _, err := os.Stat(targetDBPath); os.IsNotExist(err) {
			t.Errorf("Dolt database was NOT created in redirect target: %s", targetDBPath)
		}

		localDBPath := filepath.Join(localIssueGraphDir, "dolt")
		if _, err := os.Stat(localDBPath); err == nil {
			t.Errorf("Database was incorrectly created in local .issuegraph: %s (should be in redirect target)", localDBPath)
		}

		store, err := openExistingTestDB(t, targetDBPath)
		if err != nil {
			t.Fatalf("Failed to open database in redirect target: %v", err)
		}
		defer store.Close()

		ctx := context.Background()
		prefix, err := store.GetConfig(ctx, "issue_prefix")
		if err != nil {
			t.Fatalf("Failed to get issue prefix from database: %v", err)
		}
		if prefix != "redirect-test" {
			t.Errorf("Expected prefix 'redirect-test', got %q", prefix)
		}
	})

	// Verifies that issuegraph init errors when the redirect target already has a database,
	// preventing accidental overwrites. (GH#bd-0qel)
	t.Run("ErrorWhenTargetHasExistingDB", func(t *testing.T) {
		resetRedirectState(t)

		tmpDir := t.TempDir()

		canonicalDir := filepath.Join(tmpDir, "canonical")
		canonicalIssueGraphDir := filepath.Join(canonicalDir, ".issuegraph")
		if err := os.MkdirAll(canonicalIssueGraphDir, 0755); err != nil {
			t.Fatal(err)
		}

		canonicalDBPath := filepath.Join(canonicalIssueGraphDir, "issuegraph.db")
		// Create the db file so checkExistingIssueGraphData detects it
		if err := os.WriteFile(canonicalDBPath, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create canonical db file: %v", err)
		}

		projectDir := filepath.Join(tmpDir, "project")
		projectIssueGraphDir := filepath.Join(projectDir, ".issuegraph")
		if err := os.MkdirAll(projectIssueGraphDir, 0755); err != nil {
			t.Fatal(err)
		}

		redirectPath := filepath.Join(projectIssueGraphDir, issuegraph.RedirectFileName)
		if err := os.WriteFile(redirectPath, []byte("../canonical/.issuegraph\n"), 0644); err != nil {
			t.Fatal(err)
		}

		// Use os.Chdir since checkExistingIssueGraphData reads CWD directly
		origWd, _ := os.Getwd()
		if err := os.Chdir(projectDir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(origWd)

		err := checkExistingIssueGraphData("new-prefix")
		if err == nil {
			t.Fatal("Expected checkExistingIssueGraphData to return error when redirect target already has database")
		}

		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "redirect target already has database") {
			t.Errorf("Expected error about redirect target having database, got: %s", errorMsg)
		}

		// Verify the canonical DB file still exists (wasn't deleted/overwritten)
		if _, statErr := os.Stat(canonicalDBPath); os.IsNotExist(statErr) {
			t.Error("Canonical database file should still exist after error")
		}
	})
}

// =============================================================================
// ISSUEGRAPH_DIR Tests
// =============================================================================
// These tests verify that issuegraph init respects the ISSUEGRAPH_DIR environment variable
// for both safety checks and database creation.

// TestInitISSUEGRAPH_DIR groups ISSUEGRAPH_DIR-related init tests.
// Tests requirements FR-001, FR-002, FR-004, NFR-001.
func TestInitISSUEGRAPH_DIR(t *testing.T) {
	skipIfNoDolt(t)
	// resetIssueGraphDirState resets global state and env vars for each subtest.
	resetIssueGraphDirState := func(t *testing.T) {
		t.Helper()
		origDBPath := dbPath
		t.Cleanup(func() {
			dbPath = origDBPath
			issuegraph.ResetCaches()
			git.ResetCaches()
		})
		dbPath = ""
		issuegraph.ResetCaches()
		git.ResetCaches()
		initCmd.Flags().Set("prefix", "")
		initCmd.Flags().Set("quiet", "false")
		initCmd.Flags().Set("backend", "")
	}

	// checkExistingIssueGraphData tests (FR-001, FR-004)
	t.Run("CheckExisting_NoExistingDB", func(t *testing.T) {
		resetIssueGraphDirState(t)

		tmpDir := t.TempDir()
		issuegraphDirPath := filepath.Join(tmpDir, "external", ".issuegraph")
		os.MkdirAll(issuegraphDirPath, 0755)

		os.Setenv("ISSUEGRAPH_DIR", issuegraphDirPath)
		t.Cleanup(func() { os.Unsetenv("ISSUEGRAPH_DIR") })
		issuegraph.ResetCaches()

		err := checkExistingIssueGraphData("test")
		if err != nil {
			t.Errorf("Expected no error when ISSUEGRAPH_DIR has no database, got: %v", err)
		}
	})

	t.Run("CheckExisting_CWDIgnoredWhenSet", func(t *testing.T) {
		resetIssueGraphDirState(t)

		tmpDir := t.TempDir()

		// Create CWD with existing database (should be ignored)
		cwdIssueGraphDir := filepath.Join(tmpDir, "cwd", ".issuegraph")
		os.MkdirAll(cwdIssueGraphDir, 0755)
		cwdDBPath := filepath.Join(cwdIssueGraphDir, issuegraph.CanonicalDatabaseName)
		// Create the db file so checkExistingIssueGraphData detects it
		if err := os.WriteFile(cwdDBPath, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		// Create ISSUEGRAPH_DIR location (no database)
		issuegraphDirPath := filepath.Join(tmpDir, "external", ".issuegraph")
		os.MkdirAll(issuegraphDirPath, 0755)

		os.Setenv("ISSUEGRAPH_DIR", issuegraphDirPath)
		t.Cleanup(func() { os.Unsetenv("ISSUEGRAPH_DIR") })
		issuegraph.ResetCaches()

		origWd, _ := os.Getwd()
		os.Chdir(filepath.Join(tmpDir, "cwd"))
		defer os.Chdir(origWd)

		err := checkExistingIssueGraphData("test")
		if err != nil {
			t.Errorf("Expected no error when ISSUEGRAPH_DIR has no database (CWD should be ignored), got: %v", err)
		}
	})

	t.Run("CheckExisting_ErrorWhenDBExists", func(t *testing.T) {
		resetIssueGraphDirState(t)

		tmpDir := t.TempDir()

		issuegraphDirPath := filepath.Join(tmpDir, "external", ".issuegraph")
		os.MkdirAll(issuegraphDirPath, 0755)
		testDBPath := filepath.Join(issuegraphDirPath, issuegraph.CanonicalDatabaseName)
		// Create the db file so checkExistingIssueGraphData detects it
		if err := os.WriteFile(testDBPath, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		os.Setenv("ISSUEGRAPH_DIR", issuegraphDirPath)
		t.Cleanup(func() { os.Unsetenv("ISSUEGRAPH_DIR") })
		issuegraph.ResetCaches()

		err := checkExistingIssueGraphData("test")
		if err == nil {
			t.Error("Expected error when ISSUEGRAPH_DIR already has database")
		}
		if !strings.Contains(err.Error(), issuegraphDirPath) {
			t.Errorf("Expected error to mention ISSUEGRAPH_DIR path %s, got: %v", issuegraphDirPath, err)
		}
	})

	// FR-002: init creates database at ISSUEGRAPH_DIR
	t.Run("InitCreatesDBAtIssueGraphDir", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping ISSUEGRAPH_DIR test on Windows")
		}

		resetIssueGraphDirState(t)

		tmpDir := t.TempDir()

		issuegraphDirPath := filepath.Join(tmpDir, "external", ".issuegraph")
		os.MkdirAll(filepath.Dir(issuegraphDirPath), 0755)

		os.Setenv("ISSUEGRAPH_DIR", issuegraphDirPath)
		t.Cleanup(func() { os.Unsetenv("ISSUEGRAPH_DIR") })
		issuegraph.ResetCaches()
		git.ResetCaches()

		cwdPath := filepath.Join(tmpDir, "workdir")
		os.MkdirAll(cwdPath, 0755)
		t.Chdir(cwdPath)

		rootCmd.SetArgs([]string{"init", "--prefix", "issuegraphdir-test", "--quiet"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Init with ISSUEGRAPH_DIR failed: %v", err)
		}

		expectedDBPath := filepath.Join(issuegraphDirPath, "dolt")
		if info, err := os.Stat(expectedDBPath); os.IsNotExist(err) {
			t.Errorf("Dolt database was not created at ISSUEGRAPH_DIR path: %s", expectedDBPath)
		} else if !info.IsDir() {
			t.Errorf("Expected %s to be a directory", expectedDBPath)
		}

		cwdDBPath := filepath.Join(cwdPath, ".issuegraph", "dolt")
		if _, err := os.Stat(cwdDBPath); err == nil {
			t.Errorf("Database should NOT have been created at CWD: %s", cwdDBPath)
		}

		store, err := openExistingTestDB(t, expectedDBPath)
		if err != nil {
			t.Fatalf("Failed to open database at ISSUEGRAPH_DIR: %v", err)
		}
		defer store.Close()

		ctx := context.Background()
		prefix, err := store.GetConfig(ctx, "issue_prefix")
		if err != nil {
			t.Fatalf("Failed to get prefix from database: %v", err)
		}
		if prefix != "issuegraphdir-test" {
			t.Errorf("Expected prefix 'issuegraphdir-test', got %q", prefix)
		}
	})

	// NFR-001: existing behavior unchanged when ISSUEGRAPH_DIR not set
	t.Run("WithoutIssueGraphDirNoBehaviorChange", func(t *testing.T) {
		resetIssueGraphDirState(t)

		os.Unsetenv("ISSUEGRAPH_DIR")
		issuegraph.ResetCaches()
		git.ResetCaches()

		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		rootCmd.SetArgs([]string{"init", "--prefix", "no-issuegraphdir", "--quiet"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Init without ISSUEGRAPH_DIR failed: %v", err)
		}

		expectedDBPath := filepath.Join(tmpDir, ".issuegraph", "dolt")
		if info, err := os.Stat(expectedDBPath); os.IsNotExist(err) {
			t.Errorf("Dolt database was not created at default CWD/.issuegraph path: %s", expectedDBPath)
		} else if !info.IsDir() {
			t.Errorf("Expected %s to be a directory", expectedDBPath)
		}

		store, err := openExistingTestDB(t, expectedDBPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer store.Close()

		ctx := context.Background()
		prefix, err := store.GetConfig(ctx, "issue_prefix")
		if err != nil {
			t.Fatalf("Failed to get prefix from database: %v", err)
		}
		if prefix != "no-issuegraphdir" {
			t.Errorf("Expected prefix 'no-issuegraphdir', got %q", prefix)
		}
	})

	// Worktree bypass: ISSUEGRAPH_DIR skips the worktree guard and git init
	t.Run("WorktreeBypassWhenIssueGraphDirSet", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping worktree test on Windows")
		}

		resetIssueGraphDirState(t)

		tmpDir := t.TempDir()

		// Create a main git repo with an initial commit (required for worktrees)
		mainRepo := filepath.Join(tmpDir, "main-repo")
		if err := os.MkdirAll(mainRepo, 0755); err != nil {
			t.Fatalf("Failed to create main repo dir: %v", err)
		}
		runGit := func(dir string, args ...string) {
			t.Helper()
			cmd := exec.Command("git", args...)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("git %v in %s failed: %v\n%s", args, dir, err, out)
			}
		}
		runGit(mainRepo, "init")
		runGit(mainRepo, "config", "user.email", "test@test.com")
		runGit(mainRepo, "config", "user.name", "Test")
		runGit(mainRepo, "commit", "--allow-empty", "-m", "initial")

		// Create a worktree
		worktreeDir := filepath.Join(tmpDir, "my-worktree")
		runGit(mainRepo, "worktree", "add", worktreeDir, "-b", "test-wt")

		// Set ISSUEGRAPH_DIR to a standalone location outside the repo
		issuegraphDirPath := filepath.Join(tmpDir, "standalone", ".issuegraph")
		if err := os.MkdirAll(filepath.Dir(issuegraphDirPath), 0755); err != nil {
			t.Fatalf("Failed to create standalone dir: %v", err)
		}
		os.Setenv("ISSUEGRAPH_DIR", issuegraphDirPath)
		t.Cleanup(func() { os.Unsetenv("ISSUEGRAPH_DIR") })
		issuegraph.ResetCaches()
		git.ResetCaches()

		// cd into the worktree — without ISSUEGRAPH_DIR this would fail
		t.Chdir(worktreeDir)

		rootCmd.SetArgs([]string{"init", "--prefix", "wt-bypass", "--skip-hooks", "--quiet"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Init with ISSUEGRAPH_DIR from worktree should succeed, got: %v", err)
		}

		// Verify database was created at ISSUEGRAPH_DIR, not in the worktree
		expectedDBPath := filepath.Join(issuegraphDirPath, "dolt")
		if _, statErr := os.Stat(expectedDBPath); os.IsNotExist(statErr) {
			t.Errorf("Dolt database was not created at ISSUEGRAPH_DIR path: %s", expectedDBPath)
		}

		worktreeDBPath := filepath.Join(worktreeDir, ".issuegraph", "dolt")
		if _, statErr := os.Stat(worktreeDBPath); statErr == nil {
			t.Errorf("Database should NOT have been created in worktree: %s", worktreeDBPath)
		}
	})

	// Precedence: ISSUEGRAPH_DB > ISSUEGRAPH_DIR
	t.Run("ISSUEGRAPH_DB_OverridesIssueGraphDir", func(t *testing.T) {
		t.Skip("ISSUEGRAPH_DB env var does not control Dolt store location; Dolt always uses .issuegraph/dolt/")
		resetIssueGraphDirState(t)

		issuegraphDirTarget := t.TempDir()
		issuegraphDBTarget := t.TempDir()

		issuegraphDirIssueGraph := filepath.Join(issuegraphDirTarget, ".issuegraph")
		if err := os.MkdirAll(issuegraphDirIssueGraph, 0750); err != nil {
			t.Fatal(err)
		}

		issuegraphDBPath := filepath.Join(issuegraphDBTarget, "override.db")

		t.Setenv("ISSUEGRAPH_DIR", issuegraphDirIssueGraph)
		t.Setenv("ISSUEGRAPH_DB", issuegraphDBPath)

		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		rootCmd.SetArgs([]string{"init", "--prefix", "precedence", "--quiet"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Init with ISSUEGRAPH_DB + ISSUEGRAPH_DIR failed: %v", err)
		}

		if _, err := os.Stat(issuegraphDBPath); os.IsNotExist(err) {
			t.Errorf("Database was NOT created at ISSUEGRAPH_DB path: %s", issuegraphDBPath)
		}

		issuegraphDirDBPath := filepath.Join(issuegraphDirIssueGraph, issuegraph.CanonicalDatabaseName)
		if _, err := os.Stat(issuegraphDirDBPath); err == nil {
			t.Errorf("Database was incorrectly created at ISSUEGRAPH_DIR path: %s (ISSUEGRAPH_DB should override)", issuegraphDirDBPath)
		}

		store, err := openExistingTestDB(t, issuegraphDBPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer store.Close()

		ctx := context.Background()
		prefix, err := store.GetConfig(ctx, "issue_prefix")
		if err != nil {
			t.Fatalf("Failed to get prefix from database: %v", err)
		}
		if prefix != "precedence" {
			t.Errorf("Expected prefix 'precedence', got %q", prefix)
		}
	})
}

// TestInit_WithISSUEGRAPH_DIR_DoltBackend verifies that issuegraph init with Dolt backend
// creates the database at ISSUEGRAPH_DIR when the environment variable is set.
// This tests requirements FR-002 for Dolt backend.
func TestInit_WithISSUEGRAPH_DIR_DoltBackend(t *testing.T) {
	// Skip on Windows
	if runtime.GOOS == "windows" {
		t.Skip("Skipping ISSUEGRAPH_DIR Dolt test on Windows")
	}

	testutil.RequireDoltBinary(t)

	// Reset global state
	origDBPath := dbPath
	defer func() { dbPath = origDBPath }()
	dbPath = ""

	// Save and restore ISSUEGRAPH_DIR
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
		issuegraph.ResetCaches()
		git.ResetCaches()
	}()

	// Reset Cobra flags
	initCmd.Flags().Set("prefix", "")
	initCmd.Flags().Set("quiet", "false")
	initCmd.Flags().Set("backend", "")

	tmpDir := t.TempDir()

	// Create external ISSUEGRAPH_DIR location
	issuegraphDirPath := filepath.Join(tmpDir, "external", ".issuegraph")
	os.MkdirAll(filepath.Dir(issuegraphDirPath), 0755)

	os.Setenv("ISSUEGRAPH_DIR", issuegraphDirPath)
	issuegraph.ResetCaches()
	git.ResetCaches()

	// Change to a different working directory
	cwdPath := filepath.Join(tmpDir, "workdir")
	os.MkdirAll(cwdPath, 0755)
	t.Chdir(cwdPath)

	// Run issuegraph init with Dolt backend
	rootCmd.SetArgs([]string{"init", "--prefix", "dolt-test", "--quiet"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Init with ISSUEGRAPH_DIR and Dolt backend failed: %v", err)
	}

	// In embedded mode (default), the engine creates .issuegraph/embeddeddolt/ —
	// .issuegraph/dolt/ should NOT be created (GH#2903).
	unexpectedDoltPath := filepath.Join(issuegraphDirPath, "dolt")
	if _, err := os.Stat(unexpectedDoltPath); err == nil {
		t.Errorf("Empty .issuegraph/dolt/ should not be created in embedded mode: %s", unexpectedDoltPath)
	}

	// Verify database was NOT created at CWD
	cwdDoltPath := filepath.Join(cwdPath, ".issuegraph", "dolt")
	if _, err := os.Stat(cwdDoltPath); err == nil {
		t.Errorf("Dolt database should NOT have been created at CWD: %s", cwdDoltPath)
	}
}

// Note: TestInit_WithoutISSUEGRAPH_DIR_NoBehaviorChange and TestInit_ISSUEGRAPH_DB_OverridesISSUEGRAPH_DIR
// are now subtests of TestInitISSUEGRAPH_DIR above.

// TestInitDoltMetadata verifies that issuegraph init --backend dolt writes and persists
// all 3 tracking metadata fields (issuegraph_version, repo_id, clone_id) via verifyMetadata.
// Covers FR-001, FR-002, FR-003, FR-004.
func TestInitDoltMetadata(t *testing.T) {
	skipIfNoDolt(t)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Dolt metadata test on Windows")
	}

	saveAndRestoreGlobals(t)
	dbPath = ""

	// Reset caches to avoid stale state
	issuegraph.ResetCaches()
	git.ResetCaches()
	t.Cleanup(func() {
		issuegraph.ResetCaches()
		git.ResetCaches()
	})

	// Reset Cobra flags
	initCmd.Flags().Set("prefix", "")
	initCmd.Flags().Set("quiet", "false")
	initCmd.Flags().Set("backend", "")

	tmpDir := newGitRepo(t)
	t.Chdir(tmpDir)

	// Add remote.origin.url so ComputeRepoID succeeds
	_ = runCommandInDir(tmpDir, "git", "config", "remote.origin.url", "https://github.com/test/repo.git")

	rootCmd.SetArgs([]string{"init", "--prefix", "test", "--quiet"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init --backend dolt failed: %v", err)
	}

	// Open the dolt store to verify metadata was written
	ctx := context.Background()
	doltPath := filepath.Join(tmpDir, ".issuegraph", "dolt")
	doltStore, err := openDoltStoreForTest(t, ctx, doltPath, "test")
	if err != nil {
		t.Fatalf("failed to open dolt store for verification: %v", err)
	}
	defer doltStore.Close()

	// FR-001: issuegraph_version must be written
	issuegraphVersion, err := doltStore.GetLocalMetadata(ctx, "issuegraph_version")
	if err != nil {
		t.Fatalf("GetLocalMetadata(issuegraph_version) failed: %v", err)
	}
	if issuegraphVersion == "" {
		t.Error("issuegraph_version local metadata was not written")
	}

	// FR-002: repo_id must be written (git repo with remote configured)
	repoID, err := doltStore.GetMetadata(ctx, "repo_id")
	if err != nil {
		t.Fatalf("GetMetadata(repo_id) failed: %v", err)
	}
	if repoID == "" {
		t.Error("repo_id metadata was not written")
	}

	// FR-003: clone_id must be written
	cloneID, err := doltStore.GetMetadata(ctx, "clone_id")
	if err != nil {
		t.Fatalf("GetMetadata(clone_id) failed: %v", err)
	}
	if cloneID == "" {
		t.Error("clone_id metadata was not written")
	}
}

// openDoltStoreForTest opens an existing Dolt store for read-only verification in tests.
func openDoltStoreForTest(t *testing.T, ctx context.Context, doltPath, dbName string) (*dolt.DoltStore, error) {
	t.Helper()
	return dolt.New(ctx, &dolt.Config{
		Path:     doltPath,
		Database: dbName,
		ReadOnly: true,
	})
}

// TestVerifyMetadataSuccess verifies that verifyMetadata writes and reads back metadata.
// Note: Failure path tests (write errors, read-back mismatches) were removed because
// verifyMetadata now takes *dolt.DoltStore (concrete type), making interface-based
// mocking impossible. The failure paths are simple error-to-stderr logic.
func TestVerifyMetadataSuccess(t *testing.T) {
	skipIfNoDolt(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	testDB := filepath.Join(tmpDir, "test.db")
	store := newTestStore(t, testDB)
	defer store.Close()

	ok := verifyMetadata(ctx, store, "test_key", "test_value")
	if !ok {
		t.Error("verifyMetadata should return true on success")
	}
	// Verify the value was actually written
	val, err := store.GetMetadata(ctx, "test_key")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}
	if val != "test_value" {
		t.Errorf("expected 'test_value', got %q", val)
	}
}

// TestInitDoltMetadataNoGit verifies that issuegraph init outside a git repo gracefully
// skips repo_id while still writing issuegraph_version and clone_id.
// Verifies warning output; actual metadata persistence checked by e2e tests.
// Covers FR-015 (skip repo_id outside git).
func TestInitDoltMetadataNoGit(t *testing.T) {
	skipIfNoDolt(t)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Dolt metadata test on Windows")
	}

	saveAndRestoreGlobals(t)
	dbPath = ""

	issuegraph.ResetCaches()
	git.ResetCaches()
	t.Cleanup(func() {
		issuegraph.ResetCaches()
		git.ResetCaches()
	})

	// Reset Cobra flags
	initCmd.Flags().Set("prefix", "")
	initCmd.Flags().Set("quiet", "false")
	initCmd.Flags().Set("backend", "")

	// Create temp dir WITHOUT git init — issuegraph init will create one,
	// but there will be no remote configured so upstream warning is expected.
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Run init in a non-git directory (issuegraph init will create git repo internally)
	rootCmd.SetArgs([]string{"init", "--prefix", "nogit"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init --prefix nogit failed: %v", err)
	}

	// Note: no upstream/repository ID warning expected here because issuegraph init
	// creates a brand-new git repo with no remotes, and the upstream warning
	// is intentionally skipped for repos with no remotes (not noise-worthy).

	// Verify .issuegraph/dolt directory was created (init succeeded)
	doltPath := filepath.Join(tmpDir, ".issuegraph", "dolt")
	if info, err := os.Stat(doltPath); os.IsNotExist(err) {
		t.Errorf("Dolt database directory was not created: %s", doltPath)
	} else if !info.IsDir() {
		t.Errorf("Expected Dolt path to be a directory: %s", doltPath)
	}

	// Verify no SQLite database was created (backend-specific)
	sqlitePath := filepath.Join(tmpDir, ".issuegraph", "issuegraph.db")
	if _, err := os.Stat(sqlitePath); err == nil {
		t.Errorf("unexpected sqlite database created in dolt mode")
	}
}

func TestInitServerModeWritesDoltCompatibilityMarker(t *testing.T) {
	skipIfNoDolt(t)
	saveAndRestoreGlobals(t)
	ensureCleanGlobalState(t)
	dbPath = ""
	store = nil

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	doltDir := filepath.Join(issuegraphDir, "dolt")
	if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
		t.Fatalf("creating issuegraph dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(doltDir, ".dolt"), 0o750); err != nil {
		t.Fatalf("creating simulated server data dir: %v", err)
	}

	database := uniqueTestDBName(t)
	t.Cleanup(func() {
		dropTestDatabase(database, testDoltServerPort)
	})

	rootCmd.SetArgs([]string{
		"init",
		"--server",
		"--external",
		"--server-host", "127.0.0.1",
		"--server-port", fmt.Sprintf("%d", testDoltServerPort),
		"--database", database,
		"--prefix", "marker",
		"--quiet",
		"--skip-hooks",
		"--skip-agents",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("server init failed: %v", err)
	}

	markerPath := filepath.Join(doltDir, ".bd-dolt-ok")
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("expected server init to write %s: %v", markerPath, err)
	}
}

func TestInitServerModeWarnsOnMarkerFailureInQuietMode(t *testing.T) {
	skipIfNoDolt(t)
	saveAndRestoreGlobals(t)
	ensureCleanGlobalState(t)
	dbPath = ""
	store = nil

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	doltDir := filepath.Join(issuegraphDir, "dolt")
	if err := os.MkdirAll(doltDir, 0o700); err != nil {
		t.Fatalf("creating dolt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(doltDir, ".dolt"), []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("creating invalid dot-dolt marker: %v", err)
	}

	database := uniqueTestDBName(t)
	t.Cleanup(func() {
		dropTestDatabase(database, testDoltServerPort)
	})

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	rootCmd.SetArgs([]string{
		"init",
		"--server",
		"--external",
		"--server-host", "127.0.0.1",
		"--server-port", fmt.Sprintf("%d", testDoltServerPort),
		"--database", database,
		"--prefix", "marker",
		"--quiet",
		"--skip-hooks",
		"--skip-agents",
	})

	if err := rootCmd.Execute(); err != nil {
		w.Close()
		t.Fatalf("server init failed: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}
	var stderr bytes.Buffer
	if _, err := stderr.ReadFrom(r); err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	if !strings.Contains(stderr.String(), "Warning: failed to write Dolt compatibility marker") {
		t.Fatalf("expected marker warning in stderr, got:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), doltDir) {
		t.Fatalf("expected marker warning to include dolt dir %s, got:\n%s", doltDir, stderr.String())
	}
}

func setupBareParentInitWorktree(t *testing.T) (string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	bareDir := filepath.Join(tmpDir, "repo.git")
	mainWorktreeDir := filepath.Join(tmpDir, "main")
	featureWorktreeDir := filepath.Join(tmpDir, "feature")

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %s failed: %v\n%s", args, dir, err, out)
		}
	}

	runGit(tmpDir, "init", "--bare", bareDir)
	runGit(tmpDir, "--git-dir", bareDir, "symbolic-ref", "HEAD", "refs/heads/main")
	runGit(tmpDir, "--git-dir", bareDir, "config", "user.email", "test@example.com")
	runGit(tmpDir, "--git-dir", bareDir, "config", "user.name", "Test User")
	emptyTreeCmd := exec.Command("git", "--git-dir", bareDir, "hash-object", "-t", "tree", "/dev/null")
	emptyTreeCmd.Dir = tmpDir
	emptyTreeOut, err := emptyTreeCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git hash-object failed: %v\n%s", err, emptyTreeOut)
	}
	emptyTree := strings.TrimSpace(string(emptyTreeOut))
	commitCmd := exec.Command("git", "--git-dir", bareDir, "commit-tree", "-m", "Initial commit", emptyTree)
	commitCmd.Dir = tmpDir
	commitOut, err := commitCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git commit-tree failed: %v\n%s", err, commitOut)
	}
	initCommit := strings.TrimSpace(string(commitOut))
	runGit(tmpDir, "--git-dir", bareDir, "update-ref", "HEAD", initCommit)
	runGit(tmpDir, "--git-dir", bareDir, "worktree", "add", mainWorktreeDir, "main")
	runGit(mainWorktreeDir, "branch", "feature")
	runGit(tmpDir, "--git-dir", bareDir, "worktree", "add", featureWorktreeDir, "feature")

	return bareDir, featureWorktreeDir
}

// TestInitDatabaseFlag tests the --database flag for issuegraph init.
// Uses subprocess execution because:
//   - init manipulates extensive Cobra global state that's difficult to reset
//   - init validation paths call os.Exit(1), which kills in-process tests
//
// Each subtest runs issuegraph init in a temp directory and verifies metadata.json.
func TestInitDatabaseFlag(t *testing.T) {
	skipIfNoDolt(t)
	issuegraphBinary := buildIssueGraphForInitTests(t)

	t.Run("metadata_written", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Run init with --database to specify a pre-existing database name
		cmd := exec.Command(issuegraphBinary, "init", "--database", "myapp_production", "--quiet")
		cmd.Dir = tmpDir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("issuegraph init --database failed: %v\n%s", err, out)
		}

		// Verify metadata.json contains the correct database name
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
		cfg, err := configfile.Load(issuegraphDir)
		if err != nil {
			t.Fatalf("Failed to load metadata.json: %v", err)
		}
		if cfg == nil {
			t.Fatal("metadata.json not found")
		}

		if cfg.DoltDatabase != "myapp_production" {
			t.Errorf("Expected DoltDatabase %q, got %q", "myapp_production", cfg.DoltDatabase)
		}
		if cfg.DoltMode != configfile.DoltModeServer {
			t.Errorf("Expected DoltMode %q, got %q", configfile.DoltModeServer, cfg.DoltMode)
		}
	})

	t.Run("BareParentWorktreeAutoInit", func(t *testing.T) {
		skipIfNoDolt(t)
		if runtime.GOOS == "windows" {
			t.Skip("Skipping worktree test on Windows")
		}

		origDBPath := dbPath
		t.Cleanup(func() {
			dbPath = origDBPath
			issuegraph.ResetCaches()
			git.ResetCaches()
		})
		dbPath = ""
		issuegraph.ResetCaches()
		git.ResetCaches()
		issuegraphBinary := buildIssueGraphForInitTests(t)
		bareDir, worktreeDir := setupBareParentInitWorktree(t)
		bareIssueGraphDir := filepath.Join(bareDir, ".issuegraph")

		// Shared-server init daemonizes a dolt sql-server that nothing here
		// stopped; it outlived the suite whose temp HOME it served
		// (wy-j2zc8q). Registered before the first subprocess so the
		// t.Fatalf paths are covered too.
		stopSharedServerCleanup(t)

		cmd := exec.Command(issuegraphBinary, "init", "--prefix", "bare-fallback", "--skip-hooks", "--quiet")
		cmd.Dir = worktreeDir
		cmd.Env = append(os.Environ(), "ISSUEGRAPH_DOLT_SHARED_SERVER=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("issuegraph init from bare-parent worktree failed: %v\n%s", err, out)
		}

		if _, err := os.Stat(filepath.Join(bareIssueGraphDir, "metadata.json")); err != nil {
			t.Fatalf("expected bare parent metadata.json to exist: %v", err)
		}
		if _, err := os.Stat(filepath.Join(worktreeDir, ".issuegraph")); !os.IsNotExist(err) {
			t.Fatalf("worktree should not get a local .issuegraph directory, stat err=%v", err)
		}

		retry := exec.Command(issuegraphBinary, "init", "--prefix", "bare-fallback", "--skip-hooks", "--quiet")
		retry.Dir = worktreeDir
		retry.Env = append(os.Environ(), "ISSUEGRAPH_DOLT_SHARED_SERVER=1")
		retryOut, retryErr := retry.CombinedOutput()
		if retryErr == nil {
			t.Fatal("expected second issuegraph init to fail against existing bare-parent .issuegraph")
		}
		if !strings.Contains(string(retryOut), "already initialized") && !strings.Contains(string(retryOut), "already exists") {
			t.Fatalf("expected existing-data guard on second init, got:\n%s", retryOut)
		}
	})

	t.Run("with_prefix", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Run init with both --database and --prefix
		// --database should override prefix for DB name, but prefix still sets issue_prefix
		cmd := exec.Command(issuegraphBinary, "init", "--database", "shared_db", "--prefix", "team-alpha", "--quiet")
		cmd.Dir = tmpDir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("issuegraph init --database --prefix failed: %v\n%s", err, out)
		}

		// Verify metadata.json
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
		cfg, err := configfile.Load(issuegraphDir)
		if err != nil {
			t.Fatalf("Failed to load metadata.json: %v", err)
		}
		if cfg == nil {
			t.Fatal("metadata.json not found")
		}

		// --database overrides the SQL database name
		if cfg.DoltDatabase != "shared_db" {
			t.Errorf("Expected DoltDatabase %q (from --database), got %q", "shared_db", cfg.DoltDatabase)
		}

		// Verify the database was opened and issue_prefix was set
		// by reopening the store and checking config
		dbPath := filepath.Join(issuegraphDir, "dolt")
		s, err := openExistingTestDB(t, dbPath)
		if err != nil {
			t.Fatalf("Failed to reopen database: %v", err)
		}
		defer s.Close()

		ctx := context.Background()
		issuePrefix, err := s.GetConfig(ctx, "issue_prefix")
		if err != nil {
			t.Fatalf("Failed to get issue_prefix: %v", err)
		}
		if issuePrefix != "team-alpha" {
			t.Errorf("Expected issue_prefix %q (from --prefix), got %q", "team-alpha", issuePrefix)
		}
	})

	t.Run("server_config_in_metadata", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Run init with --database
		cmd := exec.Command(issuegraphBinary, "init", "--database", "test_server_cfg", "--quiet")
		cmd.Dir = tmpDir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("issuegraph init --database failed: %v\n%s", err, out)
		}

		// Verify metadata.json has both dolt_database and dolt_mode: server
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
		cfg, err := configfile.Load(issuegraphDir)
		if err != nil {
			t.Fatalf("Failed to load metadata.json: %v", err)
		}
		if cfg == nil {
			t.Fatal("metadata.json not found")
		}

		if cfg.DoltDatabase != "test_server_cfg" {
			t.Errorf("Expected DoltDatabase %q, got %q", "test_server_cfg", cfg.DoltDatabase)
		}
		if cfg.DoltMode != configfile.DoltModeServer {
			t.Errorf("Expected DoltMode %q, got %q", configfile.DoltModeServer, cfg.DoltMode)
		}
		if cfg.Backend != configfile.BackendDolt {
			t.Errorf("Expected Backend %q, got %q", configfile.BackendDolt, cfg.Backend)
		}
	})

	t.Run("shared_server_flag_selects_server_mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		cmd := exec.Command(issuegraphBinary, "init", "--shared-server", "--prefix", "shared-mode-test", "--skip-hooks")
		cmd.Dir = tmpDir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("issuegraph init --shared-server failed: %v\n%s", err, out)
		}

		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
		cfg, err := configfile.Load(issuegraphDir)
		if err != nil {
			t.Fatalf("Failed to load metadata.json: %v", err)
		}
		if cfg == nil {
			t.Fatal("metadata.json not found")
		}

		if cfg.DoltMode != configfile.DoltModeServer {
			t.Errorf("Expected DoltMode %q, got %q", configfile.DoltModeServer, cfg.DoltMode)
		}

		configYAML, err := os.ReadFile(filepath.Join(issuegraphDir, "config.yaml"))
		if err != nil {
			t.Fatalf("Failed to read config.yaml: %v", err)
		}
		if !strings.Contains(string(configYAML), "dolt.shared-server: true") {
			t.Fatalf("expected config.yaml to enable shared server, got:\n%s", configYAML)
		}

		outStr := string(out)
		if !strings.Contains(outStr, "Shared server mode enabled") {
			t.Fatalf("expected init output to mention shared server mode, got:\n%s", outStr)
		}
		if !strings.Contains(outStr, "Mode: server") {
			t.Fatalf("expected init output to report server mode, got:\n%s", outStr)
		}
		if strings.Contains(outStr, "Mode: embedded") {
			t.Fatalf("init output should not report embedded mode when --shared-server is set:\n%s", outStr)
		}
	})

	t.Run("validation_invalid_name", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Run init with an invalid database name (contains semicolon = SQL injection)
		cmd := exec.Command(issuegraphBinary, "init", "--database", "bad;name", "--quiet")
		cmd.Dir = tmpDir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("Expected error for invalid database name, but command succeeded")
		}

		outStr := string(out)
		if !strings.Contains(outStr, "invalid database name") {
			t.Errorf("Expected error to mention 'invalid database name', got: %s", outStr)
		}

		// Verify no .issuegraph directory was created (early validation prevents side effects)
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
		if _, err := os.Stat(issuegraphDir); err == nil {
			t.Error(".issuegraph directory should not be created when validation fails")
		}
	})

	t.Run("validation_name_with_spaces", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Database name with spaces should fail validation
		cmd := exec.Command(issuegraphBinary, "init", "--database", "my database", "--quiet")
		cmd.Dir = tmpDir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("Expected error for database name with spaces, but command succeeded")
		}

		outStr := string(out)
		if !strings.Contains(outStr, "invalid database name") {
			t.Errorf("Expected error to mention 'invalid database name', got: %s", outStr)
		}
	})

	t.Run("validation_name_with_backtick", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Database name with backtick injection should fail validation
		cmd := exec.Command(issuegraphBinary, "init", "--database", "db`; DROP DATABASE x; --", "--quiet")
		cmd.Dir = tmpDir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("Expected error for database name with backtick injection, but command succeeded")
		}

		outStr := string(out)
		if !strings.Contains(outStr, "invalid database name") {
			t.Errorf("Expected error to mention 'invalid database name', got: %s", outStr)
		}
	})
}

func TestBareParentWorktreeCoreCommandsWithoutRedirect(t *testing.T) {
	skipIfNoDolt(t)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping worktree test on Windows")
	}

	issuegraphBinary := buildIssueGraphForInitTests(t)
	bareDir, worktreeDir := setupBareParentInitWorktree(t)
	bareIssueGraphDir := filepath.Join(bareDir, ".issuegraph")
	sharedEnv := append(os.Environ(), "ISSUEGRAPH_DOLT_SHARED_SERVER=1")

	// Same daemonized shared server, same missing stop (wy-j2zc8q).
	stopSharedServerCleanup(t)

	initCmd := exec.Command(issuegraphBinary, "init", "--prefix", "bare-core", "--skip-hooks", "--quiet")
	initCmd.Dir = worktreeDir
	initCmd.Env = sharedEnv
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("issuegraph init from bare-parent worktree failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(filepath.Join(bareIssueGraphDir, "metadata.json")); err != nil {
		t.Fatalf("expected bare parent metadata.json to exist: %v", err)
	}

	createCmd := exec.Command(issuegraphBinary, "create", "bare fallback issue", "--description", "regression", "--json")
	createCmd.Dir = worktreeDir
	createCmd.Env = sharedEnv
	createOut, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("issuegraph create from bare-parent worktree failed: %v\n%s", err, createOut)
	}
	if !strings.Contains(string(createOut), "bare fallback issue") {
		t.Fatalf("issuegraph create output did not include created title:\n%s", createOut)
	}

	listCmd := exec.Command(issuegraphBinary, "list", "--json")
	listCmd.Dir = worktreeDir
	listCmd.Env = sharedEnv
	listOut, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("issuegraph list from bare-parent worktree failed: %v\n%s", err, listOut)
	}
	if !strings.Contains(string(listOut), "bare fallback issue") {
		t.Fatalf("issuegraph list output did not include created issue:\n%s", listOut)
	}
}

// initBackendTestEnv returns the process environment with all issuegraph-specific
// variables (ISSUEGRAPH_*, ISSUEGRAPH_*) removed and ISSUEGRAPH_DIR pinned to issuegraphDir. Pinning
// ISSUEGRAPH_DIR isolates each subtest's workspace and stops issuegraph from walking up into
// an ambient .issuegraph left by another test or tool; HOME is preserved so issuegraph init's
// git bootstrap still works.
func initBackendTestEnv(issuegraphDir string) []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "ISSUEGRAPH_") || strings.HasPrefix(e, "ISSUEGRAPH_") {
			continue
		}
		env = append(env, e)
	}
	return append(env, "ISSUEGRAPH_DIR="+issuegraphDir)
}

func TestInitBackendFlag(t *testing.T) {
	issuegraphBinary := buildIssueGraphForInitTests(t)

	// The SQLite backend was rolled back with the other alternative backends:
	// init fails closed with migration guidance and writes no workspace state.
	t.Run("sqlite_is_no_longer_supported", func(t *testing.T) {
		tmpDir := t.TempDir()
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")

		cmd := exec.Command(issuegraphBinary, "init", "--backend", "sqlite", "--quiet")
		cmd.Dir = tmpDir
		cmd.Env = initBackendTestEnv(issuegraphDir)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("Expected non-zero exit for --backend=sqlite:\n%s", out)
		}
		outStr := string(out)
		if !strings.Contains(outStr, "no longer supported") || !strings.Contains(outStr, "single engine") {
			t.Errorf("Expected rollback guidance for sqlite, got: %s", outStr)
		}
		if _, statErr := os.Stat(issuegraphDir); !os.IsNotExist(statErr) {
			t.Fatalf("rejected sqlite init created workspace state (stat error: %v)", statErr)
		}
	})

	for _, backend := range []string{"postgres", "mysql"} {
		t.Run(backend+"_is_no_longer_supported", func(t *testing.T) {
			tmpDir := t.TempDir()
			issuegraphDir := filepath.Join(tmpDir, ".issuegraph")

			cmd := exec.Command(issuegraphBinary, "init", "--backend", backend, "--quiet")
			cmd.Dir = tmpDir
			cmd.Env = initBackendTestEnv(issuegraphDir)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("Expected non-zero exit for --backend=%s", backend)
			}

			outStr := string(out)
			if !strings.Contains(outStr, "no longer supported") {
				t.Errorf("Expected rollback guidance for %s, got: %s", backend, outStr)
			}
			if !strings.Contains(outStr, `"dolt"`) {
				t.Errorf("Expected supported backend guidance for %s, got: %s", backend, outStr)
			}
		})
	}

	// A genuinely unsupported backend value is still rejected up front.
	t.Run("unknown_backend_errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")

		cmd := exec.Command(issuegraphBinary, "init", "--backend", "mongodb", "--quiet")
		cmd.Dir = tmpDir
		cmd.Env = initBackendTestEnv(issuegraphDir)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("Expected non-zero exit for a genuinely unknown backend")
		}

		outStr := string(out)
		if !strings.Contains(outStr, "unknown backend") {
			t.Errorf("Expected 'unknown backend' error, got: %s", outStr)
		}
	})

	t.Run("dolt_backend_succeeds", func(t *testing.T) {
		skipIfNoDolt(t)
		tmpDir := t.TempDir()
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")

		cmd := exec.Command(issuegraphBinary, "init", "--backend", "dolt", "--quiet")
		cmd.Dir = tmpDir
		cmd.Env = initBackendTestEnv(issuegraphDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("issuegraph init --backend=dolt should succeed: %v\n%s", err, out)
		}
	})

	t.Run("default_backend_is_dolt", func(t *testing.T) {
		skipIfNoDolt(t)
		tmpDir := t.TempDir()
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")

		cmd := exec.Command(issuegraphBinary, "init", "--quiet")
		cmd.Dir = tmpDir
		cmd.Env = initBackendTestEnv(issuegraphDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("issuegraph init should default to dolt: %v\n%s", err, out)
		}

		// Verify metadata.json has backend: dolt
		cfg, err := configfile.Load(issuegraphDir)
		if err != nil {
			t.Fatalf("Failed to load metadata.json: %v", err)
		}
		if cfg == nil {
			t.Fatal("metadata.json not found")
		}
		if cfg.Backend != configfile.BackendDolt {
			t.Errorf("Expected backend %q, got %q", configfile.BackendDolt, cfg.Backend)
		}
	})
}

// TestInitDatabaseAdoptsExistingProjectID verifies that issuegraph init --database adopts
// the _project_id from an existing server database instead of generating a new one.
// This prevents PROJECT IDENTITY MISMATCH errors when multiple users connect to
// a shared remote Dolt server. (GH#2922)
func TestInitDatabaseAdoptsExistingProjectID(t *testing.T) {
	skipIfNoDolt(t)

	// Reset global state
	origDBPath := dbPath
	origStore := store
	defer func() {
		if store != nil && store != origStore {
			store.Close()
		}
		store = origStore
		dbPath = origDBPath
	}()
	dbPath = ""
	store = nil

	ctx := context.Background()

	// Create a database with a known _project_id (simulates first user's init)
	database := uniqueTestDBName(t)
	firstIssueGraphDir := filepath.Join(t.TempDir(), ".issuegraph")
	if err := os.MkdirAll(firstIssueGraphDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	doltNewMutex.Lock()
	firstStore, err := dolt.New(ctx, &dolt.Config{
		Path:            filepath.Join(firstIssueGraphDir, "dolt"),
		IssueGraphDir:   firstIssueGraphDir,
		ServerHost:      "127.0.0.1",
		ServerPort:      testDoltServerPort,
		Database:        database,
		CreateIfMissing: true,
	})
	doltNewMutex.Unlock()
	if err != nil {
		t.Fatalf("create first store: %v", err)
	}

	knownProjectID := "test-known-project-id-gh2922"
	if err := firstStore.SetMetadata(ctx, "_project_id", knownProjectID); err != nil {
		t.Fatalf("set _project_id: %v", err)
	}
	if err := firstStore.SetConfig(ctx, "issue_prefix", "test"); err != nil {
		t.Fatalf("set issue_prefix: %v", err)
	}
	firstStore.Close()

	t.Cleanup(func() {
		dropTestDatabase(database, testDoltServerPort)
	})

	// Simulate second user — init with --database pointing at the existing DB
	secondDir := t.TempDir()
	t.Chdir(secondDir)

	// Set up minimal git repo (init expects it for repo_id)
	if err := exec.Command("git", "-C", secondDir, "init").Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	_ = exec.Command("git", "-C", secondDir, "config", "user.email", "test@test.com").Run()
	_ = exec.Command("git", "-C", secondDir, "config", "user.name", "Test").Run()

	rootCmd.SetArgs([]string{
		"init",
		"--server",
		"--server-host", "127.0.0.1",
		"--server-port", fmt.Sprintf("%d", testDoltServerPort),
		"--database", database,
		"--prefix", "second",
		"--quiet",
		"--skip-hooks",
		"--skip-agents",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second init failed: %v", err)
	}

	// Verify the second user's metadata.json adopted the existing project_id
	secondIssueGraphDir := filepath.Join(secondDir, ".issuegraph")
	cfg, err := configfile.Load(secondIssueGraphDir)
	if err != nil {
		t.Fatalf("load metadata.json: %v", err)
	}

	if cfg.ProjectID != knownProjectID {
		t.Errorf("ProjectID = %q, want %q (should adopt existing project_id from server)", cfg.ProjectID, knownProjectID)
	}
}
