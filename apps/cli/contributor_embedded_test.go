//go:build cgo

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/storage/embeddeddolt"
)

// TestEmbeddedContributorCreate exercises the full contributor mode flow in
// embedded Dolt: init → contributor wizard → create. This is the exact
// scenario that triggered GH#2988 ("no database selected" when the planning
// repo's .issuegraph directory has no metadata.json).
func TestEmbeddedContributorCreate(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt contributor tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("create_routes_to_planning_repo", func(t *testing.T) {
		dir, planningDir := initContributor(t, issuegraph, "cr")

		// Create an issue — should route to the planning repo, not the project.
		issue := issuegraphCreate(t, issuegraph, dir, "Contributor issue")
		if issue.ID == "" {
			t.Fatal("expected issue ID")
		}
		if !strings.HasPrefix(issue.ID, "cr-") {
			t.Errorf("ID should have prefix cr-, got %q", issue.ID)
		}

		// Verify the issue landed in the planning repo's embedded store.
		// The planning store database name is the sanitized prefix (same as source).
		planningIssueGraphDir := filepath.Join(planningDir, ".issuegraph")
		assertIssueInStore(t, planningIssueGraphDir, "cr", issue.ID)

		// Verify the issue is NOT in the project's store.
		projectIssueGraphDir := filepath.Join(dir, ".issuegraph")
		assertIssueNotInStore(t, projectIssueGraphDir, "cr", issue.ID)
	})

	t.Run("show_reads_routed_issue", func(t *testing.T) {
		dir, _ := initContributor(t, issuegraph, "sh")

		issue := issuegraphCreate(t, issuegraph, dir, "Show me")
		shown := issuegraphShow(t, issuegraph, dir, issue.ID)
		if shown.Title != "Show me" {
			t.Errorf("title: got %q, want %q", shown.Title, "Show me")
		}
	})

	t.Run("multiple_creates_succeed", func(t *testing.T) {
		dir, planningDir := initContributor(t, issuegraph, "mc")

		ids := make(map[string]bool)
		for i := 0; i < 3; i++ {
			issue := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("Issue %d", i))
			if issue.ID == "" {
				t.Fatalf("create %d: expected issue ID", i)
			}
			if ids[issue.ID] {
				t.Fatalf("create %d: duplicate ID %q", i, issue.ID)
			}
			ids[issue.ID] = true
		}

		// All three should be in the planning store.
		planningIssueGraphDir := filepath.Join(planningDir, ".issuegraph")
		for id := range ids {
			assertIssueInStore(t, planningIssueGraphDir, "mc", id)
		}
	})
}

// initContributor sets up a project with contributor mode enabled and returns
// the project dir and planning dir paths. The contributor wizard runs via
// subprocess with stdin piped to answer the interactive prompts.
func initContributor(t *testing.T, issuegraph, prefix string) (projectDir, planningDir string) {
	t.Helper()

	projectDir = t.TempDir()
	planningDir = filepath.Join(t.TempDir(), "planning")
	initGitRepoAt(t, projectDir)

	// Run issuegraph init with --contributor flag.
	// The wizard prompts (no upstream remote):
	//   1. "Continue with contributor setup? [y/N]:" → "y"
	//   2. "Planning repo path [press Enter for default]:" → planningDir
	args := []string{"init", "--quiet", "--prefix", prefix, "--contributor"}
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = projectDir
	cmd.Env = append(issuegraphEnv(projectDir), "ISSUEGRAPH_NON_INTERACTIVE=0")
	cmd.Stdin = strings.NewReader("y\n" + planningDir + "\n")
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph init --contributor failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	// Sanity: planning .issuegraph dir should exist (created by wizard).
	planningIssueGraphDir := filepath.Join(planningDir, ".issuegraph")
	requireFile(t, planningIssueGraphDir)

	// Sanity: issuegraph.role should be "contributor".
	roleCmd := exec.Command("git", "config", "issuegraph.role")
	roleCmd.Dir = projectDir
	roleOut, err := roleCmd.Output()
	if err != nil {
		t.Fatalf("git config issuegraph.role failed: %v", err)
	}
	if got := strings.TrimSpace(string(roleOut)); got != "contributor" {
		t.Fatalf("issuegraph.role: got %q, want %q", got, "contributor")
	}

	return projectDir, planningDir
}

// assertIssueInStore verifies that an issue with the given ID exists in the
// embedded Dolt store at issuegraphDir.
func assertIssueInStore(t *testing.T, issuegraphDir, database, issueID string) {
	t.Helper()
	dataDir := filepath.Join(issuegraphDir, "embeddeddolt")
	db, cleanup, err := embeddeddolt.OpenSQL(t.Context(), dataDir, database, "main")
	if err != nil {
		t.Fatalf("OpenSQL(%s, %s): %v", issuegraphDir, database, err)
	}
	defer cleanup()

	var count int
	err = db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM issues WHERE id = ?", issueID).Scan(&count)
	if err != nil {
		t.Fatalf("query issues for %s: %v", issueID, err)
	}
	if count == 0 {
		t.Errorf("expected issue %s in store at %s, not found", issueID, issuegraphDir)
	}
}

// assertIssueNotInStore verifies that an issue with the given ID does NOT
// exist in the embedded Dolt store at issuegraphDir.
func assertIssueNotInStore(t *testing.T, issuegraphDir, database, issueID string) {
	t.Helper()
	dataDir := filepath.Join(issuegraphDir, "embeddeddolt")
	db, cleanup, err := embeddeddolt.OpenSQL(t.Context(), dataDir, database, "main")
	if err != nil {
		t.Fatalf("OpenSQL(%s, %s): %v", issuegraphDir, database, err)
	}
	defer cleanup()

	var count int
	err = db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM issues WHERE id = ?", issueID).Scan(&count)
	if err != nil {
		t.Fatalf("query issues for %s: %v", issueID, err)
	}
	if count != 0 {
		t.Errorf("issue %s should NOT be in store at %s, but found %d rows", issueID, issuegraphDir, count)
	}
}
