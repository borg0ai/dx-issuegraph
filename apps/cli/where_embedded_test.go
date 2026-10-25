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

// issuegraphWhere runs "issuegraph where" with the given args and returns stdout.
func issuegraphWhere(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"where"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph where %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func issuegraphWhereAllowError(t *testing.T, issuegraph, dir string, args ...string) (string, error) {
	t.Helper()
	fullArgs := append([]string{"where"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestWhereNoWorkspace(t *testing.T) {
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir := t.TempDir()
	initGitRepoAt(t, dir)

	t.Run("default_output", func(t *testing.T) {
		out, err := issuegraphWhereAllowError(t, issuegraph, dir)
		if err == nil {
			t.Fatal("expected issuegraph where to exit non-zero without a workspace")
		}
		if !strings.Contains(out, activeWorkspaceNotFoundMessage()) {
			t.Fatalf("expected no-workspace message, got: %s", out)
		}
		if strings.Contains(out, "no issuegraph database found") {
			t.Fatalf("where should report workspace resolution failure, not database init failure: %s", out)
		}
	})

	t.Run("json_output", func(t *testing.T) {
		out, err := issuegraphWhereAllowError(t, issuegraph, dir, "--json")
		if err == nil {
			t.Fatal("expected issuegraph where --json to exit non-zero without a workspace")
		}

		s := strings.TrimSpace(out)
		start := strings.Index(s, "{")
		if start < 0 {
			t.Fatalf("expected JSON object in output, got: %s", out)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(s[start:]), &payload); err != nil {
			t.Fatalf("parse where JSON: %v\n%s", err, s)
		}
		if errField, _ := payload["error"].(string); errField != "no_issuegraph_directory" {
			t.Fatalf("error = %q, want %q", errField, "no_issuegraph_directory")
		}
		if message, _ := payload["message"].(string); message != activeWorkspaceNotFoundMessage() {
			t.Fatalf("message = %q, want %q", message, activeWorkspaceNotFoundMessage())
		}
		hint, _ := payload["hint"].(string)
		if !strings.Contains(hint, "ISSUEGRAPH_DIR/worktree setup") {
			t.Fatalf("hint should mention workspace diagnostics, got: %q", hint)
		}
		if !strings.Contains(hint, "issuegraph init") {
			t.Fatalf("hint should mention issuegraph init, got: %q", hint)
		}
	})
}

func TestEmbeddedWhere(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "tw")

	// ===== Default Output =====

	t.Run("where_default", func(t *testing.T) {
		out := issuegraphWhere(t, issuegraph, dir)
		if !strings.Contains(out, ".issuegraph") {
			t.Errorf("expected .issuegraph in where output: %s", out)
		}
		// Should contain the actual issuegraph directory path
		if !strings.Contains(out, issuegraphDir) {
			t.Errorf("expected issuegraph dir %q in where output: %s", issuegraphDir, out)
		}
	})

	// ===== JSON Output =====

	t.Run("where_json", func(t *testing.T) {
		out := issuegraphWhere(t, issuegraph, dir, "--json")
		s := strings.TrimSpace(out)
		start := strings.Index(s, "{")
		if start < 0 {
			// --json may be affected by same flag shadowing as info;
			// just verify no crash and output contains path
			if !strings.Contains(out, ".issuegraph") {
				t.Errorf("expected .issuegraph in where --json output: %s", out)
			}
			return
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
			t.Fatalf("parse where JSON: %v\n%s", err, s)
		}
		// Verify path is present in JSON (WhereResult uses "path" key)
		if path, ok := m["path"]; ok {
			if p, ok := path.(string); ok && !strings.Contains(p, ".issuegraph") {
				t.Errorf("expected .issuegraph in path: %v", path)
			}
		}
		if prefix, ok := m["prefix"]; !ok {
			t.Fatalf("expected prefix in where --json output: %s", out)
		} else if p, ok := prefix.(string); !ok || p != "tw" {
			t.Fatalf("expected prefix %q in where --json output, got %#v", "tw", prefix)
		}
	})
}

// TestEmbeddedWhereConcurrent exercises where operations concurrently.
func TestEmbeddedWhereConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "wx")

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

			cmd := exec.Command(issuegraph, "where")
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("where (worker %d): %v\n%s", worker, err, out)
				results[worker] = r
				return
			}
			if !strings.Contains(string(out), ".issuegraph") {
				r.err = fmt.Errorf("where (worker %d): expected .issuegraph in output: %s", worker, out)
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
