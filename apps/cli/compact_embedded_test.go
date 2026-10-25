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

// issuegraphCompact runs "issuegraph compact" with the given args and returns stdout.
func issuegraphCompact(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"compact"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph compact %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphCompactFail runs "issuegraph compact" expecting failure.
func issuegraphCompactFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"compact"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph compact %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedCompact(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	// ===== Dry Run =====

	t.Run("compact_dry_run", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "cd")
		issuegraphCreate(t, issuegraph, dir, "Compact dry-run issue", "--type", "task")

		out := issuegraphCompact(t, issuegraph, dir, "--dry-run")
		if !strings.Contains(out, "DRY RUN") && !strings.Contains(out, "Nothing to compact") && !strings.Contains(out, "nothing to compact") {
			t.Errorf("expected dry-run or nothing-to-compact output: %s", out)
		}
	})

	// ===== Nothing to Compact (1 commit) =====

	t.Run("compact_nothing", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "cn")
		out := issuegraphCompact(t, issuegraph, dir, "--force")
		if !strings.Contains(out, "Nothing to compact") && !strings.Contains(out, "nothing to compact") && !strings.Contains(out, "Only") {
			t.Errorf("expected nothing-to-compact message: %s", out)
		}
	})

	// ===== No --force Errors =====

	t.Run("compact_no_force", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "cf")
		// Create issues to generate commits
		issuegraphCreate(t, issuegraph, dir, "Compact no-force 1", "--type", "task")
		issuegraphCreate(t, issuegraph, dir, "Compact no-force 2", "--type", "task")

		// With --days 0, all commits are "old"
		// May fail with "use --force" hint or succeed with "nothing to compact"
		cmd := exec.Command(issuegraph, "compact", "--days", "0")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		out, err := cmd.CombinedOutput()
		if err == nil {
			// Succeeded — either nothing to compact or only 1 old commit
			_ = out
		} else {
			// Should contain --force hint
			if !strings.Contains(string(out), "--force") {
				t.Errorf("expected --force hint in error: %s", out)
			}
		}
	})

	// ===== Force with --days 0 =====

	t.Run("compact_force", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "cx")
		// Create issues + config changes to build commit history
		issuegraphCreate(t, issuegraph, dir, "Compact force 1", "--type", "task")
		cmd := exec.Command(issuegraph, "config", "set", "compact.test1", "v1")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		cmd.CombinedOutput()
		cmd = exec.Command(issuegraph, "dolt", "commit", "-m", "config commit 1")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		cmd.CombinedOutput()
		issuegraphCreate(t, issuegraph, dir, "Compact force 2", "--type", "task")

		// Try compacting with --days 0
		out := issuegraphCompact(t, issuegraph, dir, "--force", "--days", "0")
		// Either compacts or reports nothing to compact — both OK
		if !strings.Contains(out, "Compacted") && !strings.Contains(out, "Nothing to compact") && !strings.Contains(out, "nothing to compact") && !strings.Contains(out, "Only") {
			t.Errorf("expected compact result: %s", out)
		}
	})

	// ===== --days Flag =====

	t.Run("compact_days_flag", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dy")
		issuegraphCreate(t, issuegraph, dir, "Compact days issue", "--type", "task")

		out := issuegraphCompact(t, issuegraph, dir, "--dry-run", "--days", "7")
		if !strings.Contains(out, "7 days") && !strings.Contains(out, "Nothing to compact") && !strings.Contains(out, "nothing to compact") {
			t.Errorf("expected days reference in output: %s", out)
		}
	})

	// ===== JSON Output =====

	t.Run("compact_dry_run_json", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "cj")
		issuegraphCreate(t, issuegraph, dir, "Compact JSON issue", "--type", "task")

		cmd := exec.Command(issuegraph, "--json", "compact", "--dry-run")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph --json compact --dry-run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		// Should produce some output without crashing
		_ = stdout.String()
	})
}

// TestEmbeddedCompactConcurrent exercises compact --dry-run concurrently.
func TestEmbeddedCompactConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "cc")

	issuegraphCreate(t, issuegraph, dir, "Compact concurrent issue", "--type", "task")

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
			cmd := exec.Command(issuegraph, "compact", "--dry-run")
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("compact --dry-run (worker %d): %v\n%s", worker, err, out)
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
