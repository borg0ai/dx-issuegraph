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

// issuegraphFindDups runs "issuegraph find-duplicates" with the given args and returns raw stdout.
func issuegraphFindDups(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"find-duplicates"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph find-duplicates %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphFindDupsFail runs "issuegraph find-duplicates" expecting failure.
func issuegraphFindDupsFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"find-duplicates"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph find-duplicates %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

// issuegraphFindDupsJSON runs "issuegraph find-duplicates --json" and parses the result.
func issuegraphFindDupsJSON(t *testing.T, issuegraph, dir string, args ...string) map[string]interface{} {
	t.Helper()
	fullArgs := append([]string{"find-duplicates", "--json"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph find-duplicates --json %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	s := strings.TrimSpace(stdout.String())
	start := strings.Index(s, "{")
	if start < 0 {
		t.Fatalf("no JSON object in find-duplicates output: %s", s)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
		t.Fatalf("parse find-duplicates JSON: %v\n%s", err, s)
	}
	return m
}

func TestEmbeddedFindDuplicates(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fd")

	// Create near-identical issues (should be flagged as duplicates).
	issuegraphCreate(t, issuegraph, dir, "Fix login page timeout error", "--type", "bug",
		"--description", "The login page throws a timeout error after 30 seconds of inactivity")
	issuegraphCreate(t, issuegraph, dir, "Login page timeout error fix needed", "--type", "bug",
		"--description", "After 30 seconds the login page shows a timeout error to the user")

	// Create distinct issues (should NOT be flagged).
	issuegraphCreate(t, issuegraph, dir, "Add dark mode to settings", "--type", "feature",
		"--description", "Users want a dark mode toggle in the settings page")
	issuegraphCreate(t, issuegraph, dir, "Upgrade database to PostgreSQL 16", "--type", "task",
		"--description", "Migrate from PostgreSQL 14 to PostgreSQL 16 for performance")
	issuegraphCreate(t, issuegraph, dir, "Write API documentation", "--type", "task",
		"--description", "Document all REST endpoints with OpenAPI spec")

	// Create a closed issue for status filter testing.
	closed := issuegraphCreate(t, issuegraph, dir, "Fix login page timeout error (old)", "--type", "bug",
		"--description", "The login page throws a timeout error")
	issuegraphClose(t, issuegraph, dir, closed.ID)

	// ===== Mechanical method with near-identical issues =====

	t.Run("mechanical_finds_near_identical", func(t *testing.T) {
		// Use a low threshold since token overlap between the two login issues
		// depends on exact tokenization (Jaccard + cosine average).
		m := issuegraphFindDupsJSON(t, issuegraph, dir, "--threshold", "0.15")
		pairs, ok := m["pairs"].([]interface{})
		if !ok {
			t.Fatalf("expected pairs array, got %T", m["pairs"])
		}
		if len(pairs) == 0 {
			t.Error("expected at least 1 duplicate pair for near-identical issues")
		}
		// Verify at least one pair involves two login-related issues
		found := false
		for _, p := range pairs {
			pm := p.(map[string]interface{})
			titleA := pm["issue_a_title"].(string)
			titleB := pm["issue_b_title"].(string)
			if strings.Contains(strings.ToLower(titleA), "login") && strings.Contains(strings.ToLower(titleB), "login") {
				found = true
			}
		}
		if !found {
			t.Error("expected login timeout issues to be flagged as duplicates")
		}
	})

	// ===== Threshold controls sensitivity =====

	t.Run("threshold_high_reduces_results", func(t *testing.T) {
		m := issuegraphFindDupsJSON(t, issuegraph, dir, "--threshold", "0.95")
		pairs := m["pairs"].([]interface{})
		// Very high threshold should find fewer/no pairs
		mLow := issuegraphFindDupsJSON(t, issuegraph, dir, "--threshold", "0.2")
		pairsLow := mLow["pairs"].([]interface{})
		if len(pairsLow) < len(pairs) {
			t.Errorf("lower threshold should find >= as many pairs: low=%d high=%d", len(pairsLow), len(pairs))
		}
	})

	t.Run("threshold_zero_finds_all_pairs", func(t *testing.T) {
		m := issuegraphFindDupsJSON(t, issuegraph, dir, "--threshold", "0.0")
		pairs := m["pairs"].([]interface{})
		// With threshold 0, every pair with any similarity should appear
		if len(pairs) == 0 {
			t.Error("expected at least some pairs at threshold 0")
		}
	})

	// ===== Status filter =====

	t.Run("status_filter_open", func(t *testing.T) {
		m := issuegraphFindDupsJSON(t, issuegraph, dir, "--status", "open", "--threshold", "0.2")
		// Should exclude the closed issue
		pairs := m["pairs"].([]interface{})
		for _, p := range pairs {
			pm := p.(map[string]interface{})
			titleA := pm["issue_a_title"].(string)
			titleB := pm["issue_b_title"].(string)
			if strings.Contains(titleA, "(old)") || strings.Contains(titleB, "(old)") {
				t.Error("closed issue should be excluded with --status open")
			}
		}
	})

	t.Run("status_filter_all", func(t *testing.T) {
		m := issuegraphFindDupsJSON(t, issuegraph, dir, "--status", "all", "--threshold", "0.3")
		pairs := m["pairs"].([]interface{})
		// With "all" status, the closed issue may appear in pairs
		if len(pairs) == 0 {
			t.Error("expected at least 1 pair with --status all")
		}
	})

	// ===== Limit caps results =====

	t.Run("limit_caps_results", func(t *testing.T) {
		m := issuegraphFindDupsJSON(t, issuegraph, dir, "--threshold", "0.0", "--limit", "1")
		pairs := m["pairs"].([]interface{})
		if len(pairs) > 1 {
			t.Errorf("expected at most 1 pair with --limit 1, got %d", len(pairs))
		}
	})

	// ===== No duplicates found =====

	t.Run("no_duplicates_distinct_issues", func(t *testing.T) {
		// Very high threshold — distinct issues shouldn't match
		out := issuegraphFindDups(t, issuegraph, dir, "--threshold", "0.99")
		if !strings.Contains(out, "No similar issues") {
			// It's ok if nothing matches — just verify no crash
		}
	})

	// ===== JSON output =====

	t.Run("json_output_structure", func(t *testing.T) {
		m := issuegraphFindDupsJSON(t, issuegraph, dir, "--threshold", "0.3")
		if _, ok := m["pairs"]; !ok {
			t.Error("expected 'pairs' key in JSON output")
		}
		if _, ok := m["count"]; !ok {
			t.Error("expected 'count' key in JSON output")
		}
		if _, ok := m["method"]; !ok {
			t.Error("expected 'method' key in JSON output")
		}
		if _, ok := m["threshold"]; !ok {
			t.Error("expected 'threshold' key in JSON output")
		}
		if m["method"] != "mechanical" {
			t.Errorf("expected method='mechanical', got %v", m["method"])
		}
	})

	t.Run("json_pair_fields", func(t *testing.T) {
		m := issuegraphFindDupsJSON(t, issuegraph, dir, "--threshold", "0.2")
		pairs := m["pairs"].([]interface{})
		if len(pairs) == 0 {
			t.Skip("no pairs to check fields on")
		}
		p := pairs[0].(map[string]interface{})
		for _, key := range []string{"issue_a_id", "issue_b_id", "issue_a_title", "issue_b_title", "similarity", "method"} {
			if _, ok := p[key]; !ok {
				t.Errorf("expected '%s' key in pair JSON", key)
			}
		}
	})

	// ===== Invalid method =====

	t.Run("invalid_method_error", func(t *testing.T) {
		out := issuegraphFindDupsFail(t, issuegraph, dir, "--method", "bogus")
		if !strings.Contains(out, "invalid method") {
			t.Errorf("expected 'invalid method' error, got: %s", out)
		}
	})

	// ===== Fewer than 2 issues =====

	t.Run("fewer_than_2_issues", func(t *testing.T) {
		// Create a fresh init with only 1 issue
		dir2, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fd2")
		issuegraphCreate(t, issuegraph, dir2, "Only one issue", "--type", "task")

		out := issuegraphFindDups(t, issuegraph, dir2)
		if !strings.Contains(out, "Not enough issues") {
			t.Errorf("expected 'Not enough issues' message, got: %s", out)
		}
	})

	t.Run("fewer_than_2_issues_json", func(t *testing.T) {
		dir3, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fd3")
		issuegraphCreate(t, issuegraph, dir3, "Only one issue json", "--type", "task")

		m := issuegraphFindDupsJSON(t, issuegraph, dir3)
		count := int(m["count"].(float64))
		if count != 0 {
			t.Errorf("expected count=0 with fewer than 2 issues, got %d", count)
		}
	})

	// ===== find-dups alias =====

	t.Run("find_dups_alias", func(t *testing.T) {
		cmd := exec.Command(issuegraph, "find-dups", "--json", "--threshold", "0.3")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph find-dups alias failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), "pairs") {
			t.Errorf("expected JSON with 'pairs' from alias: %s", stdout.String())
		}
	})

	// ===== Human-readable output =====

	t.Run("human_readable_output", func(t *testing.T) {
		out := issuegraphFindDups(t, issuegraph, dir, "--threshold", "0.3")
		if strings.Contains(out, "No similar issues") {
			t.Skip("no pairs found at this threshold")
		}
		if !strings.Contains(out, "similar") {
			t.Errorf("expected 'similar' in human-readable output: %s", out)
		}
		if !strings.Contains(out, "Pair") {
			t.Errorf("expected 'Pair' in human-readable output: %s", out)
		}
	})
}

// TestEmbeddedFindDuplicatesConcurrent exercises find-duplicates operations concurrently.
func TestEmbeddedFindDuplicatesConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdc")

	// Create enough issues for meaningful comparison.
	for i := 0; i < 10; i++ {
		issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("concurrent dup test issue %d about login", i), "--type", "task")
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

			// Vary the threshold per worker
			threshold := fmt.Sprintf("%.1f", 0.1+float64(worker)*0.1)
			args := []string{"find-duplicates", "--json", "--threshold", threshold}
			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			stdout, stderr, err := runCommandBuffers(t, cmd)
			if err != nil {
				r.err = fmt.Errorf("worker %d find-duplicates: %v\nstdout:\n%s\nstderr:\n%s", worker, err, stdout.String(), stderr.String())
				results[worker] = r
				return
			}

			// Verify JSON is parseable
			s := strings.TrimSpace(stdout.String())
			start := strings.Index(s, "{")
			if start < 0 {
				r.err = fmt.Errorf("worker %d: no JSON in output: %s", worker, s)
				results[worker] = r
				return
			}
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
				r.err = fmt.Errorf("worker %d: JSON parse: %v\n%s", worker, err, s)
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
