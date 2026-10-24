//go:build cgo

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// issuegraphDelete runs "issuegraph delete" with the given args and returns stdout.
// Retries on flock contention.
func issuegraphDelete(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"delete"}, args...)
	out, err := issuegraphRunWithFlockRetry(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph delete %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// issuegraphDeleteFail runs "issuegraph delete" expecting failure.
func issuegraphDeleteFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"delete"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph delete %s to fail, but it succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

// issuegraphShowFail runs "issuegraph show" expecting failure (e.g., deleted issue).
func issuegraphShowFail(t *testing.T, issuegraph, dir, id string) string {
	t.Helper()
	cmd := exec.Command(issuegraph, "show", id, "--json")
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph show %s to fail (deleted), but succeeded:\n%s", id, out)
	}
	return string(out)
}

func TestEmbeddedDelete(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "td")

	t.Run("delete_single_issue", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Delete me", "--type", "task")
		issuegraphDelete(t, issuegraph, dir, issue.ID, "--force")
		issuegraphShowFail(t, issuegraph, dir, issue.ID)
	})

	t.Run("delete_cleans_up_dependencies", func(t *testing.T) {
		parent := issuegraphCreate(t, issuegraph, dir, "Parent", "--type", "task")
		child := issuegraphCreate(t, issuegraph, dir, "Child", "--type", "task")
		issuegraphDepAdd(t, issuegraph, dir, child.ID, parent.ID)

		// Delete child; parent should survive.
		issuegraphDelete(t, issuegraph, dir, child.ID, "--force")
		issuegraphShowFail(t, issuegraph, dir, child.ID)
		got := issuegraphShow(t, issuegraph, dir, parent.ID)
		if got.Status == types.StatusClosed {
			t.Error("expected parent to still be open")
		}
	})

	t.Run("delete_without_force_shows_preview", func(t *testing.T) {
		target := issuegraphCreate(t, issuegraph, dir, "Lonely", "--type", "task")

		// Without --force, issuegraph delete shows a preview (exits 0) but does not delete.
		out := issuegraphDelete(t, issuegraph, dir, target.ID)
		if !strings.Contains(out, "PREVIEW") && !strings.Contains(out, "preview") {
			t.Logf("expected preview output: %s", out)
		}
		got := issuegraphShow(t, issuegraph, dir, target.ID)
		if got.ID != target.ID {
			t.Errorf("expected the target to still exist after preview")
		}
	})

	// The one-id preview over a issue with an OUTSIDE dependent now refuses
	// rather than exiting 0, which is a change and is the direct route
	// converging with itself: the batch path and `--dry-run` have always
	// refused here (see TestEmbeddedDeleteJSONDependencyErrorContract, which
	// runs both), and only the unconfirmed single-id preview did not. The
	// refusal is the role's, so both routes and all three of this command's
	// paths now give the same answer to the same question.
	t.Run("delete_without_force_refuses_over_an_outside_dependent", func(t *testing.T) {
		parent := issuegraphCreate(t, issuegraph, dir, "Parent strict", "--type", "task")
		child := issuegraphCreate(t, issuegraph, dir, "Child strict", "--type", "task")
		issuegraphDepAdd(t, issuegraph, dir, child.ID, parent.ID)

		out := issuegraphDeleteFail(t, issuegraph, dir, parent.ID)
		if !strings.Contains(out, "dependents not in deletion set") {
			t.Errorf("refusal did not name the guard: %s", out)
		}
		if !strings.Contains(out, "--cascade") || !strings.Contains(out, "--force") {
			t.Errorf("refusal did not say what to send instead: %s", out)
		}
		// Nothing was deleted, on either end of the edge.
		if got := issuegraphShow(t, issuegraph, dir, parent.ID); got.ID != parent.ID {
			t.Errorf("expected the parent to survive a refused delete")
		}
		if got := issuegraphShow(t, issuegraph, dir, child.ID); got.ID != child.ID {
			t.Errorf("expected the dependent to survive a refused delete")
		}
	})

	t.Run("delete_single_quiet_forced_dry_run_is_payload_blind", func(t *testing.T) {
		titleMarker := "EMBEDDED_QUIET_TITLE_MARKER"
		descriptionMarker := "EMBEDDED_QUIET_DESCRIPTION_MARKER"
		notesMarker := "EMBEDDED_QUIET_NOTES_MARKER"
		payloadMarker := "EMBEDDED_QUIET_PAYLOAD_MARKER"
		parent := issuegraphCreate(t, issuegraph, dir, titleMarker,
			"--type", "task",
			"--description", descriptionMarker,
			"--notes", notesMarker,
			"--metadata", `{"marker":"`+payloadMarker+`"}`)
		child := issuegraphCreate(t, issuegraph, dir, "Embedded quiet dependent", "--type", "task")
		issuegraphDepAdd(t, issuegraph, dir, child.ID, parent.ID)

		cmd := exec.Command(issuegraph, "delete", parent.ID, "--force", "--dry-run", "--quiet")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("single forced quiet dry-run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		if stdout.Len() != 0 {
			t.Fatalf("quiet dry-run produced stdout: %s", stdout.String())
		}
		combined := stdout.String() + stderr.String()
		for _, marker := range []string{titleMarker, descriptionMarker, notesMarker, payloadMarker} {
			if strings.Contains(combined, marker) {
				t.Fatalf("quiet dry-run leaked %q: %s", marker, combined)
			}
		}
		if got := issuegraphShowDetails(t, issuegraph, dir, parent.ID); got["id"] != parent.ID {
			t.Fatalf("forced dry-run removed parent: got %v, want %q", got["id"], parent.ID)
		}
		if got := issuegraphShowDetails(t, issuegraph, dir, child.ID); got["id"] != child.ID {
			t.Fatalf("forced dry-run removed dependent: got %v, want %q", got["id"], child.ID)
		}
	})

	t.Run("delete_force_orphans_dependents", func(t *testing.T) {
		parent := issuegraphCreate(t, issuegraph, dir, "Force parent", "--type", "task")
		child := issuegraphCreate(t, issuegraph, dir, "Force child", "--type", "task")
		issuegraphDepAdd(t, issuegraph, dir, child.ID, parent.ID)

		issuegraphDelete(t, issuegraph, dir, parent.ID, "--force")
		issuegraphShowFail(t, issuegraph, dir, parent.ID)
		// Child should still exist (orphaned).
		got := issuegraphShow(t, issuegraph, dir, child.ID)
		if got.ID != child.ID {
			t.Errorf("expected orphaned child to survive, got %s", got.ID)
		}
	})

	t.Run("delete_batch", func(t *testing.T) {
		issue1 := issuegraphCreate(t, issuegraph, dir, "Batch 1", "--type", "task")
		issue2 := issuegraphCreate(t, issuegraph, dir, "Batch 2", "--type", "task")
		issue3 := issuegraphCreate(t, issuegraph, dir, "Batch 3", "--type", "task")

		issuegraphDelete(t, issuegraph, dir, issue1.ID, issue2.ID, issue3.ID, "--force")
		issuegraphShowFail(t, issuegraph, dir, issue1.ID)
		issuegraphShowFail(t, issuegraph, dir, issue2.ID)
		issuegraphShowFail(t, issuegraph, dir, issue3.ID)
	})

	t.Run("delete_nonexistent", func(t *testing.T) {
		issuegraphDeleteFail(t, issuegraph, dir, "td-nonexistent999", "--force")
	})
}

