package fix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassicArtifacts_NoArtifacts(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	err := ClassicArtifacts(dir)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestClassicArtifacts_RemovesSQLiteWAL(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create WAL/SHM files
	for _, name := range []string{"issuegraph.db-shm", "issuegraph.db-wal"} {
		if err := os.WriteFile(filepath.Join(issuegraphDir, name), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	err := ClassicArtifacts(dir)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Verify WAL/SHM files were removed
	for _, name := range []string{"issuegraph.db-shm", "issuegraph.db-wal"} {
		if _, err := os.Stat(filepath.Join(issuegraphDir, name)); !os.IsNotExist(err) {
			t.Errorf("%s should have been removed", name)
		}
	}
}

func TestClassicArtifacts_SkipsIssueGraphDB(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create issuegraph.db (should be skipped)
	if err := os.WriteFile(filepath.Join(issuegraphDir, "issuegraph.db"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	err := ClassicArtifacts(dir)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// issuegraph.db should still exist (not safe to delete automatically)
	if _, err := os.Stat(filepath.Join(issuegraphDir, "issuegraph.db")); os.IsNotExist(err) {
		t.Error("issuegraph.db should NOT have been removed")
	}
}

func TestClassicArtifacts_RemovesBackupDBs(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	backupName := "issuegraph.backup-20260204.db"
	if err := os.WriteFile(filepath.Join(issuegraphDir, backupName), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	err := ClassicArtifacts(dir)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(issuegraphDir, backupName)); !os.IsNotExist(err) {
		t.Error("backup db should have been removed")
	}
}

func TestClassicArtifacts_CleansJSONLInDoltDir(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	doltDir := filepath.Join(issuegraphDir, "dolt")
	if err := os.MkdirAll(doltDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create safe-to-delete JSONL artifacts
	for _, name := range []string{"issues.jsonl.new"} {
		if err := os.WriteFile(filepath.Join(issuegraphDir, name), []byte(`{"id":"test"}`), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create empty interactions.jsonl
	if err := os.WriteFile(filepath.Join(issuegraphDir, "interactions.jsonl"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Create issues.jsonl (should be skipped)
	if err := os.WriteFile(filepath.Join(issuegraphDir, "issues.jsonl"), []byte(`{"id":"real"}`), 0644); err != nil {
		t.Fatal(err)
	}

	err := ClassicArtifacts(dir)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Safe files should be removed
	for _, name := range []string{"issues.jsonl.new", "interactions.jsonl"} {
		if _, err := os.Stat(filepath.Join(issuegraphDir, name)); !os.IsNotExist(err) {
			t.Errorf("%s should have been removed", name)
		}
	}

	// issues.jsonl should be kept
	if _, err := os.Stat(filepath.Join(issuegraphDir, "issues.jsonl")); os.IsNotExist(err) {
		t.Error("issues.jsonl should NOT have been removed")
	}
}

func TestClassicArtifacts_CleansCruftIssueGraphDir(t *testing.T) {
	dir := t.TempDir()
	polecatsDir := filepath.Join(dir, "polecats", "test")
	issuegraphDir := filepath.Join(polecatsDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Add redirect (expected)
	if err := os.WriteFile(filepath.Join(issuegraphDir, "redirect"), []byte("../../mayor/rig/.issuegraph"), 0644); err != nil {
		t.Fatal(err)
	}

	// Add .gitkeep (should be preserved)
	if err := os.WriteFile(filepath.Join(issuegraphDir, ".gitkeep"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Add cruft
	if err := os.WriteFile(filepath.Join(issuegraphDir, "extra.txt"), []byte("cruft"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(issuegraphDir, "cruft-subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	err := ClassicArtifacts(dir)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// redirect should still exist
	if _, err := os.Stat(filepath.Join(issuegraphDir, "redirect")); os.IsNotExist(err) {
		t.Error("redirect should NOT have been removed")
	}

	// .gitkeep should still exist
	if _, err := os.Stat(filepath.Join(issuegraphDir, ".gitkeep")); os.IsNotExist(err) {
		t.Error(".gitkeep should NOT have been removed")
	}

	// cruft should be removed
	if _, err := os.Stat(filepath.Join(issuegraphDir, "extra.txt")); !os.IsNotExist(err) {
		t.Error("extra.txt should have been removed")
	}
	if _, err := os.Stat(filepath.Join(issuegraphDir, "cruft-subdir")); !os.IsNotExist(err) {
		t.Error("cruft-subdir should have been removed")
	}
}

// TestClassicArtifacts_PreservesMetadataJSONAlongsideRedirect is a
// regression test for borg0ai/dx-issuegraph#4692: a metadata.json co-located
// with a redirect file (the fb51196f7 / docs "Database Redirects" topology,
// where a server-mode source rig's own metadata.json supplies its
// dolt_database while a redirect points at the shared Gas Town root) must
// never be deleted as cruft while the redirect is preserved.
func TestClassicArtifacts_PreservesMetadataJSONAlongsideRedirect(t *testing.T) {
	dir := t.TempDir()
	polecatsDir := filepath.Join(dir, "polecats", "test")
	issuegraphDir := filepath.Join(polecatsDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(issuegraphDir, "redirect"), []byte("../../mayor/rig/.issuegraph"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(`{"dolt_mode":"server","dolt_database":"lola"}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Still remove unrelated cruft in the same pass.
	if err := os.WriteFile(filepath.Join(issuegraphDir, "extra.txt"), []byte("cruft"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ClassicArtifacts(dir); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(issuegraphDir, "redirect")); os.IsNotExist(err) {
		t.Error("redirect should NOT have been removed")
	}
	if _, err := os.Stat(filepath.Join(issuegraphDir, "metadata.json")); os.IsNotExist(err) {
		t.Error("metadata.json should NOT have been removed while a redirect is present")
	}
	if _, err := os.Stat(filepath.Join(issuegraphDir, "extra.txt")); !os.IsNotExist(err) {
		t.Error("extra.txt should have been removed")
	}
}

func TestClassicArtifacts_CleansCruftWithoutRedirectFile(t *testing.T) {
	// Regression test: when a redirect-expected location has cruft files but
	// NO redirect file, the fix should still clean up the cruft.
	// Previously, the fix required both isRedirectExpected AND hasRedirect,
	// leaving stale files (config.yaml, metadata.json, README.md, issues.jsonl)
	// in .git/issuegraph-worktrees/*/.issuegraph/ directories.
	dir := t.TempDir()

	// Simulate .git/issuegraph-worktrees/master/.issuegraph/ (redirect-expected location)
	worktreeIssueGraph := filepath.Join(dir, ".git", "issuegraph-worktrees", "master", ".issuegraph")
	if err := os.MkdirAll(worktreeIssueGraph, 0755); err != nil {
		t.Fatal(err)
	}

	// Add typical stale files (NO redirect file present)
	cruftFiles := []string{"config.yaml", "metadata.json", "README.md", "issues.jsonl", ".gitignore", ".local_version"}
	for _, name := range cruftFiles {
		if err := os.WriteFile(filepath.Join(worktreeIssueGraph, name), []byte("stale"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	err := ClassicArtifacts(dir)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// All cruft files should be removed
	for _, name := range cruftFiles {
		if _, err := os.Stat(filepath.Join(worktreeIssueGraph, name)); !os.IsNotExist(err) {
			t.Errorf("%s should have been removed (cruft in redirect-expected dir without redirect file)", name)
		}
	}
}

func TestIsRedirectExpectedLocation(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRedirectExpectedLocation(tt.path)
			if got != tt.expected {
				t.Errorf("isRedirectExpectedLocation(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}
