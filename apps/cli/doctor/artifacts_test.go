package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckClassicArtifacts_NoArtifacts(t *testing.T) {
	dir := t.TempDir()

	// Create a basic .issuegraph directory with no artifacts
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	check := CheckClassicArtifacts(dir)
	if check.Status != StatusOK {
		t.Errorf("expected StatusOK, got %s: %s", check.Status, check.Message)
	}
}

// TestScanForArtifacts_JSONLInDoltDir — JSONL artifact scanning removed (bd-9ni.2)
func TestScanForArtifacts_JSONLInDoltDir(t *testing.T) {
	t.Skip("JSONL artifact scanning removed as part of JSONL removal (bd-9ni.2)")
}

func TestScanForArtifacts_SQLiteFiles(t *testing.T) {
	dir := t.TempDir()

	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create dolt/ directory so isDoltNative returns true (SQLite files
	// are only flagged as artifacts when Dolt is the active backend)
	if err := os.MkdirAll(filepath.Join(issuegraphDir, "dolt"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create SQLite artifacts
	for _, name := range []string{"issuegraph.db", "issuegraph.db-shm", "issuegraph.db-wal", "issuegraph.backup-20260204.db"} {
		if err := os.WriteFile(filepath.Join(issuegraphDir, name), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	report, err := ScanForArtifacts(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.SQLiteArtifacts) != 4 {
		t.Errorf("expected 4 SQLite artifacts, got %d", len(report.SQLiteArtifacts))
	}

	// issuegraph.db should NOT be safe to delete
	for _, f := range report.SQLiteArtifacts {
		if filepath.Base(f.Path) == "issuegraph.db" && f.SafeDelete {
			t.Error("issuegraph.db should NOT be safe to delete")
		}
	}

	// WAL/SHM should be safe
	for _, f := range report.SQLiteArtifacts {
		name := filepath.Base(f.Path)
		if (name == "issuegraph.db-shm" || name == "issuegraph.db-wal") && !f.SafeDelete {
			t.Errorf("%s should be safe to delete", name)
		}
	}

	// Backup should be safe
	for _, f := range report.SQLiteArtifacts {
		name := filepath.Base(f.Path)
		if name == "issuegraph.backup-20260204.db" && !f.SafeDelete {
			t.Error("backup should be safe to delete")
		}
	}
}

func TestScanForArtifacts_CruftIssueGraphDir(t *testing.T) {
	// Create a structure like: issuegraph/polecats/testpolecat/.issuegraph/ with extra files
	dir := t.TempDir()
	polecatsDir := filepath.Join(dir, "polecats", "testpolecat")
	issuegraphDir := filepath.Join(polecatsDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Add redirect file (expected)
	if err := os.WriteFile(filepath.Join(issuegraphDir, "redirect"), []byte("../../mayor/rig/.issuegraph"), 0644); err != nil {
		t.Fatal(err)
	}

	// Add cruft files
	if err := os.WriteFile(filepath.Join(issuegraphDir, "issuegraph.db"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, "issues.jsonl"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := ScanForArtifacts(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.CruftIssueGraphDirs) != 1 {
		t.Errorf("expected 1 cruft issuegraph dir, got %d", len(report.CruftIssueGraphDirs))
	}

	if len(report.CruftIssueGraphDirs) > 0 && !report.CruftIssueGraphDirs[0].SafeDelete {
		t.Error("cruft issuegraph dir with redirect should be safe to delete")
	}
}

func TestScanForArtifacts_CruftIssueGraphDirNoRedirect(t *testing.T) {
	// Cruft dir without redirect should be detected AND safe to delete.
	// The location being redirect-expected is sufficient — stale cruft files
	// are what prevent the redirect from being created in the first place.
	dir := t.TempDir()
	polecatsDir := filepath.Join(dir, "polecats", "testpolecat")
	issuegraphDir := filepath.Join(polecatsDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// No redirect file, just extra files
	if err := os.WriteFile(filepath.Join(issuegraphDir, "issuegraph.db"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := ScanForArtifacts(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Should be detected as cruft (it's in a polecat location) and safe to delete
	if len(report.CruftIssueGraphDirs) != 1 {
		t.Errorf("expected 1 cruft issuegraph dir, got %d", len(report.CruftIssueGraphDirs))
	}
	if len(report.CruftIssueGraphDirs) > 0 && !report.CruftIssueGraphDirs[0].SafeDelete {
		t.Error("cruft issuegraph dir in redirect-expected location should be safe to delete")
	}
}

func TestScanForArtifacts_CrewDir(t *testing.T) {
	dir := t.TempDir()
	crewDir := filepath.Join(dir, "crew", "testcrew")
	issuegraphDir := filepath.Join(crewDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Add redirect and cruft
	if err := os.WriteFile(filepath.Join(issuegraphDir, "redirect"), []byte("../../mayor/rig/.issuegraph"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, "extra.txt"), []byte("cruft"), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := ScanForArtifacts(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.CruftIssueGraphDirs) != 1 {
		t.Errorf("expected 1 cruft issuegraph dir for crew, got %d", len(report.CruftIssueGraphDirs))
	}
}

func TestScanForArtifacts_RedirectValidation(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create redirect pointing to nonexistent target
	if err := os.WriteFile(filepath.Join(issuegraphDir, "redirect"), []byte("/nonexistent/path"), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := ScanForArtifacts(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.RedirectIssues) != 1 {
		t.Errorf("expected 1 redirect issue, got %d", len(report.RedirectIssues))
	}
}

func TestScanForArtifacts_EmptyRedirect(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create empty redirect
	if err := os.WriteFile(filepath.Join(issuegraphDir, "redirect"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := ScanForArtifacts(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.RedirectIssues) != 1 {
		t.Errorf("expected 1 redirect issue (empty), got %d", len(report.RedirectIssues))
	}
}

func TestScanForArtifacts_ValidRedirect(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	targetDir := filepath.Join(dir, "target-issuegraph")

	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	// A valid redirect target must have a metadata.json or database
	// (borg0ai/dx-issuegraph#4692 guard).
	if err := os.WriteFile(filepath.Join(targetDir, "metadata.json"), []byte(`{"database":"issuegraph.db"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create redirect pointing to valid target
	if err := os.WriteFile(filepath.Join(issuegraphDir, "redirect"), []byte(targetDir), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := ScanForArtifacts(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.RedirectIssues) != 0 {
		t.Errorf("expected 0 redirect issues for valid target, got %d", len(report.RedirectIssues))
	}
}

// TestScanForArtifacts_RedirectTargetHasNoDatabase is a regression test for
// borg0ai/dx-issuegraph#4692: a redirect target directory that exists but has no
// metadata.json and no recognizable database is flagged as an actionable
// warning (not SafeDelete cruft), matching FollowRedirect's own target-
// validity guard (internal/issuegraph).
func TestScanForArtifacts_RedirectTargetHasNoDatabase(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	targetDir := filepath.Join(dir, "target-issuegraph")

	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	// targetDir exists but is empty: no metadata.json, no database.

	if err := os.WriteFile(filepath.Join(issuegraphDir, "redirect"), []byte(targetDir), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := ScanForArtifacts(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.RedirectIssues) != 1 {
		t.Fatalf("expected 1 redirect issue (target has no database), got %d", len(report.RedirectIssues))
	}
	if report.RedirectIssues[0].SafeDelete {
		t.Errorf("expected SafeDelete=false: an invalid-target redirect is an actionable warning, not cruft")
	}
}

func TestIsRedirectExpectedDir(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"polecat worktree", "/foo/polecats/obsidian/.issuegraph", true},
		{"crew workspace", "/foo/crew/mel/.issuegraph", true},
		{"refinery rig", "/foo/refinery/rig/.issuegraph", true},
		{"issuegraph-worktrees", "/foo/.git/issuegraph-worktrees/abc/.issuegraph", true},
		{"regular issuegraph dir", "/foo/.issuegraph", false},
		{"nested project", "/foo/bar/.issuegraph", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRedirectExpectedDir(tt.path)
			if got != tt.expected {
				t.Errorf("isRedirectExpectedDir(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestScanForArtifacts_SkipsGitkeep(t *testing.T) {
	dir := t.TempDir()
	polecatsDir := filepath.Join(dir, "polecats", "test")
	issuegraphDir := filepath.Join(polecatsDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// redirect + .gitkeep only = no cruft
	if err := os.WriteFile(filepath.Join(issuegraphDir, "redirect"), []byte("../../mayor/rig/.issuegraph"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, ".gitkeep"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	report, err := ScanForArtifacts(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.CruftIssueGraphDirs) != 0 {
		t.Errorf("expected 0 cruft dirs (redirect + .gitkeep only), got %d", len(report.CruftIssueGraphDirs))
	}
}

func TestScanForArtifacts_SkipsGitInternalsButScansIssueGraphWorktrees(t *testing.T) {
	dir := t.TempDir()

	// A bogus .issuegraph under .git internals should be ignored entirely.
	ignoredIssueGraph := filepath.Join(dir, ".git", "objects", "pack", ".issuegraph")
	if err := os.MkdirAll(ignoredIssueGraph, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredIssueGraph, "issuegraph.db"), []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}

	// A worktree .issuegraph under .git/issuegraph-worktrees must still be scanned.
	worktreeIssueGraph := filepath.Join(dir, ".git", "issuegraph-worktrees", "test", ".issuegraph")
	if err := os.MkdirAll(worktreeIssueGraph, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktreeIssueGraph, "extra.txt"), []byte("cruft"), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := ScanForArtifacts(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.SQLiteArtifacts) != 0 {
		t.Fatalf("expected .git internals to be skipped, got %d sqlite findings", len(report.SQLiteArtifacts))
	}
	if len(report.CruftIssueGraphDirs) != 1 {
		t.Fatalf("expected 1 worktree cruft finding, got %d", len(report.CruftIssueGraphDirs))
	}
	if got := report.CruftIssueGraphDirs[0].Path; got != worktreeIssueGraph {
		t.Fatalf("cruft finding path = %q, want %q", got, worktreeIssueGraph)
	}
}

// TestScanForArtifacts_NonEmptyInteractionsJSONL — JSONL artifact scanning removed (bd-9ni.2)
func TestScanForArtifacts_NonEmptyInteractionsJSONL(t *testing.T) {
	t.Skip("JSONL artifact scanning removed as part of JSONL removal (bd-9ni.2)")
}

func TestCheckClassicArtifacts_WithArtifacts(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create dolt/ directory so isDoltNative returns true
	if err := os.MkdirAll(filepath.Join(issuegraphDir, "dolt"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a SQLite artifact
	if err := os.WriteFile(filepath.Join(issuegraphDir, "issuegraph.db-wal"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	check := CheckClassicArtifacts(dir)
	if check.Status != StatusWarning {
		t.Errorf("expected StatusWarning, got %s: %s", check.Status, check.Message)
	}
	if check.Name != "Classic Artifacts" {
		t.Errorf("expected name 'Classic Artifacts', got %q", check.Name)
	}
}
