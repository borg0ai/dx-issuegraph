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
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// issuegraphImport runs "issuegraph import" with extra args. Returns combined output.
// issuegraph import writes status lines like "Imported N issues" to stderr
// (see apps/cli/import.go), so callers grepping for those need both streams.
func issuegraphImport(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"import"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph import %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String() + stderr.String()
}

// writeJSONLFile writes issues as JSONL to the given path.
func writeJSONLFile(t *testing.T, path string, issues []types.Issue) {
	t.Helper()
	var lines []string
	for _, issue := range issues {
		b, err := json.Marshal(issue)
		if err != nil {
			t.Fatalf("marshal issue: %v", err)
		}
		lines = append(lines, string(b))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("write JSONL: %v", err)
	}
}

func TestEmbeddedImport(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt import tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("from_explicit_file", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "imfile")

		// Create a JSONL file with test issues
		jsonlPath := filepath.Join(t.TempDir(), "import.jsonl")
		now := time.Now().UTC()
		writeJSONLFile(t, jsonlPath, []types.Issue{
			{ID: "imfile-aaa", Title: "Import A", Status: types.StatusOpen, IssueType: types.TypeTask, CreatedAt: now, UpdatedAt: now},
			{ID: "imfile-bbb", Title: "Import B", Status: types.StatusOpen, IssueType: types.TypeBug, CreatedAt: now, UpdatedAt: now},
		})

		out := issuegraphImport(t, issuegraph, dir, jsonlPath)
		if !strings.Contains(out, "Imported 2 issues") {
			t.Errorf("expected 'Imported 2 issues', got: %s", out)
		}
	})

	t.Run("from_default_path", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "imdef")

		// Write issues to the default .issuegraph/issues.jsonl location
		jsonlPath := filepath.Join(dir, ".issuegraph", "issues.jsonl")
		now := time.Now().UTC()
		writeJSONLFile(t, jsonlPath, []types.Issue{
			{ID: "imdef-xxx", Title: "Default Path Issue", Status: types.StatusOpen, IssueType: types.TypeTask, CreatedAt: now, UpdatedAt: now},
		})

		out := issuegraphImport(t, issuegraph, dir)
		if !strings.Contains(out, "Imported 1 issue") {
			t.Errorf("expected 'Imported 1 issue', got: %s", out)
		}
	})

	t.Run("from_configured_import_path", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "imcfg")

		cmd := exec.Command(issuegraph, "config", "set", "import.path", "issuegraph.jsonl")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("issuegraph config set import.path failed: %v\n%s", err, out)
		}

		jsonlPath := filepath.Join(dir, ".issuegraph", "issuegraph.jsonl")
		now := time.Now().UTC()
		writeJSONLFile(t, jsonlPath, []types.Issue{
			{ID: "imcfg-xxx", Title: "Configured Import Path Issue", Status: types.StatusOpen, IssueType: types.TypeTask, CreatedAt: now, UpdatedAt: now},
		})

		out := issuegraphImport(t, issuegraph, dir)
		if !strings.Contains(out, "Imported 1 issue") {
			t.Errorf("expected 'Imported 1 issue', got: %s", out)
		}
	})

	t.Run("dry_run", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "imdry")

		jsonlPath := filepath.Join(dir, ".issuegraph", "issues.jsonl")
		now := time.Now().UTC()
		writeJSONLFile(t, jsonlPath, []types.Issue{
			{ID: "imdry-qqq", Title: "Dry Run Issue", Status: types.StatusOpen, IssueType: types.TypeTask, CreatedAt: now, UpdatedAt: now},
		})

		out := issuegraphImport(t, issuegraph, dir, "--dry-run")
		if !strings.Contains(out, "Would import") {
			t.Errorf("expected 'Would import' in dry-run output, got: %s", out)
		}

		// Verify issue was NOT actually created
		cmd := exec.Command(issuegraph, "show", "imdry-qqq", "--json")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if showOut, err := cmd.CombinedOutput(); err == nil {
			t.Errorf("issue should not exist after dry-run, but show succeeded: %s", showOut)
		}
	})

	t.Run("with_memories", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "immem")

		// Create a JSONL file with an issue and a memory record
		jsonlPath := filepath.Join(t.TempDir(), "import-memories.jsonl")
		now := time.Now().UTC()
		issue := types.Issue{
			ID: "immem-aaa", Title: "Memory Test Issue",
			Status: types.StatusOpen, IssueType: types.TypeTask,
			CreatedAt: now, UpdatedAt: now,
		}
		issueLine, _ := json.Marshal(issue)
		memoryLine, _ := json.Marshal(map[string]string{
			"_type": "memory",
			"key":   "test-key",
			"value": "test-value",
		})
		content := string(issueLine) + "\n" + string(memoryLine) + "\n"
		if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		out := issuegraphImport(t, issuegraph, dir, jsonlPath)
		if !strings.Contains(out, "1 issue") {
			t.Errorf("expected '1 issue' in output, got: %s", out)
		}
		if !strings.Contains(out, "1 memor") {
			t.Errorf("expected '1 memor' in output, got: %s", out)
		}
	})

	t.Run("empty_file", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "imemp")

		jsonlPath := filepath.Join(t.TempDir(), "empty.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		out := issuegraphImport(t, issuegraph, dir, jsonlPath)
		if !strings.Contains(out, "Empty file") {
			t.Errorf("expected 'Empty file' message, got: %s", out)
		}
	})

	t.Run("upsert_existing", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "imups")

		// Create an issue via issuegraph create
		id := issuegraphCreateSilent(t, issuegraph, dir, "Original Title")

		// Import with updated title for the same ID. UpdatedAt must be
		// strictly newer than the created row's second-granularity
		// updated_at: equal-timestamp rows keep the local row (bd-hj85c).
		jsonlPath := filepath.Join(t.TempDir(), "upsert.jsonl")
		now := time.Now().UTC().Add(time.Hour)
		writeJSONLFile(t, jsonlPath, []types.Issue{
			{ID: id, Title: "Updated Title", Status: types.StatusOpen, IssueType: types.TypeTask, CreatedAt: now, UpdatedAt: now},
		})

		issuegraphImport(t, issuegraph, dir, jsonlPath)

		// Verify the title was updated
		showCmd := exec.Command(issuegraph, "show", id, "--json")
		showCmd.Dir = dir
		showCmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, showCmd)
		if err != nil {
			t.Fatalf("issuegraph show %s failed: %v\nstdout:\n%s\nstderr:\n%s", id, err, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), "Updated Title") {
			t.Errorf("expected 'Updated Title' after upsert, got: %s", stdout.String())
		}
	})

	t.Run("prefix_sync", func(t *testing.T) {
		// Simulate a stale DB: init with --prefix issuegraph (DB has issue_prefix=issuegraph),
		// then overwrite config.yaml with issue-prefix: be. issuegraph import must sync
		// the DB to match config.yaml (be-llaf).
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "issuegraph")

		if err := os.WriteFile(filepath.Join(issuegraphDir, "config.yaml"), []byte("issue-prefix: be\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		jsonlPath := filepath.Join(t.TempDir(), "prefix-sync.jsonl")
		now := time.Now().UTC()
		writeJSONLFile(t, jsonlPath, []types.Issue{
			{ID: "bd-sync1", Title: "Prefix Sync Issue", Status: types.StatusOpen, IssueType: types.TypeTask, CreatedAt: now, UpdatedAt: now},
		})

		issuegraphImport(t, issuegraph, dir, jsonlPath)

		if val := readBack(t, issuegraphDir, "issuegraph", "issue_prefix", false); val != "be" {
			t.Errorf("issue_prefix after import: got %q, want %q", val, "be")
		}
	})

	t.Run("prefix_sync_no_config_key", func(t *testing.T) {
		// config.yaml with no issue-prefix key must not touch the DB's
		// issue_prefix; the sync block only fires when yamlPrefix != "" (be-llaf).
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "issuegraph")

		if err := os.WriteFile(filepath.Join(issuegraphDir, "config.yaml"), []byte("export:\n  auto: false\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		jsonlPath := filepath.Join(t.TempDir(), "prefix-sync-no-key.jsonl")
		now := time.Now().UTC()
		writeJSONLFile(t, jsonlPath, []types.Issue{
			{ID: "bd-nokey1", Title: "No Prefix Key Issue", Status: types.StatusOpen, IssueType: types.TypeTask, CreatedAt: now, UpdatedAt: now},
		})

		issuegraphImport(t, issuegraph, dir, jsonlPath)

		if val := readBack(t, issuegraphDir, "issuegraph", "issue_prefix", false); val != "issuegraph" {
			t.Errorf("issue_prefix after import: got %q, want %q", val, "issuegraph")
		}
	})
}

func TestEmbeddedImportConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt import tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "imconc")

	const numWorkers = 5

	type result struct {
		worker int
		out    string
		err    error
	}

	results := make([]result, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			// Each worker imports its own JSONL file with unique IDs
			jsonlPath := filepath.Join(t.TempDir(), fmt.Sprintf("import-%d.jsonl", worker))
			now := time.Now().UTC()
			var issues []types.Issue
			for i := 0; i < 3; i++ {
				issues = append(issues, types.Issue{
					ID:        fmt.Sprintf("imconc-w%d-%d", worker, i),
					Title:     fmt.Sprintf("Worker %d Issue %d", worker, i),
					Status:    types.StatusOpen,
					IssueType: types.TypeTask,
					CreatedAt: now,
					UpdatedAt: now,
				})
			}
			var lines []string
			for _, issue := range issues {
				b, _ := json.Marshal(issue)
				lines = append(lines, string(b))
			}
			_ = os.WriteFile(jsonlPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)

			cmd := exec.Command(issuegraph, "import", jsonlPath)
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
		} else if strings.Contains(r.out, "one writer at a time") ||
			strings.Contains(r.out, "nothing to commit") {
			// Expected concurrent contention errors
		} else {
			t.Errorf("worker %d failed with unexpected error: %v\n%s", r.worker, r.err, r.out)
		}
	}
	if successes < 1 {
		t.Errorf("expected at least 1 successful import, got %d", successes)
	}
	t.Logf("%d/%d import workers succeeded", successes, numWorkers)
}