func TestEmbeddedDeleteJSONDependencyErrorContract(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "jde")

	for _, tc := range []struct {
		name  string
		batch bool
	}{
		{name: "single"},
		{name: "batch", batch: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			titleMarker := "EMBEDDED_JSON_ERROR_TITLE_" + tc.name
			descriptionMarker := "EMBEDDED_JSON_ERROR_DESCRIPTION_" + tc.name
			notesMarker := "EMBEDDED_JSON_ERROR_NOTES_" + tc.name
			metadataMarker := "EMBEDDED_JSON_ERROR_METADATA_" + tc.name
			parent := issuegraphCreate(t, issuegraph, dir, titleMarker,
				"--type", "task",
				"--description", descriptionMarker,
				"--notes", notesMarker,
				"--metadata", `{"marker":"`+metadataMarker+`"}`)
			child := issuegraphCreate(t, issuegraph, dir, "JSON error dependent "+tc.name, "--type", "task")
			issuegraphDepAdd(t, issuegraph, dir, child.ID, parent.ID)

			issueIDs := []string{parent.ID}
			if tc.batch {
				other := issuegraphCreate(t, issuegraph, dir, "JSON error batch companion", "--type", "task")
				issueIDs = append(issueIDs, other.ID)
			}

			runDelete := func(flags ...string) (string, string, error) {
				args := append([]string{"delete"}, issueIDs...)
				args = append(args, flags...)
				cmd := exec.Command(issuegraph, args...)
				cmd.Dir = dir
				cmd.Env = append(issuegraphEnv(dir), "ISSUEGRAPH_JSON_ENVELOPE=0")
				stdout, stderr, err := runCommandBuffers(t, cmd)
				return stdout.String(), stderr.String(), err
			}

			stdout, stderr, err := runDelete("--dry-run", "--json")
			if err == nil {
				t.Fatalf("unforced dependency-blocked JSON dry-run succeeded\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
			}
			var preview map[string]interface{}
			if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
				t.Fatalf("unforced preview stdout is not exactly one JSON object: %v\nstdout:\n%s", err, stdout)
			}
			if preview["preview"] != true || preview["dry_run"] != true {
				t.Fatalf("unforced preview missing structural fields: %#v", preview)
			}
			if _, ok := preview["error"].(string); !ok {
				t.Fatalf("unforced preview missing dependency error: %#v", preview)
			}
			var jsonErr map[string]interface{}
			if err := json.Unmarshal([]byte(stderr), &jsonErr); err != nil {
				t.Fatalf("unforced stderr is not exactly one JSON error object: %v\nstderr:\n%s", err, stderr)
			}
			if _, ok := jsonErr["error"].(string); !ok {
				t.Fatalf("unforced stderr missing error field: %#v", jsonErr)
			}
			if strings.Contains(stderr, "Error:") {
				t.Fatalf("unforced stderr contains plaintext error prefix: %s", stderr)
			}
			for _, marker := range []string{titleMarker, descriptionMarker, notesMarker, metadataMarker} {
				if strings.Contains(stdout+stderr, marker) {
					t.Fatalf("unforced JSON dry-run leaked %q", marker)
				}
			}

			stdout, stderr, err = runDelete("--force", "--dry-run", "--quiet", "--json")
			if err != nil {
				t.Fatalf("forced quiet JSON dry-run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
			}
			if strings.TrimSpace(stderr) != "" {
				t.Fatalf("forced quiet JSON dry-run wrote stderr: %s", stderr)
			}
			var forcedPreview map[string]interface{}
			if err := json.Unmarshal([]byte(stdout), &forcedPreview); err != nil {
				t.Fatalf("forced quiet JSON stdout is not exactly one JSON object: %v\nstdout:\n%s", err, stdout)
			}
			for _, key := range []string{"preview", "dry_run", "would_delete", "would_orphan"} {
				if _, ok := forcedPreview[key]; !ok {
					t.Fatalf("forced quiet JSON preview missing %q: %#v", key, forcedPreview)
				}
			}
			for _, marker := range []string{titleMarker, descriptionMarker, notesMarker, metadataMarker} {
				if strings.Contains(stdout, marker) {
					t.Fatalf("forced quiet JSON dry-run leaked %q", marker)
				}
			}

			if got := issuegraphShowDetails(t, issuegraph, dir, parent.ID); got["id"] != parent.ID {
				t.Fatalf("dry-run removed parent: got %v, want %q", got["id"], parent.ID)
			}
			if got := issuegraphShowDetails(t, issuegraph, dir, child.ID); got["id"] != child.ID {
				t.Fatalf("dry-run removed dependent: got %v, want %q", got["id"], child.ID)
			}
			assertDepExists(t, issuegraphDir, "jde", child.ID, parent.ID)
		})
	}
}

