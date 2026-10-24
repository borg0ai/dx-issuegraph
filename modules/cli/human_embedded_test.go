//go:build cgo

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// issuegraphHuman runs "issuegraph human" with the given args and returns stdout.
func issuegraphHuman(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"human"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph human %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// createHumanIssue creates a issue carrying the 'human' label and returns its ID.
func createHumanIssue(t *testing.T, issuegraph, dir, title string) string {
	t.Helper()
	id := issuegraphCreateSilent(t, issuegraph, dir, title, "--labels", "human")
	if id == "" {
		t.Fatalf("could not find issue ID in create output for %q", title)
	}
	return id
}

// humanShowClosed asserts the issue is closed and returns its parsed issue.
func humanShowClosed(t *testing.T, issuegraph, dir, id string) *types.Issue {
	t.Helper()
	issue := issuegraphShow(t, issuegraph, dir, id)
	if issue.Status != types.StatusClosed {
		t.Errorf("expected issue %s to be closed, got status %q", id, issue.Status)
	}
	return issue
}

func TestEmbeddedHuman(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "th")

	// ===== Default Help Output =====

	t.Run("human_default", func(t *testing.T) {
		out := issuegraphHuman(t, issuegraph, dir)
		if len(strings.TrimSpace(out)) == 0 {
			t.Error("expected non-empty human output")
		}
	})

	// ===== List =====

	t.Run("human_list_empty", func(t *testing.T) {
		out := issuegraphHuman(t, issuegraph, dir, "list")
		// No human-labeled issues yet — should succeed without error
		_ = out
	})

	// ===== Stats =====

	t.Run("human_stats", func(t *testing.T) {
		out := issuegraphHuman(t, issuegraph, dir, "stats")
		// Should succeed and produce output
		if len(strings.TrimSpace(out)) == 0 {
			t.Error("expected non-empty stats output")
		}
	})

	// ===== Respond and Dismiss =====

	t.Run("human_respond_and_dismiss", func(t *testing.T) {
		id := createHumanIssue(t, issuegraph, dir, "Human test issue")

		// Verify it shows up in human list
		listOut := issuegraphHuman(t, issuegraph, dir, "list")
		if !strings.Contains(listOut, id) {
			t.Errorf("expected issue %s in human list output:\n%s", id, listOut)
		}

		// Test Respond
		issuegraphHuman(t, issuegraph, dir, "respond", id, "--response", "Approved")
		humanShowClosed(t, issuegraph, dir, id)

		// Closed issuegraph drop out of the default list but show with
		// --status=closed and --status=all.
		listOut = issuegraphHuman(t, issuegraph, dir, "list")
		if strings.Contains(listOut, id) {
			t.Errorf("closed issue %s should be hidden from default human list:\n%s", id, listOut)
		}
		closedOut := issuegraphHuman(t, issuegraph, dir, "list", "--status=closed")
		if !strings.Contains(closedOut, id) {
			t.Errorf("expected closed issue %s in human list --status=closed:\n%s", id, closedOut)
		}
		allOut := issuegraphHuman(t, issuegraph, dir, "list", "--status=all")
		if !strings.Contains(allOut, id) {
			t.Errorf("expected closed issue %s in human list --status=all:\n%s", id, allOut)
		}

		// An invalid status is an error, not a silent empty list.
		if out, _ := issuegraphRunFailCode(t, issuegraph, dir, "human", "list", "--status=colsed"); !strings.Contains(out, "invalid status") {
			t.Errorf("expected invalid-status error, got:\n%s", out)
		}

		// Test Dismiss
		id2 := createHumanIssue(t, issuegraph, dir, "Dismiss test issue")
		issuegraphHuman(t, issuegraph, dir, "dismiss", id2, "--reason", "Not needed")
		issue2 := humanShowClosed(t, issuegraph, dir, id2)
		if issue2.CloseReason != "Dismissed: Not needed" {
			t.Errorf("expected dismiss reason %q, got %q", "Dismissed: Not needed", issue2.CloseReason)
		}
	})

	// ===== Respond via positional text and --file =====

	t.Run("human_respond_positional_and_file", func(t *testing.T) {
		// Positional response text, multiple words without a flag.
		id := createHumanIssue(t, issuegraph, dir, "Positional respond test")
		issuegraphHuman(t, issuegraph, dir, "respond", id, "Approved,", "proceed", "with", "implementation")
		humanShowClosed(t, issuegraph, dir, id)

		// Response text from a file.
		id2 := createHumanIssue(t, issuegraph, dir, "File respond test")
		respFile := filepath.Join(t.TempDir(), "response.md")
		if err := os.WriteFile(respFile, []byte("Approved via file\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		issuegraphHuman(t, issuegraph, dir, "respond", id2, "--file", respFile)
		humanShowClosed(t, issuegraph, dir, id2)

		// Positional dismiss reason.
		id3 := createHumanIssue(t, issuegraph, dir, "Positional dismiss test")
		issuegraphHuman(t, issuegraph, dir, "dismiss", id3, "No", "longer", "applicable")
		issue3 := humanShowClosed(t, issuegraph, dir, id3)
		if issue3.CloseReason != "Dismissed: No longer applicable" {
			t.Errorf("expected dismiss reason %q, got %q", "Dismissed: No longer applicable", issue3.CloseReason)
		}
	})

	// ===== Guard rails: conflicting sources and empty text =====

	t.Run("human_respond_dismiss_guards", func(t *testing.T) {
		id := createHumanIssue(t, issuegraph, dir, "Guard test issue")

		// Positional text combined with a flag source is a conflict, not a
		// silent drop of the positional text.
		if out, _ := issuegraphRunFailCode(t, issuegraph, dir, "human", "respond", id, "typed", "text", "--response", "flag text"); !strings.Contains(out, "cannot combine positional text") {
			t.Errorf("expected positional/flag conflict error, got:\n%s", out)
		}
		if out, _ := issuegraphRunFailCode(t, issuegraph, dir, "human", "dismiss", id, "typed", "text", "--reason", "flag text"); !strings.Contains(out, "cannot combine positional text") {
			t.Errorf("expected positional/flag conflict error, got:\n%s", out)
		}

		// Whitespace-only response must not close the issue.
		if _, code := issuegraphRunFailCode(t, issuegraph, dir, "human", "respond", id, "   "); code == 0 {
			t.Error("whitespace-only response should fail")
		}

		// ID-shaped free text is accepted as a dismiss reason like any other
		// text: respond/dismiss act on exactly one issue, args[0].
		idText := createHumanIssue(t, issuegraph, dir, "Shaped-text issue")
		issuegraphHuman(t, issuegraph, dir, "dismiss", idText, "bd-dev")
		issueText := humanShowClosed(t, issuegraph, dir, idText)
		if issueText.CloseReason != "Dismissed: bd-dev" {
			t.Errorf("expected ID-shaped free text kept as dismiss reason, got %q", issueText.CloseReason)
		}

		// The issue targeted by the rejected invocations is untouched.
		if issue := issuegraphShow(t, issuegraph, dir, id); issue.Status == types.StatusClosed {
			t.Errorf("rejected commands must not close the issue, got status %q", issue.Status)
		}
	})
}

// TestEmbeddedHumanConcurrent exercises human operations concurrently.
func TestEmbeddedHumanConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "hx")

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
			switch worker % 2 {
			case 0:
				args = []string{"human", "list"}
			case 1:
				args = []string{"human", "stats"}
			}
			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("human (worker %d): %v\n%s", worker, err, out)
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
