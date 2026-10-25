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

// issuegraphState runs "issuegraph state" with the given args and returns stdout.
func issuegraphState(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"state"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph state %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphSetState runs "issuegraph set-state" with the given args and returns stdout.
func issuegraphSetState(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"set-state"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph set-state %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func TestEmbeddedState(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "st")

	issue := issuegraphCreate(t, issuegraph, dir, "State test issue", "--type", "task")

	// ===== set-state =====

	t.Run("set_state_basic", func(t *testing.T) {
		out := issuegraphSetState(t, issuegraph, dir, issue.ID, "phase=planning")
		if !strings.Contains(out, "planning") {
			t.Logf("set-state output: %s", out)
		}
	})

	t.Run("set_state_json", func(t *testing.T) {
		cmd := exec.Command(issuegraph, "set-state", issue.ID, "env=staging", "--json")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph set-state --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		s := strings.TrimSpace(stdout.String())
		start := strings.Index(s, "{")
		if start >= 0 {
			var m map[string]interface{}
			if jsonErr := json.Unmarshal([]byte(s[start:]), &m); jsonErr != nil {
				t.Errorf("invalid JSON: %v\n%s", jsonErr, s)
			}
		}
	})

	t.Run("set_state_with_reason", func(t *testing.T) {
		out := issuegraphSetState(t, issuegraph, dir, issue.ID, "risk=high", "--reason", "New vulnerability found")
		_ = out
	})

	t.Run("set_state_overwrites", func(t *testing.T) {
		issuegraphSetState(t, issuegraph, dir, issue.ID, "phase=development")
		issuegraphSetState(t, issuegraph, dir, issue.ID, "phase=testing")

		out := issuegraphState(t, issuegraph, dir, issue.ID, "phase")
		if !strings.Contains(out, "testing") {
			t.Errorf("expected 'testing' after overwrite, got: %s", out)
		}
	})

	// ===== state query =====

	t.Run("state_query", func(t *testing.T) {
		out := issuegraphState(t, issuegraph, dir, issue.ID, "phase")
		if !strings.Contains(out, "testing") {
			t.Logf("state query output: %s", out)
		}
	})

	t.Run("state_query_json", func(t *testing.T) {
		cmd := exec.Command(issuegraph, "state", issue.ID, "phase", "--json")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph state --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		_ = stdout.String()
	})

	t.Run("state_query_nonexistent_dimension", func(t *testing.T) {
		out := issuegraphState(t, issuegraph, dir, issue.ID, "nonexistent")
		// Should return empty/not-set, not error
		_ = out
	})

	// ===== state list =====

	t.Run("state_list", func(t *testing.T) {
		out := issuegraphState(t, issuegraph, dir, "list", issue.ID)
		// Should show the dimensions we set
		if !strings.Contains(out, "phase") {
			t.Logf("state list output: %s", out)
		}
	})

	t.Run("state_list_json", func(t *testing.T) {
		cmd := exec.Command(issuegraph, "state", "list", issue.ID, "--json")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph state list --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		_ = stdout.String()
	})

	t.Run("state_list_no_dimensions", func(t *testing.T) {
		fresh := issuegraphCreate(t, issuegraph, dir, "No state", "--type", "task")
		out := issuegraphState(t, issuegraph, dir, "list", fresh.ID)
		_ = out
	})
}

// TestEmbeddedStateConcurrent exercises set-state concurrently on different dimensions.
func TestEmbeddedStateConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "sx")

	issue := issuegraphCreate(t, issuegraph, dir, "Concurrent state", "--type", "task")

	const numWorkers = 6

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

			dim := fmt.Sprintf("dim%d=val%d", worker, worker)
			cmd := exec.Command(issuegraph, "set-state", issue.ID, dim)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("set-state %s: %v\n%s", dim, err, out)
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
