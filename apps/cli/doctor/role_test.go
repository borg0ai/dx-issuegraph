package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckIssueGraphRole_NotConfigured(t *testing.T) {
	// Create a temp directory with git init but no issuegraph.role config
	tmpDir := newGitRepo(t)

	// Check role - should return warning since not configured
	check := CheckIssueGraphRole(tmpDir)

	if check.Status != StatusWarning {
		t.Errorf("expected status %s, got %s", StatusWarning, check.Status)
	}
	if check.Name != "Role Configuration" {
		t.Errorf("expected name 'Role Configuration', got %q", check.Name)
	}
	if check.Fix != "git config issuegraph.role maintainer" {
		t.Errorf("expected fix 'git config issuegraph.role maintainer', got %q", check.Fix)
	}
}

func TestCheckIssueGraphRole_Maintainer(t *testing.T) {
	tmpDir := newGitRepo(t)

	// Set issuegraph.role to maintainer
	cmd := exec.Command("git", "config", "issuegraph.role", "maintainer")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}

	check := CheckIssueGraphRole(tmpDir)

	if check.Status != StatusOK {
		t.Errorf("expected status %s, got %s", StatusOK, check.Status)
	}
	if check.Message != "Configured as maintainer" {
		t.Errorf("expected message 'Configured as maintainer', got %q", check.Message)
	}
}

func TestCheckIssueGraphRole_Contributor(t *testing.T) {
	tmpDir := newGitRepo(t)

	// Set issuegraph.role to contributor
	cmd := exec.Command("git", "config", "issuegraph.role", "contributor")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}

	check := CheckIssueGraphRole(tmpDir)

	if check.Status != StatusOK {
		t.Errorf("expected status %s, got %s", StatusOK, check.Status)
	}
	if check.Message != "Configured as contributor" {
		t.Errorf("expected message 'Configured as contributor', got %q", check.Message)
	}
}

func TestCheckIssueGraphRole_InvalidValue(t *testing.T) {
	tmpDir := newGitRepo(t)

	// Set issuegraph.role to an invalid value
	cmd := exec.Command("git", "config", "issuegraph.role", "admin")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}

	check := CheckIssueGraphRole(tmpDir)

	if check.Status != StatusWarning {
		t.Errorf("expected status %s, got %s", StatusWarning, check.Status)
	}
	if check.Fix != "issuegraph init" {
		t.Errorf("expected fix 'issuegraph init', got %q", check.Fix)
	}
}

func TestCheckIssueGraphRole_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-role-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Don't initialize git - just a plain directory
	check := CheckIssueGraphRole(tmpDir)

	// Should return OK/N/A since we're not in a git repo — the role may
	// be correctly configured in a worktree (e.g., rig roots use .repo.git).
	if check.Status != StatusOK {
		t.Errorf("expected status %s, got %s", StatusOK, check.Status)
	}
	if check.Message != "N/A (not a git repository)" {
		t.Errorf("expected message 'N/A (not a git repository)', got %q", check.Message)
	}
}

func TestCheckIssueGraphRole_NonexistentPath(t *testing.T) {
	// Test with a path that doesn't exist — git will report "not a git repository"
	check := CheckIssueGraphRole(filepath.Join(os.TempDir(), "nonexistent-issuegraph-test-dir"))

	// Should return OK/N/A since the path is not a git repository
	if check.Status != StatusOK {
		t.Errorf("expected status %s, got %s", StatusOK, check.Status)
	}
}
