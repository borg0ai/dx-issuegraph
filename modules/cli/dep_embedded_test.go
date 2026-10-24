//go:build cgo

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage/embeddeddolt"
)

// issuegraphDep runs "issuegraph dep" with the given args and returns raw stdout.
func issuegraphDep(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph dep %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphDepFail runs "issuegraph dep" expecting failure.
func issuegraphDepFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph dep %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

// issuegraphDepJSON runs "issuegraph dep" with --json and parses the result.
func issuegraphDepJSON(t *testing.T, issuegraph, dir string, args ...string) map[string]interface{} {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	fullArgs = append(fullArgs, "--json")
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph dep --json %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	s := strings.TrimSpace(stdout.String())
	start := strings.IndexAny(s, "{[")
	if start < 0 {
		t.Fatalf("no JSON in dep output: %s", s)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
		t.Fatalf("parse dep JSON: %v\n%s", err, s)
	}
	return m
}

func issuegraphDepWithInput(t *testing.T, issuegraph, dir, input string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	cmd.Stdin = strings.NewReader(input)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph dep %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphDepWithInputFail runs "issuegraph dep" with stdin input expecting failure.
func issuegraphDepWithInputFail(t *testing.T, issuegraph, dir, input string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph dep %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedDep(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dp")

	// Pre-create issues for dependency testing.
	issueA := issuegraphCreate(t, issuegraph, dir, "Dep issue A", "--type", "task")
	issueB := issuegraphCreate(t, issuegraph, dir, "Dep issue B", "--type", "task")
	issueC := issuegraphCreate(t, issuegraph, dir, "Dep issue C", "--type", "task")
	issueD := issuegraphCreate(t, issuegraph, dir, "Dep issue D", "--type", "task")
	epic := issuegraphCreate(t, issuegraph, dir, "Dep epic", "--type", "epic")
	child1 := issuegraphCreate(t, issuegraph, dir, "Dep child 1", "--type", "task")
	child2 := issuegraphCreate(t, issuegraph, dir, "Dep child 2", "--type", "task")

	// ===== dep --blocks =====

	t.Run("root_blocks_flag", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, issueA.ID, "--blocks", issueB.ID)
		if !strings.Contains(out, "Added") || !strings.Contains(out, "blocks") {
			t.Errorf("expected 'Added ... blocks' output: %s", out)
		}
	})

	// ===== dep add =====

	t.Run("add_positional_args", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, "add", issueC.ID, issueD.ID)
		if !strings.Contains(out, "Added dependency") {
			t.Errorf("expected 'Added dependency': %s", out)
		}
	})

	t.Run("add_type_parent_child", func(t *testing.T) {
		issuegraphDep(t, issuegraph, dir, "add", child1.ID, epic.ID, "--type", "parent-child")
		issuegraphDep(t, issuegraph, dir, "add", child2.ID, epic.ID, "--type", "parent-child")
		// Verify via dep list
		out := issuegraphDep(t, issuegraph, dir, "list", epic.ID, "--direction", "up")
		if !strings.Contains(out, child1.ID) {
			t.Errorf("expected child1 in dependents: %s", out)
		}
	})

	t.Run("add_type_tracks", func(t *testing.T) {
		tracker := issuegraphCreate(t, issuegraph, dir, "Tracker", "--type", "task")
		tracked := issuegraphCreate(t, issuegraph, dir, "Tracked", "--type", "task")
		issuegraphDep(t, issuegraph, dir, "add", tracker.ID, tracked.ID, "--type", "tracks")
		out := issuegraphDep(t, issuegraph, dir, "list", tracker.ID)
		if !strings.Contains(out, tracked.ID) {
			t.Errorf("expected tracked issue in deps: %s", out)
		}
	})

	t.Run("add_type_related", func(t *testing.T) {
		r1 := issuegraphCreate(t, issuegraph, dir, "Related 1", "--type", "task")
		r2 := issuegraphCreate(t, issuegraph, dir, "Related 2", "--type", "task")
		issuegraphDep(t, issuegraph, dir, "add", r1.ID, r2.ID, "--type", "related")
		// Should succeed without error
	})

	t.Run("add_blocked_by_flag", func(t *testing.T) {
		x := issuegraphCreate(t, issuegraph, dir, "Blocked by test", "--type", "task")
		y := issuegraphCreate(t, issuegraph, dir, "Blocker test", "--type", "task")
		issuegraphDep(t, issuegraph, dir, "add", x.ID, "--blocked-by", y.ID)
		out := issuegraphDep(t, issuegraph, dir, "list", x.ID)
		if !strings.Contains(out, y.ID) {
			t.Errorf("expected blocker in deps: %s", out)
		}
	})

	t.Run("add_depends_on_flag", func(t *testing.T) {
		x := issuegraphCreate(t, issuegraph, dir, "Depends on test", "--type", "task")
		y := issuegraphCreate(t, issuegraph, dir, "Dependency test", "--type", "task")
		issuegraphDep(t, issuegraph, dir, "add", x.ID, "--depends-on", y.ID)
		out := issuegraphDep(t, issuegraph, dir, "list", x.ID)
		if !strings.Contains(out, y.ID) {
			t.Errorf("expected dependency in deps: %s", out)
		}
	})

	t.Run("add_cycle_rejected", func(t *testing.T) {
		// A->B already exists, add B->A to create cycle — should be rejected
		cyA := issuegraphCreate(t, issuegraph, dir, "Cycle A", "--type", "task")
		cyB := issuegraphCreate(t, issuegraph, dir, "Cycle B", "--type", "task")
		issuegraphDep(t, issuegraph, dir, "add", cyA.ID, cyB.ID)
		out := issuegraphDepFail(t, issuegraph, dir, "add", cyB.ID, cyA.ID)
		if !strings.Contains(out, "cycle") {
			t.Errorf("expected 'cycle' error: %s", out)
		}
	})

	t.Run("add_child_parent_antipattern", func(t *testing.T) {
		p := issuegraphCreate(t, issuegraph, dir, "AP Parent", "--type", "epic")
		// Create child with hierarchical ID
		c := issuegraphCreate(t, issuegraph, dir, "AP Child", "--type", "task")
		issuegraphDep(t, issuegraph, dir, "add", c.ID, p.ID, "--type", "parent-child")
		// Try to add child->parent blocking dep (should fail)
		issuegraphDepFail(t, issuegraph, dir, "add", c.ID, p.ID)
	})

	t.Run("add_json_output", func(t *testing.T) {
		j1 := issuegraphCreate(t, issuegraph, dir, "JSON dep A", "--type", "task")
		j2 := issuegraphCreate(t, issuegraph, dir, "JSON dep B", "--type", "task")
		m := issuegraphDepJSON(t, issuegraph, dir, "add", j1.ID, j2.ID)
		if m["status"] != "added" {
			t.Errorf("expected status=added, got %v", m["status"])
		}
	})

	t.Run("add_bulk_file_jsonl", func(t *testing.T) {
		b1 := issuegraphCreate(t, issuegraph, dir, "Bulk dep A", "--type", "task")
		b2 := issuegraphCreate(t, issuegraph, dir, "Bulk dep B", "--type", "task")
		b3 := issuegraphCreate(t, issuegraph, dir, "Bulk dep C", "--type", "task")
		path := filepath.Join(t.TempDir(), "deps.jsonl")
		body := fmt.Sprintf("{\"from\":%q,\"to\":%q}\n{\"issue_id\":%q,\"depends_on_id\":%q,\"type\":\"tracks\"}\n", b1.ID, b2.ID, b3.ID, b2.ID)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("write deps file: %v", err)
		}

		out := issuegraphDep(t, issuegraph, dir, "add", "--file", path)
		if !strings.Contains(out, "Added 2 dependencies") {
			t.Fatalf("expected bulk add summary, got: %s", out)
		}
		list1 := issuegraphDep(t, issuegraph, dir, "list", b1.ID)
		if !strings.Contains(list1, b2.ID) {
			t.Fatalf("expected first bulk dependency in list: %s", list1)
		}
		list3 := issuegraphDep(t, issuegraph, dir, "list", b3.ID)
		if !strings.Contains(list3, b2.ID) || !strings.Contains(list3, "tracks") {
			t.Fatalf("expected typed bulk dependency in list: %s", list3)
		}
	})

	t.Run("add_bulk_file_validation_no_partial_mutation", func(t *testing.T) {
		v1 := issuegraphCreate(t, issuegraph, dir, "Bulk validation A", "--type", "task")
		v2 := issuegraphCreate(t, issuegraph, dir, "Bulk validation B", "--type", "task")
		path := filepath.Join(t.TempDir(), "bad-deps.jsonl")
		body := fmt.Sprintf("{\"from\":%q,\"to\":%q}\n{\"from\":\"\",\"to\":%q}\n{\"from\":%q,\"to\":\"\"}\n", v1.ID, v2.ID, v2.ID, v1.ID)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("write bad deps file: %v", err)
		}

		out := issuegraphDepFail(t, issuegraph, dir, "add", "--file", path)
		if !strings.Contains(out, "line 2: missing from") || !strings.Contains(out, "line 3: missing to") {
			t.Fatalf("expected all validation errors, got: %s", out)
		}
		list := issuegraphDep(t, issuegraph, dir, "list", v1.ID)
		if strings.Contains(list, v2.ID) {
			t.Fatalf("bulk validation failure should not add valid rows: %s", list)
		}
	})

	// ===== dep remove =====

	t.Run("remove_basic", func(t *testing.T) {
		r1 := issuegraphCreate(t, issuegraph, dir, "Remove A", "--type", "task")
		r2 := issuegraphCreate(t, issuegraph, dir, "Remove B", "--type", "task")
		issuegraphDep(t, issuegraph, dir, "add", r1.ID, r2.ID)
		out := issuegraphDep(t, issuegraph, dir, "remove", r1.ID, r2.ID)
		if !strings.Contains(out, "Removed") {
			t.Errorf("expected 'Removed' in output: %s", out)
		}
	})

	t.Run("remove_rm_alias", func(t *testing.T) {
		r1 := issuegraphCreate(t, issuegraph, dir, "Rm A", "--type", "task")
		r2 := issuegraphCreate(t, issuegraph, dir, "Rm B", "--type", "task")
		issuegraphDep(t, issuegraph, dir, "add", r1.ID, r2.ID)
		out := issuegraphDep(t, issuegraph, dir, "rm", r1.ID, r2.ID)
		if !strings.Contains(out, "Removed") {
			t.Errorf("expected 'Removed' via rm alias: %s", out)
		}
	})

	t.Run("remove_json_output", func(t *testing.T) {
		r1 := issuegraphCreate(t, issuegraph, dir, "RmJSON A", "--type", "task")
		r2 := issuegraphCreate(t, issuegraph, dir, "RmJSON B", "--type", "task")
		issuegraphDep(t, issuegraph, dir, "add", r1.ID, r2.ID)
		m := issuegraphDepJSON(t, issuegraph, dir, "remove", r1.ID, r2.ID)
		if m["status"] != "removed" {
			t.Errorf("expected status=removed, got %v", m["status"])
		}
		if m["removed"] != true {
			t.Errorf("expected removed=true, got %v", m["removed"])
		}
	})

	t.Run("remove_missing_edge_reports_noop", func(t *testing.T) {
		r1 := issuegraphCreate(t, issuegraph, dir, "Missing edge A", "--type", "task")
		r2 := issuegraphCreate(t, issuegraph, dir, "Missing edge B", "--type", "task")
		out := issuegraphDep(t, issuegraph, dir, "remove", r1.ID, r2.ID)
		if !strings.Contains(out, "No dependency found") || strings.Contains(out, "✓") {
			t.Errorf("missing edge should report a no-op, got: %s", out)
		}
		m := issuegraphDepJSON(t, issuegraph, dir, "remove", r1.ID, r2.ID)
		if m["status"] != "not_found" || m["removed"] != false {
			t.Errorf("missing edge JSON = %v, want status=not_found removed=false", m)
		}
	})

	// ===== dep list =====

	t.Run("list_default_direction_down", func(t *testing.T) {
		// issueC depends on issueD (added earlier)
		out := issuegraphDep(t, issuegraph, dir, "list", issueC.ID)
		if !strings.Contains(out, issueD.ID) {
			t.Errorf("expected dependency in list output: %s", out)
		}
	})

	t.Run("list_direction_up", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, "list", issueD.ID, "--direction", "up")
		if !strings.Contains(out, issueC.ID) {
			t.Errorf("expected dependent in list --direction up: %s", out)
		}
	})

	t.Run("list_type_filter", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, "list", epic.ID, "--direction", "up", "--type", "parent-child")
		if !strings.Contains(out, child1.ID) || !strings.Contains(out, child2.ID) {
			t.Errorf("expected children in type-filtered list: %s", out)
		}
	})

	t.Run("list_json_output", func(t *testing.T) {
		fullArgs := []string{"dep", "list", issueC.ID, "--json"}
		cmd := exec.Command(issuegraph, fullArgs...)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("dep list --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), issueD.ID) {
			t.Errorf("expected dependency ID in JSON: %s", stdout.String())
		}
	})

	// ===== dep tree =====

	t.Run("tree_basic", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, "tree", epic.ID)
		if !strings.Contains(out, epic.ID) {
			t.Errorf("expected epic ID in tree: %s", out)
		}
	})

	t.Run("tree_direction_up", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, "tree", child1.ID, "--direction", "up")
		if len(out) == 0 {
			t.Error("expected non-empty tree output")
		}
	})

	t.Run("tree_direction_both", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, "tree", epic.ID, "--direction", "both")
		if len(out) == 0 {
			t.Error("expected non-empty tree output for --direction both")
		}
	})

	t.Run("tree_max_depth", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, "tree", epic.ID, "--max-depth", "1")
		// Should show epic but not recurse deeply
		if !strings.Contains(out, epic.ID) {
			t.Errorf("expected epic in shallow tree: %s", out)
		}
	})

	t.Run("tree_status_filter", func(t *testing.T) {
		// Filter to open only — should still work
		issuegraphDep(t, issuegraph, dir, "tree", epic.ID, "--status", "open")
		// Just verify no crash
	})

	t.Run("tree_format_mermaid", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, "tree", epic.ID, "--format", "mermaid")
		if !strings.Contains(out, "flowchart") && !strings.Contains(out, "graph") {
			t.Errorf("expected mermaid flowchart/graph syntax: %s", out)
		}
	})

	t.Run("tree_json_output", func(t *testing.T) {
		fullArgs := []string{"dep", "tree", epic.ID, "--json"}
		cmd := exec.Command(issuegraph, fullArgs...)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("dep tree --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), epic.ID) {
			t.Errorf("expected epic ID in JSON tree: %s", stdout.String())
		}
	})

	t.Run("tree_ignores_bidirectional_relates_to", func(t *testing.T) {
		root := issuegraphCreate(t, issuegraph, dir, "Tree relates root", "--type", "task")
		blocker := issuegraphCreate(t, issuegraph, dir, "Tree real blocker", "--type", "task")
		related := issuegraphCreate(t, issuegraph, dir, "Tree loose relation", "--type", "task")

		issuegraphDep(t, issuegraph, dir, "add", root.ID, blocker.ID, "--type", "blocks")
		issuegraphDep(t, issuegraph, dir, "add", root.ID, related.ID, "--type", "relates-to")
		issuegraphDep(t, issuegraph, dir, "add", related.ID, root.ID, "--type", "relates-to")

		listOut := issuegraphDep(t, issuegraph, dir, "list", root.ID)
		if !strings.Contains(listOut, related.ID) || !strings.Contains(listOut, "relates-to") {
			t.Fatalf("expected relates-to edge in dep list output: %s", listOut)
		}

		treeOut := issuegraphDep(t, issuegraph, dir, "tree", root.ID)
		if !strings.Contains(treeOut, root.ID) || !strings.Contains(treeOut, blocker.ID) {
			t.Fatalf("expected root and real blocker in dep tree: %s", treeOut)
		}
		if strings.Contains(treeOut, related.ID) {
			t.Fatalf("relates-to edge should not render as a dependency tree edge: %s", treeOut)
		}

		upOut := issuegraphDep(t, issuegraph, dir, "tree", related.ID, "--direction", "up")
		if strings.Contains(upOut, root.ID) {
			t.Fatalf("reverse relates-to edge should not render as a dependent tree edge: %s", upOut)
		}
	})

	// ===== dep cycles =====

	t.Run("cycles_detect", func(t *testing.T) {
		// We created a cycle earlier (cyA <-> cyB)
		out := issuegraphDep(t, issuegraph, dir, "cycles")
		// Should report at least one cycle
		if !strings.Contains(out, "cycle") && !strings.Contains(out, "Cycle") && !strings.Contains(out, "No cycles") {
			t.Errorf("expected cycle info in output: %s", out)
		}
	})

	t.Run("cycles_no_cycles", func(t *testing.T) {
		// Fresh init with no cycles
		dir2, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dp2")
		a := issuegraphCreate(t, issuegraph, dir2, "No cycle A", "--type", "task")
		b := issuegraphCreate(t, issuegraph, dir2, "No cycle B", "--type", "task")
		issuegraphDep(t, issuegraph, dir2, "add", a.ID, b.ID)
		out := issuegraphDep(t, issuegraph, dir2, "cycles")
		if !strings.Contains(out, "No dependency cycles detected") {
			t.Errorf("expected no-cycle message: %s", out)
		}
	})
}