func TestEmbeddedGetDependencies(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "gd")

	parent := issuegraphCreate(t, issuegraph, dir, "Parent", "--type", "task")
	child := issuegraphCreate(t, issuegraph, dir, "Child", "--type", "task")
	issuegraphDepAdd(t, issuegraph, dir, child.ID, parent.ID)

	store := openStore(t, issuegraphDir, "gd")

	t.Run("get_dependencies", func(t *testing.T) {
		deps, err := store.GetDependencies(t.Context(), child.ID)
		if err != nil {
			t.Fatalf("GetDependencies: %v", err)
		}
		if len(deps) != 1 {
			t.Fatalf("expected 1 dependency, got %d", len(deps))
		}
		if deps[0].ID != parent.ID {
			t.Errorf("expected dependency on %s, got %s", parent.ID, deps[0].ID)
		}
	})

	t.Run("get_dependents", func(t *testing.T) {
		deps, err := store.GetDependents(t.Context(), parent.ID)
		if err != nil {
			t.Fatalf("GetDependents: %v", err)
		}
		if len(deps) != 1 {
			t.Fatalf("expected 1 dependent, got %d", len(deps))
		}
		if deps[0].ID != child.ID {
			t.Errorf("expected dependent %s, got %s", child.ID, deps[0].ID)
		}
	})

	t.Run("get_dependencies_empty", func(t *testing.T) {
		deps, err := store.GetDependencies(t.Context(), parent.ID)
		if err != nil {
			t.Fatalf("GetDependencies: %v", err)
		}
		if len(deps) != 0 {
			t.Errorf("expected 0 dependencies for parent, got %d", len(deps))
		}
	})
}
