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

// issuegraphDuplicate runs "issuegraph duplicate" with the given args and returns raw stdout.
func issuegraphDuplicate(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"duplicate"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph duplicate %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphDuplicateFail runs "issuegraph duplicate" expecting failure.
func issuegraphDuplicateFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"duplicate"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph duplicate %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedDuplicate(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "du")

	// ===== Mark as duplicate =====

	t.Run("mark_duplicate", func(t *testing.T) {
		canonical := issuegraphCreate(t, issuegraph, dir, "Canonical issue", "--type", "bug")
		dupe := issuegraphCreate(t, issuegraph, dir, "Duplicate issue", "--type", "bug")
		out := issuegraphDuplicate(t, issuegraph, dir, dupe.ID, "--of", canonical.ID)
		if !strings.Contains(out, "duplicate") {
			t.Errorf("expected 'duplicate' in output: %s", out)
		}
	})

	// ===== Verify closure =====

	t.Run("duplicate_is_closed", func(t *testing.T) {
		canonical := issuegraphCreate(t, issuegraph, dir, "Canon closed", "--type", "task")
		dupe := issuegraphCreate(t, issuegraph, dir, "Dupe closed", "--type", "task")
		issuegraphDuplicate(t, issuegraph, dir, dupe.ID, "--of", canonical.ID)

		s := openStore(t, issuegraphDir, "du")
		issue, err := s.GetIssue(t.Context(), dupe.ID)
		if err != nil {
			t.Fatalf("GetIssue: %v", err)
		}
		if issue.Status != "closed" {
			t.Errorf("expected status=closed, got %s", issue.Status)
		}
		if issue.ClosedAt == nil {
			t.Error("expected duplicate close to set closed_at")
		}
	})

	// ===== Creates dependency link =====

	t.Run("creates_dep_link", func(t *testing.T) {
		canonical := issuegraphCreate(t, issuegraph, dir, "Canon link", "--type", "task")
		dupe := issuegraphCreate(t, issuegraph, dir, "Dupe link", "--type", "task")
		issuegraphDuplicate(t, issuegraph, dir, dupe.ID, "--of", canonical.ID)

		// Check via dep list
		out := issuegraphDep(t, issuegraph, dir, "list", dupe.ID)
		if !strings.Contains(out, canonical.ID) {
			t.Errorf("expected canonical in dep list: %s", out)
		}
	})

	// ===== JSON output =====

	t.Run("json_output", func(t *testing.T) {
		canonical := issuegraphCreate(t, issuegraph, dir, "Canon JSON", "--type", "task")
		dupe := issuegraphCreate(t, issuegraph, dir, "Dupe JSON", "--type", "task")
		fullArgs := []string{"duplicate", dupe.ID, "--of", canonical.ID, "--json"}
		cmd := exec.Command(issuegraph, fullArgs...)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("duplicate --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
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
		if m["duplicate"] != dupe.ID {
			t.Errorf("expected duplicate=%s, got %v", dupe.ID, m["duplicate"])
		}
		if m["canonical"] != canonical.ID {
			t.Errorf("expected canonical=%s, got %v", canonical.ID, m["canonical"])
		}
	})

	// ===== Error: same ID =====

	t.Run("error_same_id", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Same ID", "--type", "task")
		issuegraphDuplicateFail(t, issuegraph, dir, issue.ID, "--of", issue.ID)
	})

	// ===== Error: nonexistent canonical =====

	t.Run("error_nonexistent_canonical", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "No canon", "--type", "task")
		issuegraphDuplicateFail(t, issuegraph, dir, issue.ID, "--of", "du-nonexistent999")
	})
}

// TestEmbeddedDuplicateConcurrent exercises duplicate operations concurrently.
func TestEmbeddedDuplicateConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "duc")

	canonical := issuegraphCreate(t, issuegraph, dir, "Concurrent canonical", "--type", "task")
	var dupeIDs []string
	for i := 0; i < 8; i++ {
		d := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("concurrent-dupe-%d", i), "--type", "task")
		dupeIDs = append(dupeIDs, d.ID)
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

			args := []string{"duplicate", dupeIDs[worker], "--of", canonical.ID}
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