// TestEmbeddedDepConcurrent exercises dep operations concurrently.
func TestEmbeddedDepConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dpc")

	// Pre-create issues.
	var ids []string
	for i := 0; i < 16; i++ {
		issue := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("concurrent-dep-%d", i), "--type", "task")
		ids = append(ids, issue.ID)
	}
	// Pre-create some deps for read operations.
	for i := 0; i < 8; i++ {
		issuegraphDep(t, issuegraph, dir, "add", ids[i*2], ids[i*2+1])
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
			id := ids[worker*2]

			// Read operations: list and tree
			args := []string{"dep", "list", id, "--json"}
			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("worker %d dep list: %v\n%s", worker, err, out)
				results[worker] = r
				return
			}

			args = []string{"dep", "tree", id}
			cmd = exec.Command(issuegraph, args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err = cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("worker %d dep tree: %v\n%s", worker, err, out)
				results[worker] = r
				return
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

func TestEmbeddedDepNoCycleCheck(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ncc")

	// Create a linear chain of issues to simulate bulk dependency wiring.
	const n = 10
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		issue := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("bulk-dep-%d", i), "--type", "task")
		ids[i] = issue.ID
	}

	// Wire the chain with --no-cycle-check — each call must succeed without
	// hanging on a full-graph cycle traversal.
	for i := 1; i < n; i++ {
		out := issuegraphDep(t, issuegraph, dir, "add", ids[i], ids[i-1], "--no-cycle-check")
		// Must not print a cycle warning (there are no cycles in a linear chain).
		if strings.Contains(out, "cycle") {
			t.Errorf("unexpected cycle warning with --no-cycle-check: %s", out)
		}
	}

	// Verify the graph is acyclic after bulk wiring.
	cyclesOut := issuegraphDep(t, issuegraph, dir, "cycles")
	if strings.Contains(cyclesOut, "Found") {
		t.Errorf("unexpected cycles after bulk wiring: %s", cyclesOut)
	}

	// Verify --no-cycle-check also works with the dep --blocks shorthand.
	extra := issuegraphCreate(t, issuegraph, dir, "bulk-dep-extra", "--type", "task")
	issuegraphDep(t, issuegraph, dir, extra.ID, "--blocks", ids[n-1], "--no-cycle-check")
}

