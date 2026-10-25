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
)

// issuegraphSwarm runs "issuegraph swarm" with the given args and returns raw stdout.
// Stderr (warnings, tips) is captured separately so it does not pollute
// callers that parse stdout.
func issuegraphSwarm(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"swarm"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph swarm %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphSwarmFail runs "issuegraph swarm" expecting failure.
func issuegraphSwarmFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"swarm"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph swarm %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

// issuegraphSwarmJSON runs "issuegraph swarm" with --json and parses the result as a map.
// Stderr is captured separately so warnings do not corrupt JSON parsing.
func issuegraphSwarmJSON(t *testing.T, issuegraph, dir string, args ...string) map[string]interface{} {
	t.Helper()
	fullArgs := append([]string{"swarm"}, args...)
	fullArgs = append(fullArgs, "--json")
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph swarm --json %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	s := strings.TrimSpace(stdout.String())
	start := strings.IndexAny(s, "{[")
	if start < 0 {
		t.Fatalf("no JSON in swarm stdout: %s\nstderr:\n%s", s, stderr.String())
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
		t.Fatalf("parse swarm JSON: %v\n%s\nstderr:\n%s", err, s, stderr.String())
	}
	return m
}

// createSwarmableEpic creates an epic with 3 children forming a DAG suitable for swarming.
func createSwarmableEpic(t *testing.T, issuegraph, dir, prefix string) (epicID string, childIDs []string) {
	t.Helper()
	plan := fmt.Sprintf(`{
		"nodes": [
			{"key": "epic", "title": %q, "type": "epic"},
			{"key": "child1", "title": %q, "type": "task", "parent_key": "epic"},
			{"key": "child2", "title": %q, "type": "task", "parent_key": "epic"},
			{"key": "child3", "title": %q, "type": "task", "parent_key": "epic"}
		],
		"edges": [
			{"from_key": "child3", "to_key": "child1", "type": "blocks"}
		]
	}`, prefix+" epic", prefix+" child 1", prefix+" child 2", prefix+" child 3")
	planFile := filepath.Join(dir, "swarm-graph-plan.json")
	if err := os.WriteFile(planFile, []byte(plan), 0o600); err != nil {
		t.Fatalf("write swarm graph plan: %v", err)
	}

	result := issuegraphCreateGraph(t, issuegraph, dir, planFile)
	epicID = result.IDs["epic"]
	childIDs = []string{result.IDs["child1"], result.IDs["child2"], result.IDs["child3"]}
	allIDs := append([]string{epicID}, childIDs...)
	for i, id := range allIDs {
		if id == "" {
			t.Fatalf("graph create returned an empty ID at index %d: %#v", i, result.IDs)
		}
		for j := 0; j < i; j++ {
			if id == allIDs[j] {
				t.Fatalf("graph create returned duplicate IDs at indexes %d and %d: %#v", j, i, result.IDs)
			}
		}
	}
	return epicID, childIDs
}

func TestEmbeddedSwarm(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "sw")

	epicID, _ := createSwarmableEpic(t, issuegraph, dir, "Swarm")

	// ===== swarm validate =====

	t.Run("validate_epic", func(t *testing.T) {
		out := issuegraphSwarm(t, issuegraph, dir, "validate", epicID)
		if !strings.Contains(out, epicID) {
			t.Errorf("expected epic ID in validate output: %s", out)
		}
	})

	t.Run("validate_verbose", func(t *testing.T) {
		m := issuegraphSwarmJSON(t, issuegraph, dir, "validate", epicID, "--verbose")
		if _, ok := m["issues"]; !ok {
			t.Error("expected 'issues' key with --verbose")
		}
	})

	t.Run("validate_json", func(t *testing.T) {
		m := issuegraphSwarmJSON(t, issuegraph, dir, "validate", epicID)
		if _, ok := m["swarmable"]; !ok {
			t.Error("expected 'swarmable' key in JSON")
		}
		if _, ok := m["total_issues"]; !ok {
			t.Error("expected 'total_issues' key in JSON")
		}
	})

	t.Run("validate_non_swarmable", func(t *testing.T) {
		// Epic with no children — validate succeeds but warns
		emptyEpic := issuegraphCreate(t, issuegraph, dir, "Empty epic", "--type", "epic")
		out := issuegraphSwarm(t, issuegraph, dir, "validate", emptyEpic.ID)
		if !strings.Contains(out, "no children") && !strings.Contains(out, "0") {
			t.Errorf("expected warning about no children: %s", out)
		}
	})

	// ===== swarm create =====

	t.Run("create_swarm", func(t *testing.T) {
		eID, _ := createSwarmableEpic(t, issuegraph, dir, "Create")
		out := issuegraphSwarm(t, issuegraph, dir, "create", eID)
		if !strings.Contains(out, "Created swarm molecule") {
			t.Errorf("expected 'Created swarm molecule': %s", out)
		}
	})

	t.Run("create_auto_wrap_single", func(t *testing.T) {
		// Creating swarm on a non-epic should auto-wrap
		task := issuegraphCreate(t, issuegraph, dir, "Auto wrap task", "--type", "task")
		out := issuegraphSwarm(t, issuegraph, dir, "create", task.ID)
		if !strings.Contains(out, "Created swarm molecule") && !strings.Contains(out, "Auto-wrapping") {
			t.Errorf("expected auto-wrap or creation: %s", out)
		}
	})

	t.Run("create_coordinator", func(t *testing.T) {
		eID, _ := createSwarmableEpic(t, issuegraph, dir, "Coord")
		m := issuegraphSwarmJSON(t, issuegraph, dir, "create", eID, "--coordinator", "alice")
		if m["coordinator"] != "alice" {
			t.Errorf("expected coordinator=alice, got %v", m["coordinator"])
		}
	})

	t.Run("create_force", func(t *testing.T) {
		eID, _ := createSwarmableEpic(t, issuegraph, dir, "Force")
		issuegraphSwarm(t, issuegraph, dir, "create", eID)
		// Second create without --force should fail
		issuegraphSwarmFail(t, issuegraph, dir, "create", eID)
		// With --force should succeed
		issuegraphSwarm(t, issuegraph, dir, "create", eID, "--force")
	})

	t.Run("create_json_output", func(t *testing.T) {
		eID, _ := createSwarmableEpic(t, issuegraph, dir, "JSON")
		m := issuegraphSwarmJSON(t, issuegraph, dir, "create", eID)
		if _, ok := m["swarm_id"]; !ok {
			t.Error("expected 'swarm_id' in JSON")
		}
		if _, ok := m["epic_id"]; !ok {
			t.Error("expected 'epic_id' in JSON")
		}
	})

	t.Run("create_error_existing", func(t *testing.T) {
		eID, _ := createSwarmableEpic(t, issuegraph, dir, "Exist")
		issuegraphSwarm(t, issuegraph, dir, "create", eID)
		issuegraphSwarmFail(t, issuegraph, dir, "create", eID)
	})

	// ===== swarm status =====

	t.Run("status_shows_progress", func(t *testing.T) {
		eID, _ := createSwarmableEpic(t, issuegraph, dir, "Status")
		issuegraphSwarm(t, issuegraph, dir, "create", eID)
		out := issuegraphSwarm(t, issuegraph, dir, "status", eID)
		if len(out) == 0 {
			t.Error("expected non-empty status output")
		}
	})

	t.Run("status_json", func(t *testing.T) {
		eID, _ := createSwarmableEpic(t, issuegraph, dir, "StatJSON")
		issuegraphSwarm(t, issuegraph, dir, "create", eID)
		m := issuegraphSwarmJSON(t, issuegraph, dir, "status", eID)
		if len(m) == 0 {
			t.Error("expected non-empty JSON status")
		}
	})

	// ===== swarm list =====

	t.Run("list_shows_swarms", func(t *testing.T) {
		out := issuegraphSwarm(t, issuegraph, dir, "list")
		// We've created several swarms above
		if len(out) == 0 {
			t.Error("expected non-empty list output")
		}
	})

	t.Run("list_json", func(t *testing.T) {
		fullArgs := []string{"swarm", "list", "--json"}
		cmd := exec.Command(issuegraph, fullArgs...)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("swarm list --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		// Should be valid JSON on stdout (warnings, if any, are on stderr).
		s := strings.TrimSpace(stdout.String())
		if !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "[") {
			t.Errorf("expected JSON output on stdout: %s\nstderr: %s", s, stderr.String())
		}
	})

	t.Run("list_empty", func(t *testing.T) {
		dir2, _, _ := issuegraphInit(t, issuegraph, "--prefix", "sw2")
		out := issuegraphSwarm(t, issuegraph, dir2, "list")
		if !strings.Contains(out, "No swarm") && !strings.Contains(out, "no swarm") && len(strings.TrimSpace(out)) > 0 {
			// Just verify no crash — empty list message varies
		}
	})
}

// TestEmbeddedSwarmConcurrent exercises swarm operations concurrently.
func TestEmbeddedSwarmConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "swc")

	// Create several swarmable epics.
	var epicIDs []string
	for i := 0; i < 4; i++ {
		eID, _ := createSwarmableEpic(t, issuegraph, dir, fmt.Sprintf("Conc%d", i))
		epicIDs = append(epicIDs, eID)
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

			eID := epicIDs[worker%len(epicIDs)]
			queries := [][]string{
				{"validate", eID, "--json"},
				{"validate", eID, "--verbose", "--json"},
				{"validate", eID, "--json"},
				{"list", "--json"},
				{"validate", eID, "--json"},
				{"list", "--json"},
				{"validate", eID, "--json"},
				{"validate", eID, "--verbose", "--json"},
			}
			q := queries[worker%len(queries)]

			args := append([]string{"swarm"}, q...)
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
		if r.err != nil && !strings.Contains(r.err.Error(), "one writer at a time") && !strings.Contains(r.err.Error(), "exclusive lock") {
			t.Errorf("worker %d failed: %v", r.worker, r.err)
		}
	}
}
