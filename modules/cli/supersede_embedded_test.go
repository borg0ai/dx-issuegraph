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

// issuegraphSupersede runs "issuegraph supersede" with the given args and returns raw stdout.
func issuegraphSupersede(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"supersede"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph supersede %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphSupersedeFail runs "issuegraph supersede" expecting failure.
func issuegraphSupersedeFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"supersede"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph supersede %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedSupersede(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "ss")

	// ===== Mark as superseded =====

	t.Run("mark_superseded", func(t *testing.T) {
		oldIssue := issuegraphCreate(t, issuegraph, dir, "Old spec v1", "--type", "task")
		newIssue := issuegraphCreate(t, issuegraph, dir, "New spec v2", "--type", "task")
		out := issuegraphSupersede(t, issuegraph, dir, oldIssue.ID, "--with", newIssue.ID)
		if !strings.Contains(out, "superseded") {
			t.Errorf("expected 'superseded' in output: %s", out)
		}
	})

	// ===== Verify closure =====

	t.Run("superseded_is_closed", func(t *testing.T) {
		oldIssue := issuegraphCreate(t, issuegraph, dir, "Closed old", "--type", "task")
		newIssue := issuegraphCreate(t, issuegraph, dir, "Closed new", "--type", "task")
		issuegraphSupersede(t, issuegraph, dir, oldIssue.ID, "--with", newIssue.ID)

		s := openStore(t, issuegraphDir, "ss")
		issue, err := s.GetIssue(t.Context(), oldIssue.ID)
		if err != nil {
			t.Fatalf("GetIssue: %v", err)
		}
		if issue.Status != "closed" {
			t.Errorf("expected status=closed, got %s", issue.Status)
		}
		if issue.ClosedAt == nil {
			t.Error("expected supersede close to set closed_at")
		}
	})

	// ===== Creates supersedes link =====

	t.Run("creates_supersedes_link", func(t *testing.T) {
		oldIssue := issuegraphCreate(t, issuegraph, dir, "Link old", "--type", "task")
		newIssue := issuegraphCreate(t, issuegraph, dir, "Link new", "--type", "task")
		issuegraphSupersede(t, issuegraph, dir, oldIssue.ID, "--with", newIssue.ID)

		out := issuegraphDep(t, issuegraph, dir, "list", oldIssue.ID)
		if !strings.Contains(out, newIssue.ID) {
			t.Errorf("expected new issue in dep list: %s", out)
		}
	})

	// ===== JSON output =====

	t.Run("json_output", func(t *testing.T) {
		oldIssue := issuegraphCreate(t, issuegraph, dir, "JSON old", "--type", "task")
		newIssue := issuegraphCreate(t, issuegraph, dir, "JSON new", "--type", "task")
		fullArgs := []string{"supersede", oldIssue.ID, "--with", newIssue.ID, "--json"}
		cmd := exec.Command(issuegraph, fullArgs...)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("supersede --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		s := strings.TrimSpace(stdout.String())
		start := strings.Index(s, "{")
		if start < 0 {
			t.Fatalf("no JSON: %s", s)
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if m["superseded"] != oldIssue.ID {
			t.Errorf("expected superseded=%s, got %v", oldIssue.ID, m["superseded"])
		}
		if m["replacement"] != newIssue.ID {
			t.Errorf("expected replacement=%s, got %v", newIssue.ID, m["replacement"])
		}
	})

	// ===== Error: same ID =====

	t.Run("error_same_id", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Same ID", "--type", "task")
		issuegraphSupersedeFail(t, issuegraph, dir, issue.ID, "--with", issue.ID)
	})

	// ===== Error: nonexistent replacement =====

	t.Run("error_nonexistent_replacement", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "No replacement", "--type", "task")
		issuegraphSupersedeFail(t, issuegraph, dir, issue.ID, "--with", "ss-nonexistent999")
	})
}

// TestEmbeddedSupersedeConcurrent exercises supersede operations concurrently.
func TestEmbeddedSupersedeConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ssc")

	newIssue := issuegraphCreate(t, issuegraph, dir, "Concurrent replacement", "--type", "task")
	var oldIDs []string
	for i := 0; i < 8; i++ {
		old := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("concurrent-old-%d", i), "--type", "task")
		oldIDs = append(oldIDs, old.ID)
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

			args := []string{"supersede", oldIDs[worker], "--with", newIssue.ID}
			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("worker %d: %v\n%s", worker, err, out)
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
