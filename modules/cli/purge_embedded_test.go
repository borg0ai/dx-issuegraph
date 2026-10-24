//go:build cgo

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// issuegraphPurge runs "issuegraph purge" with the given args and returns stdout.
func issuegraphPurge(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"purge"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph purge %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphPurgeFail runs "issuegraph purge" expecting failure.
func issuegraphPurgeFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"purge"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph purge %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

// createAndCloseEphemeral creates an ephemeral issue and closes it.
func createAndCloseEphemeral(t *testing.T, issuegraph, dir, title string) string {
	t.Helper()
	issue := issuegraphCreate(t, issuegraph, dir, title, "--ephemeral")
	cmd := exec.Command(issuegraph, "close", issue.ID)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("close %s failed: %v\n%s", issue.ID, err, out)
	}
	return issue.ID
}

func TestEmbeddedPurge(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	// ===== Nothing to Purge =====

	t.Run("purge_nothing", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pn")
		// No ephemeral issues — preview should show nothing
		cmd := exec.Command(issuegraph, "purge")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		out, _ := cmd.CombinedOutput()
		_ = out // Should not crash, regardless of exit code
	})

	// ===== Preview (no flags) =====

	t.Run("purge_preview", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pp")
		createAndCloseEphemeral(t, issuegraph, dir, "Purge preview 1")
		createAndCloseEphemeral(t, issuegraph, dir, "Purge preview 2")

		// Without --force, should show preview and fail
		issuegraphPurgeFail(t, issuegraph, dir)
	})

	// ===== Dry Run =====

	t.Run("purge_dry_run", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pd")
		createAndCloseEphemeral(t, issuegraph, dir, "Purge dry-run 1")

		out := issuegraphPurge(t, issuegraph, dir, "--dry-run")
		if len(strings.TrimSpace(out)) == 0 {
			t.Error("expected non-empty dry-run output")
		}
	})

	// ===== Force =====

	t.Run("purge_force", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pf")
		createAndCloseEphemeral(t, issuegraph, dir, "Purge force 1")
		createAndCloseEphemeral(t, issuegraph, dir, "Purge force 2")

		out := issuegraphPurge(t, issuegraph, dir, "--force")
		if !strings.Contains(out, "Purged") {
			t.Errorf("expected 'Purged' in output: %s", out)
		}
	})

	// ===== Older Than =====

	t.Run("purge_older_than", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "po")
		createAndCloseEphemeral(t, issuegraph, dir, "Purge older-than 1")

		// --older-than 1d means closed more than 1 day ago
		out := issuegraphPurge(t, issuegraph, dir, "--older-than", "1d", "--force")
		_ = out // Should succeed (may find 0 matches since just closed)
	})

	// ===== Pattern =====

	t.Run("purge_pattern", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pt")
		createAndCloseEphemeral(t, issuegraph, dir, "Purge pattern test")

		// Pattern matching — use prefix wildcard
		out := issuegraphPurge(t, issuegraph, dir, "--pattern", "pt-*", "--force")
		_ = out // Should succeed or find no matches
	})
}

// TestEmbeddedPurgeConcurrent exercises purge --dry-run concurrently.
func TestEmbeddedPurgeConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "px")

	createAndCloseEphemeral(t, issuegraph, dir, "Purge concurrent 1")
	createAndCloseEphemeral(t, issuegraph, dir, "Purge concurrent 2")

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
			cmd := exec.Command(issuegraph, "purge", "--dry-run")
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("purge --dry-run (worker %d): %v\n%s", worker, err, out)
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
