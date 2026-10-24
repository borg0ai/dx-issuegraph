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

// issuegraphOrphans runs "issuegraph orphans" with the given args and returns stdout.
func issuegraphOrphans(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"orphans"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph orphans %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func TestEmbeddedOrphans(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "or")

	// Create some issues so the db isn't empty
	issuegraphCreate(t, issuegraph, dir, "Orphans test issue 1", "--type", "task")
	issuegraphCreate(t, issuegraph, dir, "Orphans test issue 2", "--type", "task")

	// ===== Default =====

	t.Run("orphans_default", func(t *testing.T) {
		out := issuegraphOrphans(t, issuegraph, dir)
		// On a clean db, should report no orphans or empty output
		_ = out
	})

	// ===== --json =====

	t.Run("orphans_json", func(t *testing.T) {
		out := issuegraphOrphans(t, issuegraph, dir, "--json")
		s := strings.TrimSpace(out)
		// orphans --json may return "null", "[]", or a JSON array/object
		if !json.Valid([]byte(s)) {
			t.Errorf("invalid JSON in orphans --json output: %s", s[:min(200, len(s))])
		}
	})

	// ===== --details =====

	t.Run("orphans_details", func(t *testing.T) {
		out := issuegraphOrphans(t, issuegraph, dir, "--details")
		_ = out // Should succeed without crashing
	})

	// ===== --label =====

	t.Run("orphans_label", func(t *testing.T) {
		// --label with a label that matches no issues should return no orphans
		out := issuegraphOrphans(t, issuegraph, dir, "--label", "nonexistent-label-xyz")
		_ = out // Should succeed without crashing
	})

	t.Run("orphans_label_json", func(t *testing.T) {
		out := issuegraphOrphans(t, issuegraph, dir, "--label", "nonexistent-label-xyz", "--json")
		s := strings.TrimSpace(out)
		if !json.Valid([]byte(s)) {
			t.Errorf("invalid JSON with --label --json: %s", s[:min(200, len(s))])
		}
	})

	t.Run("orphans_label_any", func(t *testing.T) {
		out := issuegraphOrphans(t, issuegraph, dir, "--label-any", "nonexistent-label-xyz")
		_ = out // Should succeed without crashing
	})
}

// TestEmbeddedOrphansConcurrent exercises orphans concurrently.
func TestEmbeddedOrphansConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ox")

	issuegraphCreate(t, issuegraph, dir, "Orphans concurrent issue", "--type", "task")

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
			cmd := exec.Command(issuegraph, "orphans")
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("orphans (worker %d): %v\n%s", worker, err, out)
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
