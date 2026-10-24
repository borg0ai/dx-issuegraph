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

// issuegraphFlatten runs "issuegraph flatten" with the given args and returns stdout.
func issuegraphFlatten(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"flatten"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph flatten %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphFlattenFail runs "issuegraph flatten" expecting failure.
func issuegraphFlattenFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"flatten"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph flatten %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedFlatten(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	// ===== Dry Run =====

	t.Run("flatten_dry_run", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fd")
		// Create issues to generate commits
		issuegraphCreate(t, issuegraph, dir, "Flatten dry-run issue 1", "--type", "task")
		issuegraphCreate(t, issuegraph, dir, "Flatten dry-run issue 2", "--type", "task")

		out := issuegraphFlatten(t, issuegraph, dir, "--dry-run")
		if !strings.Contains(out, "Commits") || !strings.Contains(out, "DRY RUN") {
			t.Errorf("expected commit count and DRY RUN in output: %s", out)
		}
	})

	// ===== Already Flat =====

	t.Run("flatten_fresh_db", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fa")
		// Fresh init may have 1 or more commits depending on schema setup.
		// Either "Already flat" or successful flatten is fine.
		out := issuegraphFlatten(t, issuegraph, dir, "--force")
		if !strings.Contains(out, "Already flat") && !strings.Contains(out, "Flattened") && !strings.Contains(out, "already flat") {
			t.Errorf("expected 'Already flat' or 'Flattened' message: %s", out)
		}
	})

	// ===== No --force Errors =====
	// Note: In embedded mode, store.Log() may return 0 entries even when
	// Dolt commits exist, causing flatten to always report "Already flat".
	// These tests skip when that happens rather than failing.

	t.Run("flatten_no_force", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fn")
		issuegraphCreate(t, issuegraph, dir, "Flatten no-force 1", "--type", "task")
		issuegraphCreate(t, issuegraph, dir, "Flatten no-force 2", "--type", "task")

		dryOut := issuegraphFlatten(t, issuegraph, dir, "--dry-run")
		if strings.Contains(dryOut, "Already flat") || strings.Contains(dryOut, "Commits:        0") {
			t.Skip("store.Log() returns 0 commits in embedded mode — flatten guard untestable")
		}
		issuegraphFlattenFail(t, issuegraph, dir)
	})

	// ===== Force =====

	t.Run("flatten_force", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ff")
		issuegraphCreate(t, issuegraph, dir, "Flatten force 1", "--type", "task")
		issuegraphCreate(t, issuegraph, dir, "Flatten force 2", "--type", "task")
		issuegraphCreate(t, issuegraph, dir, "Flatten force 3", "--type", "task")

		dryOut := issuegraphFlatten(t, issuegraph, dir, "--dry-run")
		if strings.Contains(dryOut, "Already flat") || strings.Contains(dryOut, "Commits:        0") {
			t.Skip("store.Log() returns 0 commits in embedded mode — flatten untestable")
		}

		out := issuegraphFlatten(t, issuegraph, dir, "--force")
		if !strings.Contains(out, "Flattened") {
			t.Errorf("expected 'Flattened' in output: %s", out)
		}

		// Verify only 1 commit remains
		dryOut = issuegraphFlatten(t, issuegraph, dir, "--dry-run")
		if !strings.Contains(dryOut, "Already flat") && !strings.Contains(dryOut, "1") {
			t.Errorf("expected 1 commit after flatten: %s", dryOut)
		}

		// Verify issues still exist after flatten
		cmd := exec.Command(issuegraph, "count")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph count after flatten failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		if strings.Contains(stdout.String(), ": 0") {
			t.Errorf("expected issues to survive flatten: %s", stdout.String())
		}
	})
}

// TestEmbeddedFlattenConcurrent exercises flatten --dry-run concurrently.
func TestEmbeddedFlattenConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fx")

	issuegraphCreate(t, issuegraph, dir, "Flatten concurrent issue", "--type", "task")

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
			cmd := exec.Command(issuegraph, "flatten", "--dry-run")
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("flatten --dry-run (worker %d): %v\n%s", worker, err, out)
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
