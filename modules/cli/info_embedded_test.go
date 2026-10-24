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

// issuegraphInfo runs "issuegraph info" with the given args and returns stdout.
func issuegraphInfo(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"info"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph info %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func TestEmbeddedInfo(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ti")

	// ===== Default Output =====

	t.Run("info_default", func(t *testing.T) {
		out := issuegraphInfo(t, issuegraph, dir)
		if !strings.Contains(out, ".issuegraph") {
			t.Errorf("expected .issuegraph path in info output: %s", out)
		}
		if !strings.Contains(out, "Issue Count") {
			t.Errorf("expected 'Issue Count' in info output: %s", out)
		}
	})

	t.Run("info_with_issues", func(t *testing.T) {
		issuegraphCreate(t, issuegraph, dir, "Info test issue 1", "--type", "task")
		issuegraphCreate(t, issuegraph, dir, "Info test issue 2", "--type", "task")

		out := issuegraphInfo(t, issuegraph, dir)
		// Should show non-zero issue count
		if strings.Contains(out, "Issue Count: 0") {
			t.Errorf("expected non-zero issue count after creating issues: %s", out)
		}
	})

	// ===== Schema Flag =====

	t.Run("info_schema", func(t *testing.T) {
		out := issuegraphInfo(t, issuegraph, dir, "--schema")
		if !strings.Contains(out, "issues") {
			t.Errorf("expected 'issues' table in schema output: %s", out)
		}
		if !strings.Contains(out, "Schema") {
			t.Errorf("expected 'Schema' heading in output: %s", out)
		}
	})

	// ===== Whats New =====

	t.Run("info_whats_new", func(t *testing.T) {
		out := issuegraphInfo(t, issuegraph, dir, "--whats-new")
		if len(strings.TrimSpace(out)) == 0 {
			t.Error("expected non-empty --whats-new output")
		}
		if !strings.Contains(out, "v0.") {
			t.Errorf("expected version string in whats-new output: %s", out[:min(200, len(out))])
		}
	})

	// ===== --json =====

	infoJSON := func(t *testing.T, args ...string) map[string]interface{} {
		t.Helper()
		out := issuegraphInfo(t, issuegraph, dir, append([]string{"--json"}, args...)...)
		var got map[string]interface{}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("info --json %v did not emit JSON: %v\n%s", args, err, out)
		}
		return got
	}

	t.Run("info_json", func(t *testing.T) {
		got := infoJSON(t)
		if got["mode"] != "direct" {
			t.Errorf("mode = %v, want direct", got["mode"])
		}
		if path, ok := got["database_path"].(string); !ok || !strings.Contains(path, ".issuegraph") {
			t.Errorf("database_path missing or not a .issuegraph path: %v", got["database_path"])
		}
		if _, ok := got["issue_count"].(float64); !ok {
			t.Errorf("issue_count missing or not a number: %v", got["issue_count"])
		}
	})

	t.Run("info_schema_json", func(t *testing.T) {
		got := infoJSON(t, "--schema")
		schema, ok := got["schema"].(map[string]interface{})
		if !ok {
			t.Fatalf("schema block missing: %v", got["schema"])
		}
		tables, ok := schema["tables"].([]interface{})
		if !ok {
			t.Fatalf("schema tables missing: %v", schema["tables"])
		}
		foundIssues := false
		for _, tbl := range tables {
			if tbl == "issues" {
				foundIssues = true
				break
			}
		}
		if !foundIssues {
			t.Errorf("expected 'issues' in schema tables: %v", tables)
		}
	})

	t.Run("info_whats_new_json", func(t *testing.T) {
		got := infoJSON(t, "--whats-new")
		if _, ok := got["current_version"].(string); !ok {
			t.Errorf("current_version missing or not a string: %v", got["current_version"])
		}
		changes, ok := got["recent_changes"].([]interface{})
		if !ok || len(changes) == 0 {
			t.Errorf("recent_changes missing or empty: %v", got["recent_changes"])
		}
	})
}

// TestEmbeddedInfoSuppressGitHooks pins GH#6027: issuegraph info must honor
// doctor.suppress.git-hooks the same way issuegraph doctor does, instead of printing
// the missing-hooks warning unconditionally.
func TestEmbeddedInfoSuppressGitHooks(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ih")
	issuegraphRunOK(t, issuegraph, dir, "hooks", "uninstall")

	const hookWarning = "Git hooks not installed"
	if out := issuegraphInfo(t, issuegraph, dir); !strings.Contains(out, hookWarning) {
		t.Fatalf("precondition: expected %q with hooks uninstalled:\n%s", hookWarning, out)
	}

	issuegraphConfig(t, issuegraph, dir, "set", "doctor.suppress.git-hooks", "true")
	if out := issuegraphInfo(t, issuegraph, dir); strings.Contains(out, hookWarning) {
		t.Errorf("doctor.suppress.git-hooks=true should hide %q:\n%s", hookWarning, out)
	}

	issuegraphConfig(t, issuegraph, dir, "set", "doctor.suppress.git-hooks", "false")
	if out := issuegraphInfo(t, issuegraph, dir); !strings.Contains(out, hookWarning) {
		t.Errorf("doctor.suppress.git-hooks=false should keep %q:\n%s", hookWarning, out)
	}
}

// TestEmbeddedInfoConcurrent exercises info operations concurrently.
func TestEmbeddedInfoConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ix")

	for i := 0; i < 3; i++ {
		issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("info-concurrent-%d", i), "--type", "task")
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

			var args []string
			switch worker % 3 {
			case 0:
				args = []string{"info"}
			case 1:
				args = []string{"info", "--schema"}
			case 2:
				args = []string{"info", "--whats-new"}
			}
			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("info (worker %d): %v\n%s", worker, err, out)
				results[worker] = r
				return
			}

			if len(strings.TrimSpace(string(out))) == 0 {
				r.err = fmt.Errorf("info (worker %d): empty output", worker)
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
