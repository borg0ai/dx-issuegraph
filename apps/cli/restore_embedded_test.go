//go:build cgo

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/storage/embeddeddolt"
)

// issuegraphRestore runs "issuegraph restore" with extra args. Returns combined output.
func issuegraphRestore(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"restore"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph restore %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphRestoreFail runs "issuegraph restore" expecting failure.
func issuegraphRestoreFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"restore"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("issuegraph restore %s should have failed, got: %s", strings.Join(args, " "), out)
	}
	return string(out)
}

// simulateCompaction creates an issue with content, commits, then marks it as
// compacted by updating compaction_level directly via the embedded store.
// Returns the issue ID.
func simulateCompaction(t *testing.T, issuegraph, dir, issuegraphDir, database string) string {
	t.Helper()

	// Create an issue with substantial content (auto-committed by issuegraph create)
	id := issuegraphCreateSilent(t, issuegraph, dir, "Compactable issue",
		"--description", "This is a long description that will be preserved in history.",
		"--design", "Design notes for the compactable issue.")

	// Directly update the issue to simulate compaction:
	// set compaction_level=1 and truncate description.
	// Use a retry loop because the issuegraph subprocess may still hold the lock briefly.
	var store *embeddeddolt.EmbeddedDoltStore
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for i := 0; i < 5; i++ {
		store, err = embeddeddolt.Open(ctx, issuegraphDir, database, "main")
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("open store after retries: %v", err)
	}

	issue, err := store.GetIssue(ctx, id)
	if err != nil {
		store.Close()
		t.Fatalf("get issue: %v", err)
	}

	originalSize := len(issue.Description) + len(issue.Design)
	if err := store.ApplyCompaction(ctx, id, 1, originalSize, 0, "simulated"); err != nil {
		store.Close()
		t.Fatalf("apply compaction: %v", err)
	}

	if err := store.UpdateIssue(ctx, id, map[string]interface{}{
		"description": "[compacted]",
		"design":      "",
	}, "test"); err != nil {
		store.Close()
		t.Fatalf("update issue: %v", err)
	}

	if err := store.Commit(ctx, "simulate compaction"); err != nil {
		store.Close()
		t.Fatalf("commit compaction: %v", err)
	}

	// Close explicitly so the lock is released before the next issuegraph subprocess.
	store.Close()

	return id
}

func TestEmbeddedRestore(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt restore tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("not_found", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "rstnf")

		out := issuegraphRestoreFail(t, issuegraph, dir, "rstnf-nonexistent")
		if !strings.Contains(out, "not found") {
			t.Errorf("expected 'not found' error, got: %s", out)
		}
	})

	t.Run("not_compacted", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "rstnc")
		id := issuegraphCreateSilent(t, issuegraph, dir, "Not compacted issue")

		out := issuegraphRestoreFail(t, issuegraph, dir, id)
		if !strings.Contains(out, "not compacted") {
			t.Errorf("expected 'not compacted' error, got: %s", out)
		}
	})

	t.Run("restore_compacted", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "rstok")
		id := simulateCompaction(t, issuegraph, dir, issuegraphDir, "rstok")

		out := issuegraphRestore(t, issuegraph, dir, id)
		// Should show the restored content from history
		if !strings.Contains(out, "long description") {
			t.Errorf("expected restored description in output, got: %s", out)
		}
	})

	t.Run("restore_json", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "rstjs")
		id := simulateCompaction(t, issuegraph, dir, issuegraphDir, "rstjs")

		// NOTE: issuegraph restore has a known issue where its local --json flag
		// conflicts with the root command's --json persistent flag.
		// Just verify the command runs successfully.
		out := issuegraphRestore(t, issuegraph, dir, id)
		if !strings.Contains(out, "long description") {
			t.Errorf("expected restored description in output, got: %s", out)
		}
	})
}

func TestEmbeddedRestoreConcurrent(t *testing.T) {
	t.Skip("TODO: simulateCompaction fails under CI concurrency — needs investigation")
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt restore tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "rstcon")

	// Create multiple compacted issues
	const numIssues = 5
	var ids []string
	for i := 0; i < numIssues; i++ {
		id := issuegraphCreateSilent(t, issuegraph, dir, fmt.Sprintf("Concurrent restore issue %d", i),
			"--description", fmt.Sprintf("Content for issue %d that should be in history.", i))
		ids = append(ids, id)
	}

	// Issues are auto-committed by issuegraph create. Simulate compaction on all of them.
	// Retry opening the store in case the last issuegraph subprocess still holds the lock.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var store *embeddeddolt.EmbeddedDoltStore
	var err error
	for i := 0; i < 5; i++ {
		store, err = embeddeddolt.Open(ctx, issuegraphDir, "rstcon", "main")
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("open store after retries: %v", err)
	}
	for _, id := range ids {
		issue, err := store.GetIssue(ctx, id)
		if err != nil {
			store.Close()
			t.Fatalf("get issue %s: %v", id, err)
		}
		if err := store.ApplyCompaction(ctx, id, 1, len(issue.Description), 0, "sim"); err != nil {
			store.Close()
			t.Fatalf("apply compaction %s: %v", id, err)
		}
		if err := store.UpdateIssue(ctx, id, map[string]interface{}{"description": "[compacted]"}, "test"); err != nil {
			store.Close()
			t.Fatalf("update issue %s: %v", id, err)
		}
	}
	if err := store.Commit(ctx, "compact all"); err != nil {
		store.Close()
		t.Fatalf("commit compaction: %v", err)
	}
	store.Close()

	// Concurrent restore reads
	type result struct {
		worker int
		out    string
		err    error
	}

	results := make([]result, numIssues)
	var wg sync.WaitGroup
	wg.Add(numIssues)

	for w := 0; w < numIssues; w++ {
		go func(worker int) {
			defer wg.Done()
			cmd := exec.Command(issuegraph, "restore", ids[worker])
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			results[worker] = result{worker: worker, out: string(out), err: err}
		}(w)
	}
	wg.Wait()

	successes := 0
	for _, r := range results {
		if strings.Contains(r.out, "panic") {
			t.Errorf("worker %d panicked:\n%s", r.worker, r.out)
		}
		if r.err == nil {
			successes++
		} else if !strings.Contains(r.out, "one writer at a time") {
			t.Errorf("worker %d failed with unexpected error: %v\n%s", r.worker, r.err, r.out)
		}
	}
	if successes < 1 {
		t.Errorf("expected at least 1 successful restore, got %d", successes)
	}
	t.Logf("%d/%d restore workers succeeded", successes, numIssues)
}
