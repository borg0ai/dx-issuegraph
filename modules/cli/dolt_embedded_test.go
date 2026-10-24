//go:build cgo

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// issuegraphDolt runs "issuegraph dolt" with the given args and returns stdout.
func issuegraphDolt(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"dolt"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph dolt %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphDoltFail runs "issuegraph dolt" expecting failure and returns stderr+stdout.
func issuegraphDoltFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"dolt"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph dolt %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedDolt(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "td")

	// ===== Server-only commands must fail in embedded mode =====

	// status and show are intentionally NOT in this list: they report
	// embedded-mode state rather than erroring out. See the
	// embedded_status/embedded_show subtests below.
	serverOnlyCmds := []struct {
		name string
		args []string
	}{
		{"start", []string{"start"}},
		{"stop", []string{"stop"}},
		{"test", []string{"test"}},
		{"set", []string{"set", "host", "127.0.0.1"}},
		{"killall", []string{"killall"}},
		{"clean-databases", []string{"clean-databases"}},
	}

	for _, tc := range serverOnlyCmds {
		t.Run("blocked_"+tc.name, func(t *testing.T) {
			out := issuegraphDoltFail(t, issuegraph, dir, tc.args...)
			if !strings.Contains(out, "not supported in embedded mode") {
				t.Errorf("expected 'not supported in embedded mode' in output for dolt %s: %s", tc.name, out)
			}
		})
	}

	// ===== Embedded-mode inspection commands succeed with embedded-mode output =====

	t.Run("embedded_status", func(t *testing.T) {
		out := issuegraphDolt(t, issuegraph, dir, "status")
		if !strings.Contains(out, "embedded") {
			t.Errorf("expected embedded-mode status output to mention 'embedded': %s", out)
		}
	})

	t.Run("embedded_status_json", func(t *testing.T) {
		out := issuegraphDolt(t, issuegraph, dir, "status", "--json")
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("status --json returned invalid JSON: %v\n%s", err, out)
		}
		if result["mode"] != "embedded" {
			t.Errorf("mode = %v, want embedded", result["mode"])
		}
		if result["server_running"] != false {
			t.Errorf("server_running = %v, want false", result["server_running"])
		}
		if _, ok := result["running"]; ok {
			t.Errorf("running should be omitted for embedded mode; use server_running instead: %v", result["running"])
		}
	})

	t.Run("embedded_show", func(t *testing.T) {
		out := issuegraphDolt(t, issuegraph, dir, "show")
		if !strings.Contains(out, "Mode:") || !strings.Contains(out, "embedded") {
			t.Errorf("expected embedded-mode show output to contain 'Mode:' and 'embedded': %s", out)
		}
	})

	// ===== Working commands =====

	// Create an issue so there are pending changes to commit
	issuegraphCreate(t, issuegraph, dir, "Dolt commit test issue", "--type", "task")

	t.Run("commit", func(t *testing.T) {
		out := issuegraphDolt(t, issuegraph, dir, "commit")
		// Should succeed — either commits or reports nothing to commit
		_ = out
	})

	t.Run("commit_with_message", func(t *testing.T) {
		issuegraphCreate(t, issuegraph, dir, "Another issue for commit", "--type", "task")
		out := issuegraphDolt(t, issuegraph, dir, "commit", "-m", "test commit message")
		_ = out
	})

	t.Run("commit_clean_reports_nothing", func(t *testing.T) {
		// issuegraph init leaves the working set clean (bootstrap's own commit
		// tolerates nothing-to-commit, GH#3886). A second, truly no-op
		// "issuegraph dolt commit" here must print "Nothing to commit.", not
		// "Committed." — this is the case EmbeddedDoltStore.Commit's new
		// nothing-to-commit tolerance made possible to reach with a nil
		// error.
		cleanDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "tdcln")
		out := issuegraphDolt(t, issuegraph, cleanDir, "commit")
		if !strings.Contains(out, "Nothing to commit.") {
			t.Errorf("expected 'Nothing to commit.' on a clean working set, got: %s", out)
		}
	})

	// ===== Remote management =====

	t.Run("remote_list_empty", func(t *testing.T) {
		out := issuegraphDolt(t, issuegraph, dir, "remote", "list")
		_ = out // Should succeed even with no remotes
	})

	t.Run("remote_add_and_list", func(t *testing.T) {
		remoteDir := t.TempDir()
		issuegraphDolt(t, issuegraph, dir, "remote", "add", "test-remote", "file://"+remoteDir)
		out := issuegraphDolt(t, issuegraph, dir, "remote", "list")
		if !strings.Contains(out, "test-remote") {
			t.Errorf("expected 'test-remote' in remote list: %s", out)
		}
	})

	t.Run("remote_remove", func(t *testing.T) {
		issuegraphDolt(t, issuegraph, dir, "remote", "remove", "test-remote")
		out := issuegraphDolt(t, issuegraph, dir, "remote", "list")
		if strings.Contains(out, "test-remote") {
			t.Errorf("expected 'test-remote' to be removed: %s", out)
		}
	})

	// ===== Push/Pull with file remote =====

	t.Run("push_to_file_remote", func(t *testing.T) {
		ppDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pp")

		remoteDir := t.TempDir()
		issuegraphDolt(t, issuegraph, ppDir, "remote", "add", "origin", "file://"+remoteDir)

		// Create an issue and commit
		issuegraphCreate(t, issuegraph, ppDir, "Push roundtrip issue", "--type", "task")
		issuegraphDolt(t, issuegraph, ppDir, "commit", "-m", "roundtrip commit")

		// Push to file remote
		issuegraphDolt(t, issuegraph, ppDir, "push")
	})

	t.Run("push_force", func(t *testing.T) {
		pfDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pf")
		remoteDir := t.TempDir()
		issuegraphDolt(t, issuegraph, pfDir, "remote", "add", "origin", "file://"+remoteDir)

		issuegraphCreate(t, issuegraph, pfDir, "Force push issue", "--type", "task")
		issuegraphDolt(t, issuegraph, pfDir, "commit", "-m", "force push commit")
		issuegraphDolt(t, issuegraph, pfDir, "push", "--force")
	})

	t.Run("push_no_remote", func(t *testing.T) {
		// Push without a remote should fail gracefully, not panic
		cmd := exec.Command(issuegraph, "dolt", "push")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Log("push without remote succeeded unexpectedly")
		}
		_ = out
	})

	t.Run("pull_no_remote", func(t *testing.T) {
		cmd := exec.Command(issuegraph, "dolt", "pull")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Log("pull without remote succeeded unexpectedly")
		}
		_ = out
	})

	// ===== Push/Pull between two repos via shared file remote =====

	t.Run("push_then_push_more", func(t *testing.T) {
		// Push twice to verify incremental pushes work
		ppDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pm")
		remoteDir := t.TempDir()
		issuegraphDolt(t, issuegraph, ppDir, "remote", "add", "origin", "file://"+remoteDir)

		// First push
		issuegraphCreate(t, issuegraph, ppDir, "First push issue", "--type", "task")
		issuegraphDolt(t, issuegraph, ppDir, "commit", "-m", "first commit")
		issuegraphDolt(t, issuegraph, ppDir, "push")

		// Second push
		issuegraphCreate(t, issuegraph, ppDir, "Second push issue", "--type", "task")
		issuegraphDolt(t, issuegraph, ppDir, "commit", "-m", "second commit")
		issuegraphDolt(t, issuegraph, ppDir, "push")
	})
}