// TestEmbeddedDepBulkNoCycleCheckKeepsWholeGraphGate pins the bd-6dnrw.8
// contract: bulk --no-cycle-check skips the per-edge recursive check for
// speed, but one whole-graph cycle check still gates the commit, so a bulk
// add that would introduce a cycle rolls back atomically instead of landing
// and poisoning ready-work.
func TestEmbeddedDepBulkNoCycleCheckKeepsWholeGraphGate(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "bcy")

	a := issuegraphCreate(t, issuegraph, dir, "Bulk cycle A", "--type", "task")
	b := issuegraphCreate(t, issuegraph, dir, "Bulk cycle B", "--type", "task")
	c := issuegraphCreate(t, issuegraph, dir, "Bulk cycle C", "--type", "task")

	// Cycle-free bulk wiring succeeds with the per-edge check skipped.
	okInput := fmt.Sprintf("{\"from\":%q,\"to\":%q}\n{\"from\":%q,\"to\":%q}\n", a.ID, b.ID, b.ID, c.ID)
	out := issuegraphDepWithInput(t, issuegraph, dir, okInput, "add", "--file", "-", "--no-cycle-check")
	if !strings.Contains(out, "Added 2 dependencies") {
		t.Fatalf("expected bulk add summary, got: %s", out)
	}

	// Closing the chain into a cycle is refused by the final whole-graph
	// check, and the refused batch commits nothing.
	cycleInput := fmt.Sprintf("{\"from\":%q,\"to\":%q}\n", c.ID, a.ID)
	failOut := issuegraphDepWithInputFail(t, issuegraph, dir, cycleInput, "add", "--file", "-", "--no-cycle-check")
	if !strings.Contains(failOut, "dependency cycle would be created") || !strings.Contains(failOut, "no edges added") {
		t.Fatalf("expected whole-graph cycle rejection, got: %s", failOut)
	}

	cycles := issuegraphDep(t, issuegraph, dir, "cycles")
	if !strings.Contains(cycles, "No dependency cycles detected") {
		t.Fatalf("expected rolled-back bulk add to leave the graph acyclic, got: %s", cycles)
	}
}

