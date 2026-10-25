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

// issuegraphLint runs "issuegraph lint" with the given args and returns raw stdout and exit code.
func issuegraphLint(t *testing.T, issuegraph, dir string, args ...string) (string, int) {
	t.Helper()
	fullArgs := append([]string{"lint"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(out), exitErr.ExitCode()
		}
		t.Fatalf("issuegraph lint %s failed unexpectedly: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out), 0
}

// issuegraphLintJSON runs "issuegraph lint --json" and parses the result.
func issuegraphLintJSON(t *testing.T, issuegraph, dir string, args ...string) map[string]interface{} {
	t.Helper()
	fullArgs := append([]string{"lint", "--json"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	// lint exits 1 on warnings even with --json, so ignore exit error
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("issuegraph lint --json %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
		}
	}
	s := strings.TrimSpace(stdout.String())
	start := strings.Index(s, "{")
	if start < 0 {
		t.Fatalf("no JSON object in lint output:\nstdout: %s\nstderr: %s", s, stderr.String())
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
		t.Fatalf("parse lint JSON: %v\nstdout: %s", err, s)
	}
	return m
}

func TestEmbeddedLint(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ln")

	// Create issues with and without template violations.
	// Bug without "Steps to Reproduce" or "Acceptance Criteria" → lint warning.
	bugBare := issuegraphCreate(t, issuegraph, dir, "Bug without template", "--type", "bug",
		"--description", "Something is broken")

	// Bug with proper template sections → no warning.
	bugGood := issuegraphCreate(t, issuegraph, dir, "Bug with template", "--type", "bug",
		"--description", "## Steps to Reproduce\n1. Do X\n2. See Y\n\n## Acceptance Criteria\nShould not crash")

	// Task without acceptance criteria → lint warning.
	taskBare := issuegraphCreate(t, issuegraph, dir, "Task without AC", "--type", "task",
		"--description", "Just do it")

	// Chore → no template requirements, never warns.
	issuegraphCreate(t, issuegraph, dir, "Chore is fine", "--type", "chore")

	// Feature without AC → lint warning.
	issuegraphCreate(t, issuegraph, dir, "Feature no AC", "--type", "feature",
		"--description", "Add dark mode")

	// Closed issue for --status all testing.
	closedBug := issuegraphCreate(t, issuegraph, dir, "Closed bug bare", "--type", "bug",
		"--description", "Old bug")
	issuegraphClose(t, issuegraph, dir, closedBug.ID)

	// ===== Lint specific issue IDs =====

	t.Run("lint_specific_id_with_warnings", func(t *testing.T) {
		m := issuegraphLintJSON(t, issuegraph, dir, bugBare.ID)
		total := int(m["total"].(float64))
		if total == 0 {
			t.Error("expected warnings for bare bug")
		}
		results := m["results"].([]interface{})
		if len(results) == 0 {
			t.Error("expected results for bare bug")
		}
	})

	t.Run("lint_specific_id_clean", func(t *testing.T) {
		m := issuegraphLintJSON(t, issuegraph, dir, bugGood.ID)
		total := int(m["total"].(float64))
		if total != 0 {
			t.Errorf("expected 0 warnings for well-formatted bug, got %d", total)
		}
	})

	t.Run("lint_multiple_ids", func(t *testing.T) {
		m := issuegraphLintJSON(t, issuegraph, dir, bugBare.ID, taskBare.ID)
		issues := int(m["issues"].(float64))
		if issues < 2 {
			t.Errorf("expected at least 2 issues with warnings, got %d", issues)
		}
	})

	// ===== Lint by --status and --type filters =====

	t.Run("lint_by_type_bug", func(t *testing.T) {
		m := issuegraphLintJSON(t, issuegraph, dir, "--type", "bug")
		// Should find at least the bare bug
		results := m["results"].([]interface{})
		found := false
		for _, r := range results {
			rm := r.(map[string]interface{})
			if rm["id"] == bugBare.ID {
				found = true
			}
		}
		if !found {
			t.Errorf("expected bare bug %s in lint results", bugBare.ID)
		}
	})

	t.Run("lint_by_type_chore_no_warnings", func(t *testing.T) {
		m := issuegraphLintJSON(t, issuegraph, dir, "--type", "chore")
		total := int(m["total"].(float64))
		if total != 0 {
			t.Errorf("expected 0 warnings for chores, got %d", total)
		}
	})

	// ===== --status all includes closed =====

	t.Run("lint_status_all", func(t *testing.T) {
		m := issuegraphLintJSON(t, issuegraph, dir, "--status", "all")
		// Should include the closed bare bug
		results := m["results"].([]interface{})
		found := false
		for _, r := range results {
			rm := r.(map[string]interface{})
			if rm["id"] == closedBug.ID {
				found = true
			}
		}
		if !found {
			t.Errorf("expected closed bug %s in --status all results", closedBug.ID)
		}
	})

	t.Run("lint_status_default_excludes_closed", func(t *testing.T) {
		m := issuegraphLintJSON(t, issuegraph, dir)
		results := m["results"].([]interface{})
		for _, r := range results {
			rm := r.(map[string]interface{})
			if rm["id"] == closedBug.ID {
				t.Errorf("closed bug %s should be excluded by default", closedBug.ID)
			}
		}
	})

	// ===== Issues with/without template violations =====

	t.Run("clean_issues_not_in_results", func(t *testing.T) {
		m := issuegraphLintJSON(t, issuegraph, dir)
		results := m["results"].([]interface{})
		for _, r := range results {
			rm := r.(map[string]interface{})
			if rm["id"] == bugGood.ID {
				t.Errorf("well-formatted bug %s should not have warnings", bugGood.ID)
			}
		}
	})

	// ===== JSON output with missing sections =====

	t.Run("json_missing_sections", func(t *testing.T) {
		m := issuegraphLintJSON(t, issuegraph, dir, bugBare.ID)
		results := m["results"].([]interface{})
		if len(results) == 0 {
			t.Fatal("expected results")
		}
		rm := results[0].(map[string]interface{})
		missing, ok := rm["missing"].([]interface{})
		if !ok || len(missing) == 0 {
			t.Error("expected non-empty 'missing' array")
		}
	})

	// ===== Exit code 1 when warnings found =====

	t.Run("exit_code_1_on_warnings", func(t *testing.T) {
		_, exitCode := issuegraphLint(t, issuegraph, dir)
		if exitCode != 1 {
			t.Errorf("expected exit code 1 when warnings exist, got %d", exitCode)
		}
	})

	t.Run("exit_code_0_when_clean", func(t *testing.T) {
		_, exitCode := issuegraphLint(t, issuegraph, dir, "--type", "chore")
		if exitCode != 0 {
			t.Errorf("expected exit code 0 for chores, got %d", exitCode)
		}
	})

	// ===== Nonexistent issue ID =====

	t.Run("nonexistent_id_graceful", func(t *testing.T) {
		// Should not crash, just print error to stderr
		m := issuegraphLintJSON(t, issuegraph, dir, "ln-nonexistent999")
		total := int(m["total"].(float64))
		if total != 0 {
			t.Errorf("expected 0 warnings for nonexistent issue, got %d", total)
		}
	})

	// ===== Human-readable output =====

	t.Run("human_readable_warnings", func(t *testing.T) {
		out, _ := issuegraphLint(t, issuegraph, dir)
		if !strings.Contains(out, "Missing:") {
			t.Errorf("expected 'Missing:' in human output: %s", out)
		}
	})

	t.Run("human_readable_clean", func(t *testing.T) {
		out, _ := issuegraphLint(t, issuegraph, dir, "--type", "chore")
		if !strings.Contains(out, "No template warnings") {
			t.Errorf("expected 'No template warnings' for chores: %s", out)
		}
	})
}

// TestEmbeddedLintConcurrent exercises lint operations concurrently.
func TestEmbeddedLintConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "lc")

	// Create a mix of issues.
	for i := 0; i < 10; i++ {
		issueType := "task"
		if i%3 == 0 {
			issueType = "bug"
		}
		issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("concurrent-lint-%d", i), "--type", issueType,
			"--description", "Bare issue for lint")
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

			queries := [][]string{
				{"--json"},
				{"--json", "--type", "bug"},
				{"--json", "--type", "task"},
				{"--json", "--status", "all"},
				{"--json"},
				{"--json", "--type", "chore"},
				{"--json", "--type", "bug"},
				{"--json"},
			}
			q := queries[worker%len(queries)]

			args := append([]string{"lint"}, q...)
			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				// lint exits 1 on warnings, which is expected
				if _, ok := err.(*exec.ExitError); !ok {
					r.err = fmt.Errorf("worker %d lint: %v\n%s", worker, err, out)
				}
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