// TestEmbeddedDoltConcurrent exercises dolt operations concurrently.
func TestEmbeddedDoltConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dx")

	// Pre-create issues so commits have something to work with
	for i := 0; i < 4; i++ {
		issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("dolt-concurrent-%d", i), "--type", "task")
	}

	const numWorkers = 8

	type workerResult struct {
		worker int
		err    error
	}

	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			r := workerResult{worker: worker}

			var args []string
			expectFail := false
			switch worker % 3 {
			case 0:
				args = []string{"dolt", "commit"}
			case 1:
				// Status is an embedded-mode inspection command.
				args = []string{"dolt", "status"}
			case 2:
				// Server-only command should fail fast.
				args = []string{"dolt", "start"}
				expectFail = true
			}

			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()

			if expectFail {
				if err == nil {
					r.err = fmt.Errorf("expected dolt %s to fail in embedded mode", args[1])
				} else if !strings.Contains(string(out), "not supported in embedded mode") {
					r.err = fmt.Errorf("unexpected error from dolt %s: %s", args[1], out)
				}
			} else {
				if err != nil {
					r.err = fmt.Errorf("dolt %s (worker %d): %v\n%s", strings.Join(args[1:], " "), worker, err, out)
				}
			}

			results[worker] = r
		}(w)
	}
	wg.Wait()

	for _, r := range results {
		if r.err != nil && !strings.Contains(r.err.Error(), "one writer at a time") {
			t.Errorf("worker %d failed: %v", r.worker, r.err)
		}
	}
}

// TestAdminEmbeddedBlocked verifies that issuegraph admin subcommands are blocked in
// embedded mode. The guard was previously on adminCmd.PersistentPreRunE which
// fired before cmdCtx.ServerMode was populated, causing a false "embedded mode"
// error even in server-mode projects. The guard is now inside each subcommand's
// Run func, after store init. This test ensures embedded mode still correctly
// rejects admin commands.
func TestEmbeddedAdminBlocked(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ta")

	blockedCmds := []struct {
		name string
		args []string
	}{
		{"compact --dolt", []string{"admin", "compact", "--dolt"}},
		{"cleanup --dry-run", []string{"admin", "cleanup", "--dry-run"}},
		{"reset", []string{"admin", "reset"}},
	}

	for _, tc := range blockedCmds {
		tc := tc
		t.Run("blocked_"+tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command(issuegraph, tc.args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("issuegraph %s should fail in embedded mode but succeeded: %s", tc.name, out)
				return
			}
			if !strings.Contains(string(out), "not yet supported in embedded mode") {
				t.Errorf("issuegraph %s: expected 'not yet supported in embedded mode', got: %s", tc.name, out)
			}
		})
	}
}

// TestAdminEmbeddedCompactReadOnlyAllowed verifies read-only compact modes are
// not blocked by the embedded-mode admin guard.
func TestEmbeddedAdminCompactReadOnlyAllowed(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ta")

	readOnlyCmds := []struct {
		name string
		args []string
	}{
		{"compact --stats", []string{"admin", "compact", "--stats"}},
		{"compact --dolt --dry-run", []string{"admin", "compact", "--dolt", "--dry-run"}},
	}

	for _, tc := range readOnlyCmds {
		tc := tc
		t.Run("allowed_"+tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command(issuegraph, tc.args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("issuegraph %s should be allowed in embedded mode: %v\n%s", tc.name, err, out)
				return
			}
			if strings.Contains(string(out), "not yet supported in embedded mode") {
				t.Errorf("issuegraph %s hit embedded-mode guard unexpectedly: %s", tc.name, out)
			}
		})
	}
}