// TestEmbeddedDepAddWispSource is the front-door test for a wisp-sourced
// dependency edge: `issuegraph dep add <wisp-id> <target>` on the real CLI against a
// real embedded store. Coverage of source routing otherwise stops at the
// storage role's own contract, and the behaviour this pins is the user-visible
// one — pinning an edge's source to the issues plane refused every one of
// these with "issue <id> not found", because a wisp has no row there.
//
// It also discharges the CHANGELOG's staging claim: every edge here is
// wisp-sourced, so all of them land in the dolt-ignored wisp plane, nothing
// versioned is staged, and the adds must record no Dolt commit and leave the
// working set clean.
func TestEmbeddedDepAddWispSource(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "dws")

	// --ephemeral is the front door's route to a wisp: the row lands in the
	// dolt-ignored `wisps` table, which is what the routing probe reads.
	wispA := issuegraphCreate(t, issuegraph, dir, "Wisp dep source", "--ephemeral").ID
	wispB := issuegraphCreate(t, issuegraph, dir, "Wisp dep target", "--ephemeral").ID
	wispC := issuegraphCreate(t, issuegraph, dir, "Wisp dep source two", "--ephemeral").ID
	issue := issuegraphCreate(t, issuegraph, dir, "Issue dep target", "--type", "task").ID

	// Baseline AFTER the fixtures, so a commit or a dirty table left by
	// creation cannot be misread as one the dependency adds produced.
	beforeCommits, beforeDirty := embeddedLogCountAndDirtyTables(t, issuegraphDir)
	if len(beforeDirty) != 0 {
		t.Fatalf("fixture setup left a dirty working set, so later state cannot be attributed: %v", beforeDirty)
	}

	t.Run("wisp_to_wisp", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, "add", wispA, wispB)
		if !strings.Contains(out, "Added dependency") {
			t.Fatalf("expected wisp-sourced dep add to succeed, got: %s", out)
		}
		if list := issuegraphDep(t, issuegraph, dir, "list", wispA); !strings.Contains(list, wispB) {
			t.Errorf("expected %s in dep list of wisp %s, got: %s", wispB, wispA, list)
		}
	})

	t.Run("wisp_to_issue", func(t *testing.T) {
		out := issuegraphDep(t, issuegraph, dir, "add", wispC, issue)
		if !strings.Contains(out, "Added dependency") {
			t.Fatalf("expected wisp->issue dep add to succeed, got: %s", out)
		}
		if list := issuegraphDep(t, issuegraph, dir, "list", wispC); !strings.Contains(list, issue) {
			t.Errorf("expected %s in dep list of wisp %s, got: %s", issue, wispC, list)
		}
		// The edge lives on the wisp's plane, so walking up from the issue
		// finds it only if the reverse read scans wisp_dependencies too.
		if up := issuegraphDep(t, issuegraph, dir, "list", issue, "--direction", "up"); !strings.Contains(up, wispC) {
			t.Errorf("expected wisp dependent %s walking up from %s, got: %s", wispC, issue, up)
		}
	})

	// Both adds were wisp-sourced. ChangedTables drops the wisp tables, so the
	// staged set was empty each time and the auto-commit had nothing to write.
	afterCommits, afterDirty := embeddedLogCountAndDirtyTables(t, issuegraphDir)
	if afterCommits != beforeCommits {
		t.Errorf("wisp-sourced dep adds recorded %d Dolt commit(s); the wisp plane is dolt-ignored and must record none",
			afterCommits-beforeCommits)
	}
	if len(afterDirty) != 0 {
		t.Errorf("wisp-sourced dep adds left versioned tables dirty: %v", afterDirty)
	}
}

// embeddedLogCountAndDirtyTables reports the embedded store's Dolt commit
// count and the tables its working set currently reports as changed.
func embeddedLogCountAndDirtyTables(t *testing.T, issuegraphDir string) (commits int, dirty []string) {
	t.Helper()
	cfg, _ := configfile.Load(issuegraphDir)
	database := ""
	if cfg != nil {
		database = cfg.GetDoltDatabase()
	}
	db, cleanup, err := embeddeddolt.OpenSQL(t.Context(), filepath.Join(issuegraphDir, "embeddeddolt"), database, "main")
	if err != nil {
		t.Fatalf("OpenSQL: %v", err)
	}
	defer cleanup()

	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM dolt_log").Scan(&commits); err != nil {
		t.Fatalf("query dolt_log: %v", err)
	}
	rows, err := db.QueryContext(t.Context(), "SELECT table_name, status FROM dolt_status")
	if err != nil {
		t.Fatalf("query dolt_status: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var table, status string
		if err := rows.Scan(&table, &status); err != nil {
			t.Fatalf("scan dolt_status: %v", err)
		}
		dirty = append(dirty, table+":"+status)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate dolt_status: %v", err)
	}
	return commits, dirty
}
