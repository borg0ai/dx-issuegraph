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

func TestEmbeddedStatus(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ss")

	issuegraphCreate(t, issuegraph, dir, "Status open 1", "--type", "task")
	issuegraphCreate(t, issuegraph, dir, "Status open 2", "--type", "bug")
	ip := issuegraphCreate(t, issuegraph, dir, "Status in_progress", "--type", "task", "--assignee", "alice")
	issuegraphUpdate(t, issuegraph, dir, ip.ID, "--status", "in_progress")
	closed := issuegraphCreate(t, issuegraph, dir, "Status closed", "--type", "task")
	issuegraphClose(t, issuegraph, dir, closed.ID)
	issuegraphCreate(t, issuegraph, dir, "Status assigned bob", "--type", "task", "--assignee", "bob")

	runStatus := func(t *testing.T, env []string, actor string, args ...string) []byte {
		t.Helper()
		commandArgs := append([]string{"status"}, args...)
		if actor != "" {
			commandArgs = append([]string{"--actor", actor}, commandArgs...)
		}
		cmd := exec.Command(issuegraph, commandArgs...)
		cmd.Dir = dir
		cmd.Env = env
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph status %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
		}
		return stdout.Bytes()
	}
	assertCount := func(t *testing.T, name string, got, want int) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %d, want %d", name, got, want)
		}
	}
	assertPointerCount := func(t *testing.T, name string, got *int, want int) {
		t.Helper()
		if got == nil {
			t.Errorf("%s = nil, want %d", name, want)
			return
		}
		assertCount(t, name, *got, want)
	}

	t.Run("known_counts_without_activity", func(t *testing.T) {
		out := runStatus(t, issuegraphEnv(dir), "", "--json", "--no-activity")
		var got StatusOutput
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal status: %v\n%s", err, out)
		}
		if got.Summary == nil {
			t.Fatalf("status summary is nil: %s", out)
		}
		assertCount(t, "total issues", got.Summary.TotalIssues, 5)
		assertCount(t, "open issues", got.Summary.OpenIssues, 3)
		assertCount(t, "in-progress issues", got.Summary.InProgressIssues, 1)
		assertCount(t, "closed issues", got.Summary.ClosedIssues, 1)
		assertCount(t, "deferred issues", got.Summary.DeferredIssues, 0)
		assertPointerCount(t, "blocked issues", got.Summary.BlockedIssues, 0)
		assertPointerCount(t, "ready issues", got.Summary.ReadyIssues, 3)
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(out, &envelope); err != nil {
			t.Fatalf("unmarshal status envelope: %v", err)
		}
		if _, ok := envelope["recent_activity"]; ok {
			t.Errorf("recent_activity present with --no-activity: %s", out)
		}
	})

	t.Run("assigned_counts_without_activity", func(t *testing.T) {
		out := runStatus(t, issuegraphEnv(dir), "alice", "--assigned", "--json", "--no-activity")
		var got StatusOutput
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal assigned status: %v\n%s", err, out)
		}
		if got.Summary == nil {
			t.Fatalf("assigned status summary is nil: %s", out)
		}
		assertCount(t, "assigned total issues", got.Summary.TotalIssues, 1)
		assertCount(t, "assigned open issues", got.Summary.OpenIssues, 0)
		assertCount(t, "assigned in-progress issues", got.Summary.InProgressIssues, 1)
		assertCount(t, "assigned closed issues", got.Summary.ClosedIssues, 0)
		assertCount(t, "assigned deferred issues", got.Summary.DeferredIssues, 0)
		assertPointerCount(t, "assigned blocked issues", got.Summary.BlockedIssues, 0)
		assertPointerCount(t, "assigned ready issues", got.Summary.ReadyIssues, 1)
	})
}

// TestEmbeddedStatusConcurrent exercises status operations concurrently.
func TestEmbeddedStatusConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ssc")

	for i := 0; i < 10; i++ {
		issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("concurrent-status-%d", i), "--type", "task")
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
				{"--json", "--no-activity"},
				{"--json"},
				{"--json", "--no-activity"},
				{"--json"},
				{"--json"},
				{"--json", "--no-activity"},
				{"--json"},
			}
			q := queries[worker%len(queries)]

			args := append([]string{"status"}, q...)
			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			stdout, stderr, err := runCommandBuffers(t, cmd)
			if err != nil {
				r.err = fmt.Errorf("worker %d status: %v\nstdout:\n%s\nstderr:\n%s", worker, err, stdout.String(), stderr.String())
				results[worker] = r
				return
			}

			// Verify JSON parses
			s := strings.TrimSpace(stdout.String())
			start := strings.Index(s, "{")
			if start < 0 {
				r.err = fmt.Errorf("worker %d: no JSON: %s", worker, s)
				results[worker] = r
				return
			}
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
				r.err = fmt.Errorf("worker %d: JSON parse: %v", worker, err)
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
