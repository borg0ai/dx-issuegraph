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

// issuegraphPromote runs "issuegraph promote" with the given args and returns stdout.
func issuegraphPromote(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"promote"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph promote %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphPromoteFail runs "issuegraph promote" expecting failure.
func issuegraphPromoteFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"promote"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph promote %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedPromoteCLI(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "pr")

	t.Run("promote_basic", func(t *testing.T) {
		// Create ephemeral issue
		issue := issuegraphCreate(t, issuegraph, dir, "CLI promote basic", "--ephemeral")

		// Verify it's ephemeral
		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if !got.Ephemeral {
			t.Skip("issue not ephemeral; cannot test promote")
		}

		out := issuegraphPromote(t, issuegraph, dir, issue.ID)
		if !strings.Contains(out, "Promoted") {
			t.Errorf("expected 'Promoted' in output: %s", out)
		}

		// Verify no longer ephemeral
		got = issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Ephemeral {
			t.Error("expected non-ephemeral after promote")
		}
	})

	t.Run("promote_with_reason", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "CLI promote reason", "--ephemeral")

		out := issuegraphPromote(t, issuegraph, dir, issue.ID, "--reason", "Important enough to persist")
		if !strings.Contains(out, "Promoted") {
			t.Errorf("expected 'Promoted' in output: %s", out)
		}
	})

	t.Run("promote_with_reason_short", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "CLI promote -r", "--ephemeral")

		out := issuegraphPromote(t, issuegraph, dir, issue.ID, "-r", "Short reason")
		if !strings.Contains(out, "Promoted") {
			t.Errorf("expected 'Promoted' in output: %s", out)
		}
	})

	t.Run("promote_json", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "CLI promote JSON", "--ephemeral")

		cmd := exec.Command(issuegraph, "promote", issue.ID, "--json")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph promote --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		s := strings.TrimSpace(stdout.String())
		start := strings.Index(s, "{")
		if start < 0 {
			t.Fatalf("no JSON object in output: %s", s)
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
			t.Fatalf("parse promote JSON: %v\n%s", err, s)
		}
		if m["id"] != issue.ID {
			t.Errorf("expected id=%s, got %v", issue.ID, m["id"])
		}
	})

	t.Run("promote_preserves_labels", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Promote labels", "--ephemeral", "--label", "keep-me")
		issuegraphPromote(t, issuegraph, dir, issue.ID)

		store := openStore(t, issuegraphDir, "pr")
		labels, err := store.GetLabels(t.Context(), issue.ID)
		if err != nil {
			t.Fatalf("GetLabels: %v", err)
		}
		found := false
		for _, l := range labels {
			if l == "keep-me" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected label 'keep-me' preserved after promote, got %v", labels)
		}
	})

	t.Run("promote_nonexistent_fails", func(t *testing.T) {
		issuegraphPromoteFail(t, issuegraph, dir, "pr-nonexistent999")
	})

	t.Run("promote_already_permanent_fails", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Already permanent", "--type", "task")
		issuegraphPromoteFail(t, issuegraph, dir, issue.ID)
	})

	t.Run("promote_no_args_fails", func(t *testing.T) {
		cmd := exec.Command(issuegraph, "promote")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected promote with no args to fail, got: %s", out)
		}
	})
}

// TestEmbeddedPromoteCLIConcurrent exercises promote concurrently.
func TestEmbeddedPromoteCLIConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "px")

	const numWorkers = 6

	// Pre-create ephemeral issues
	var issueIDs []string
	for i := 0; i < numWorkers; i++ {
		issue := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("concurrent-promote-%d", i), "--ephemeral")
		issueIDs = append(issueIDs, issue.ID)
	}

	type workerResult struct {
		worker int
		out    string
		err    error
	}

	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			r := workerResult{worker: worker}

			cmd := exec.Command(issuegraph, "promote", issueIDs[worker])
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			r.out = string(out)
			if err != nil {
				r.err = fmt.Errorf("promote %s: %v", issueIDs[worker], err)
			}

			results[worker] = r
		}(w)
	}
	wg.Wait()

	// The driver handles concurrency internally — all workers should succeed.
	for _, r := range results {
		if r.err != nil {
			t.Errorf("worker %d failed: %v", r.worker, r.err)
		}
	}

	// Verify all issues are permanent.
	for i, id := range issueIDs {
		got := issuegraphShow(t, issuegraph, dir, id)
		if got.Ephemeral {
			t.Errorf("expected %s (worker %d) to be permanent after promote", id, i)
		}
	}
}
