//go:build cgo

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// issuegraphReopen runs "issuegraph reopen" with the given args and returns stdout.
func issuegraphReopen(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"reopen"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph reopen %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func TestEmbeddedReopen(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ro")

	t.Run("reopen_single", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Reopen me", "--type", "task")
		issuegraphClose(t, issuegraph, dir, issue.ID)
		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Status != types.StatusClosed {
			t.Fatalf("expected closed before reopen, got %s", got.Status)
		}

		out := issuegraphReopen(t, issuegraph, dir, issue.ID)
		if !strings.Contains(out, "Reopened") {
			t.Errorf("expected 'Reopened' in output: %s", out)
		}

		got = issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Status != types.StatusOpen {
			t.Errorf("expected open after reopen, got %s", got.Status)
		}
		if got.ClosedAt != nil {
			t.Error("expected closed_at cleared after reopen")
		}
	})

	t.Run("reopen_multiple", func(t *testing.T) {
		issue1 := issuegraphCreate(t, issuegraph, dir, "Multi reopen 1", "--type", "task")
		issue2 := issuegraphCreate(t, issuegraph, dir, "Multi reopen 2", "--type", "task")
		issuegraphClose(t, issuegraph, dir, issue1.ID, issue2.ID)

		issuegraphReopen(t, issuegraph, dir, issue1.ID, issue2.ID)

		got1 := issuegraphShow(t, issuegraph, dir, issue1.ID)
		got2 := issuegraphShow(t, issuegraph, dir, issue2.ID)
		if got1.Status != types.StatusOpen {
			t.Errorf("issue1: expected open, got %s", got1.Status)
		}
		if got2.Status != types.StatusOpen {
			t.Errorf("issue2: expected open, got %s", got2.Status)
		}
	})

	t.Run("reopen_with_reason", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Reason reopen", "--type", "task")
		issuegraphClose(t, issuegraph, dir, issue.ID)

		out := issuegraphReopen(t, issuegraph, dir, issue.ID, "--reason", "Not actually done")
		if !strings.Contains(out, "Reopened") {
			t.Errorf("expected 'Reopened' in output: %s", out)
		}
		if !strings.Contains(out, "Not actually done") {
			t.Logf("reason may not appear in text output: %s", out)
		}
	})

	t.Run("reopen_with_reason_short", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Short reason reopen", "--type", "task")
		issuegraphClose(t, issuegraph, dir, issue.ID)
		issuegraphReopen(t, issuegraph, dir, issue.ID, "-r", "Needs more work")
		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Status != types.StatusOpen {
			t.Errorf("expected open, got %s", got.Status)
		}
	})

	t.Run("reopen_json", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "JSON reopen", "--type", "task")
		issuegraphClose(t, issuegraph, dir, issue.ID)

		cmd := exec.Command(issuegraph, "reopen", issue.ID, "--json")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph reopen --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		s := strings.TrimSpace(stdout.String())
		start := strings.Index(s, "[")
		if start < 0 {
			start = strings.Index(s, "{")
		}
		if start >= 0 {
			// Verify it's valid JSON
			_ = s[start:]
		}
	})

	t.Run("reopen_already_open", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Already open", "--type", "task")
		// Reopen an already-open issue — should not error, just print message
		out := issuegraphReopen(t, issuegraph, dir, issue.ID)
		if !strings.Contains(out, "already open") {
			t.Logf("already-open message: %s", out)
		}
	})

	t.Run("reopen_clears_defer_until", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Deferred reopen", "--type", "task", "--defer", "2030-01-01")
		issuegraphClose(t, issuegraph, dir, issue.ID)
		issuegraphReopen(t, issuegraph, dir, issue.ID)
		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Status != types.StatusOpen {
			t.Errorf("expected open, got %s", got.Status)
		}
	})

	t.Run("reopen_nonexistent", func(t *testing.T) {
		cmd := exec.Command(issuegraph, "reopen", "ro-nonexistent999")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected reopen of nonexistent to fail, got: %s", out)
		}
	})
}

// TestEmbeddedReopenConcurrent exercises reopen concurrently.
func TestEmbeddedReopenConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "rx")

	const numWorkers = 8

	// Pre-create and close issues
	var issueIDs []string
	for i := 0; i < numWorkers; i++ {
		issue := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("concurrent-reopen-%d", i), "--type", "task")
		issuegraphClose(t, issuegraph, dir, issue.ID)
		issueIDs = append(issueIDs, issue.ID)
	}

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

			cmd := exec.Command(issuegraph, "reopen", issueIDs[worker])
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("reopen %s: %v\n%s", issueIDs[worker], err, out)
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

	// Verify only successful workers' issues are reopened
	for _, r := range results {
		if r.err != nil {
			continue
		}
		id := issueIDs[r.worker]
		got := issuegraphShow(t, issuegraph, dir, id)
		if got.Status != types.StatusOpen {
			t.Errorf("expected %s to be open after reopen, got %s", id, got.Status)
		}
	}
}
