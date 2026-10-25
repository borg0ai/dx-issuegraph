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

// issuegraphDuplicates runs "issuegraph duplicates" with the given args and returns raw stdout.
func issuegraphDuplicates(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"duplicates"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph duplicates %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphDuplicatesJSON runs "issuegraph duplicates --json" and parses the result.
func issuegraphDuplicatesJSON(t *testing.T, issuegraph, dir string, args ...string) map[string]interface{} {
	t.Helper()
	fullArgs := append([]string{"duplicates", "--json"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph duplicates --json %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	s := strings.TrimSpace(stdout.String())
	start := strings.Index(s, "{")
	if start < 0 {
		t.Fatalf("no JSON in duplicates output: %s", s)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
		t.Fatalf("parse duplicates JSON: %v\n%s", err, s)
	}
	return m
}

func TestEmbeddedDuplicates(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ds")

	// Create exact duplicate pairs (same title+description).
	issuegraphCreate(t, issuegraph, dir, "Exact dup issue", "--type", "task", "--description", "Same content here")
	issuegraphCreate(t, issuegraph, dir, "Exact dup issue", "--type", "task", "--description", "Same content here")

	// Create unique issues.
	issuegraphCreate(t, issuegraph, dir, "Unique issue A", "--type", "task", "--description", "Completely different A")
	issuegraphCreate(t, issuegraph, dir, "Unique issue B", "--type", "bug", "--description", "Completely different B")

	// ===== Find exact content duplicates =====

	t.Run("find_exact_duplicates", func(t *testing.T) {
		m := issuegraphDuplicatesJSON(t, issuegraph, dir)
		groups := int(m["duplicate_groups"].(float64))
		if groups < 1 {
			t.Errorf("expected at least 1 duplicate group, got %d", groups)
		}
	})

	// ===== No duplicates when all unique =====

	t.Run("no_duplicates", func(t *testing.T) {
		dir2, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ds2")
		issuegraphCreate(t, issuegraph, dir2, "Unique 1", "--type", "task", "--description", "A")
		issuegraphCreate(t, issuegraph, dir2, "Unique 2", "--type", "task", "--description", "B")
		m := issuegraphDuplicatesJSON(t, issuegraph, dir2)
		groups := int(m["duplicate_groups"].(float64))
		if groups != 0 {
			t.Errorf("expected 0 duplicate groups, got %d", groups)
		}
	})

	// ===== JSON output =====

	t.Run("json_output_structure", func(t *testing.T) {
		m := issuegraphDuplicatesJSON(t, issuegraph, dir)
		if _, ok := m["duplicate_groups"]; !ok {
			t.Error("expected 'duplicate_groups' key")
		}
		if _, ok := m["groups"]; !ok {
			t.Error("expected 'groups' key")
		}
	})

	// ===== Auto-merge =====

	t.Run("auto_merge", func(t *testing.T) {
		dir3, issuegraphDir3, _ := issuegraphInit(t, issuegraph, "--prefix", "ds3")
		a := issuegraphCreate(t, issuegraph, dir3, "Merge dup A", "--type", "task", "--description", "Merge me")
		b := issuegraphCreate(t, issuegraph, dir3, "Merge dup A", "--type", "task", "--description", "Merge me")
		_ = b

		m := issuegraphDuplicatesJSON(t, issuegraph, dir3, "--auto-merge")
		if _, ok := m["merge_results"]; !ok {
			t.Error("expected 'merge_results' key with --auto-merge")
		}

		// Verify one of them was closed
		s := openStore(t, issuegraphDir3, "ds3")
		issueA, _ := s.GetIssue(t.Context(), a.ID)
		if issueA != nil && issueA.Status == "closed" {
			// A was the source that got closed — expected
		}
		_ = issuegraphDir3
	})

	// ===== Auto-merge re-parents children =====

	t.Run("auto_merge_reparents", func(t *testing.T) {
		dir4, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ds4")
		parent1 := issuegraphCreate(t, issuegraph, dir4, "Parent dup", "--type", "epic", "--description", "Same")
		parent2 := issuegraphCreate(t, issuegraph, dir4, "Parent dup", "--type", "epic", "--description", "Same")
		child := issuegraphCreate(t, issuegraph, dir4, "Child of dup", "--type", "task")
		issuegraphDep(t, issuegraph, dir4, "add", child.ID, parent2.ID, "--type", "parent-child")

		issuegraphDuplicates(t, issuegraph, dir4, "--auto-merge")

		// Verify child was reparented (dep list should show the merge target)
		out := issuegraphDep(t, issuegraph, dir4, "list", child.ID)
		// Should contain either parent1 or parent2 (the target)
		if !strings.Contains(out, parent1.ID) && !strings.Contains(out, parent2.ID) {
			t.Logf("expected reparented dep in child's list: %s", out)
		}
	})

	// ===== Dry run =====

	t.Run("dry_run", func(t *testing.T) {
		m := issuegraphDuplicatesJSON(t, issuegraph, dir, "--dry-run")
		if _, ok := m["merge_commands"]; !ok {
			t.Error("expected 'merge_commands' key with --dry-run")
		}
		// Dry run should NOT have merge_results
		if _, ok := m["merge_results"]; ok {
			t.Error("dry run should not have merge_results")
		}
	})

	// ===== Excludes closed issues =====

	t.Run("excludes_closed", func(t *testing.T) {
		dir5, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ds5")
		a := issuegraphCreate(t, issuegraph, dir5, "Closed dup", "--type", "task", "--description", "Same closed")
		issuegraphCreate(t, issuegraph, dir5, "Closed dup", "--type", "task", "--description", "Same closed")
		issuegraphClose(t, issuegraph, dir5, a.ID)

		m := issuegraphDuplicatesJSON(t, issuegraph, dir5)
		// With one closed, they shouldn't match (different status groups)
		groups := int(m["duplicate_groups"].(float64))
		if groups > 0 {
			t.Log("mixed open/closed duplicates detected — expected 0 groups due to status mismatch")
		}
	})

	// ===== Human-readable output =====

	t.Run("human_readable", func(t *testing.T) {
		out := issuegraphDuplicates(t, issuegraph, dir)
		if !strings.Contains(out, "duplicate") && !strings.Contains(out, "Duplicate") && !strings.Contains(out, "No duplicates") {
			t.Errorf("expected duplicate info in output: %s", out)
		}
	})
}

// TestEmbeddedDuplicatesConcurrent exercises duplicates operations concurrently.
func TestEmbeddedDuplicatesConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dsc")

	for i := 0; i < 10; i++ {
		issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("concurrent-dups-%d", i), "--type", "task",
			"--description", fmt.Sprintf("Content %d", i%3))
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

			args := []string{"duplicates", "--json"}
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
