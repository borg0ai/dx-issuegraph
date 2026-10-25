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

// issuegraphDefer runs "issuegraph defer" with the given args and returns stdout.
func issuegraphDefer(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"defer"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph defer %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphUndefer runs "issuegraph undefer" with the given args and returns stdout.
func issuegraphUndefer(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"undefer"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph undefer %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// getIssueStatus returns the status of an issue via issuegraph show --json.
func getIssueStatus(t *testing.T, issuegraph, dir, id string) string {
	t.Helper()
	cmd := exec.Command(issuegraph, "show", id, "--json")
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph show %s --json failed: %v\nstdout:\n%s\nstderr:\n%s", id, err, stdout.String(), stderr.String())
	}
	s := strings.TrimSpace(stdout.String())
	// show --json may return an array or object
	start := strings.IndexAny(s, "[{")
	if start < 0 {
		t.Fatalf("no JSON in show output: %s", s)
	}
	if s[start] == '[' {
		var arr []map[string]interface{}
		if err := json.Unmarshal([]byte(s[start:]), &arr); err != nil {
			t.Fatalf("parse show JSON array: %v\n%s", err, s)
		}
		if len(arr) == 0 {
			t.Fatalf("empty JSON array in show output")
		}
		status, _ := arr[0]["status"].(string)
		return status
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
		t.Fatalf("parse show JSON: %v\n%s", err, s)
	}
	status, _ := m["status"].(string)
	return status
}

func TestEmbeddedDefer(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "df")

	// ===== Single Issue =====

	t.Run("defer_single", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Defer single test", "--type", "task")
		out := issuegraphDefer(t, issuegraph, dir, issue.ID)
		if !strings.Contains(out, "Deferred") {
			t.Errorf("expected 'Deferred' in output: %s", out)
		}
		status := getIssueStatus(t, issuegraph, dir, issue.ID)
		if status != "deferred" {
			t.Errorf("expected status=deferred, got %q", status)
		}
	})

	// ===== Multiple Issues =====

	t.Run("defer_multiple", func(t *testing.T) {
		issue1 := issuegraphCreate(t, issuegraph, dir, "Defer multi 1", "--type", "task")
		issue2 := issuegraphCreate(t, issuegraph, dir, "Defer multi 2", "--type", "task")
		out := issuegraphDefer(t, issuegraph, dir, issue1.ID, issue2.ID)
		if !strings.Contains(out, issue1.ID) || !strings.Contains(out, issue2.ID) {
			t.Errorf("expected both IDs in output: %s", out)
		}
		for _, id := range []string{issue1.ID, issue2.ID} {
			status := getIssueStatus(t, issuegraph, dir, id)
			if status != "deferred" {
				t.Errorf("expected %s status=deferred, got %q", id, status)
			}
		}
	})

	// ===== With --until =====

	t.Run("defer_until", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Defer until test", "--type", "task")
		out := issuegraphDefer(t, issuegraph, dir, issue.ID, "--until", "+1h")
		if !strings.Contains(out, "Deferred") {
			t.Errorf("expected 'Deferred' in output: %s", out)
		}
		status := getIssueStatus(t, issuegraph, dir, issue.ID)
		if status != "deferred" {
			t.Errorf("expected status=deferred, got %q", status)
		}
	})

	// ===== Already Deferred =====

	t.Run("defer_already_deferred", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Defer idempotent", "--type", "task")
		issuegraphDefer(t, issuegraph, dir, issue.ID)
		// Defer again — should succeed
		out := issuegraphDefer(t, issuegraph, dir, issue.ID)
		if !strings.Contains(out, "Deferred") {
			t.Errorf("expected 'Deferred' on second defer: %s", out)
		}
	})
}

// TestEmbeddedDeferConcurrent exercises defer operations concurrently.
func TestEmbeddedDeferConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dx")

	// Pre-create issues
	var issueIDs []string
	for i := 0; i < 8; i++ {
		issue := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("defer-concurrent-%d", i), "--type", "task")
		issueIDs = append(issueIDs, issue.ID)
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
			id := issueIDs[worker]

			cmd := exec.Command(issuegraph, "defer", id)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("defer %s (worker %d): %v\n%s", id, worker, err, out)
			}
			results[worker] = r
		}(w)
	}
	wg.Wait()

	var successes int
	for _, r := range results {
		if r.err != nil {
			if !strings.Contains(r.err.Error(), "one writer at a time") {
				t.Errorf("worker %d failed: %v", r.worker, r.err)
			}
			continue
		}
		successes++
	}
	if successes == 0 {
		t.Fatal("all workers failed; expected at least 1 success")
	}
	t.Logf("%d/%d workers succeeded (flock contention expected)", successes, numWorkers)

	// Verify only successful workers' issues are deferred
	for _, r := range results {
		if r.err != nil {
			continue
		}
		id := issueIDs[r.worker]
		status := getIssueStatus(t, issuegraph, dir, id)
		if status != "deferred" {
			t.Errorf("issue %d (%s): expected status=deferred, got %q", r.worker, id, status)
		}
	}
}
