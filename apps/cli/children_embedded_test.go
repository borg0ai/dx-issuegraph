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

// issuegraphChildren runs "issuegraph children" with the given args and returns stdout.
func issuegraphChildren(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"children"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph children %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func TestEmbeddedChildren(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ch")

	t.Run("children_basic", func(t *testing.T) {
		parent := issuegraphCreate(t, issuegraph, dir, "Parent epic", "--type", "epic")
		child1 := issuegraphCreate(t, issuegraph, dir, "Child task 1", "--type", "task")
		child2 := issuegraphCreate(t, issuegraph, dir, "Child task 2", "--type", "task")
		issuegraphDepAdd(t, issuegraph, dir, child1.ID, parent.ID, "--type", "parent-child")
		issuegraphDepAdd(t, issuegraph, dir, child2.ID, parent.ID, "--type", "parent-child")

		out := issuegraphChildren(t, issuegraph, dir, parent.ID)
		if !strings.Contains(out, child1.ID) {
			t.Errorf("expected child1 %s in output: %s", child1.ID, out)
		}
		if !strings.Contains(out, child2.ID) {
			t.Errorf("expected child2 %s in output: %s", child2.ID, out)
		}
	})

	t.Run("children_json", func(t *testing.T) {
		parent := issuegraphCreate(t, issuegraph, dir, "JSON parent", "--type", "epic")
		child := issuegraphCreate(t, issuegraph, dir, "JSON child", "--type", "task")
		issuegraphDepAdd(t, issuegraph, dir, child.ID, parent.ID, "--type", "parent-child")

		cmd := exec.Command(issuegraph, "children", parent.ID, "--json")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph children --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		s := strings.TrimSpace(stdout.String())
		start := strings.Index(s, "[")
		if start >= 0 {
			var issues []map[string]interface{}
			if err := json.Unmarshal([]byte(s[start:]), &issues); err != nil {
				t.Fatalf("parse children JSON: %v\n%s", err, s)
			}
			if len(issues) != 1 {
				t.Errorf("expected 1 child, got %d", len(issues))
			}
		}
	})

	t.Run("children_empty", func(t *testing.T) {
		parent := issuegraphCreate(t, issuegraph, dir, "No children parent", "--type", "task")
		out := issuegraphChildren(t, issuegraph, dir, parent.ID)
		// Should not error, may show empty message
		_ = out
	})

	t.Run("children_includes_all_statuses", func(t *testing.T) {
		parent := issuegraphCreate(t, issuegraph, dir, "All status parent", "--type", "epic")
		openChild := issuegraphCreate(t, issuegraph, dir, "Open child", "--type", "task")
		closedChild := issuegraphCreate(t, issuegraph, dir, "Closed child", "--type", "task")
		issuegraphDepAdd(t, issuegraph, dir, openChild.ID, parent.ID, "--type", "parent-child")
		issuegraphDepAdd(t, issuegraph, dir, closedChild.ID, parent.ID, "--type", "parent-child")
		issuegraphClose(t, issuegraph, dir, closedChild.ID)

		out := issuegraphChildren(t, issuegraph, dir, parent.ID)
		if !strings.Contains(out, openChild.ID) {
			t.Errorf("expected open child in output: %s", out)
		}
		if !strings.Contains(out, closedChild.ID) {
			t.Errorf("expected closed child in output (--all implied): %s", out)
		}
	})

	t.Run("children_nonexistent_parent", func(t *testing.T) {
		cmd := exec.Command(issuegraph, "children", "ch-nonexistent999")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected children of nonexistent to fail, got: %s", out)
		}
	})
}

// TestEmbeddedChildrenConcurrent exercises children listing concurrently.
func TestEmbeddedChildrenConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "cx")

	// Create parents with children
	var parentIDs []string
	for i := 0; i < 4; i++ {
		parent := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("Concurrent parent %d", i), "--type", "epic")
		for j := 0; j < 2; j++ {
			child := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("Child %d-%d", i, j), "--type", "task")
			issuegraphDepAdd(t, issuegraph, dir, child.ID, parent.ID, "--type", "parent-child")
		}
		parentIDs = append(parentIDs, parent.ID)
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

			parentID := parentIDs[worker%len(parentIDs)]
			cmd := exec.Command(issuegraph, "children", parentID, "--json")
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("children %s: %v\n%s", parentID, err, out)
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
