//go:build cgo

package fix

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dolt"
	"github.com/borg0ai/dx-issuegraph/internal/testutil"
)

// setupDoltWorkspace creates a temp issuegraph workspace with a Dolt database.
// The database is initialized with the metadata table but no metadata values.
// Returns the workspace root path.
func setupDoltWorkspace(t *testing.T) string {
	t.Helper()
	testutil.RequireDoltBinary(t)
	// Binary present != server listening; see dep_keys_test.go. dolt.New
	// below needs the container port TestMain publishes via ISSUEGRAPH_DOLT_PORT.
	requireFixDoltContainer(t)

	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph directory: %v", err)
	}

	// Set up git repo for repo_id computation (from cached template)
	initGitTemplate()
	if gitTemplateErr != nil {
		t.Fatalf("git template init failed: %v", gitTemplateErr)
	}
	if err := copyGitDir(gitTemplateDir, dir); err != nil {
		t.Fatalf("failed to copy git template: %v", err)
	}
	cmd := exec.Command("git", "config", "remote.origin.url", "https://github.com/test/dolt-metadata-fix.git")
	cmd.Dir = dir
	_ = cmd.Run()

	// Create Dolt config
	cfg := &configfile.Config{
		Database: "dolt",
		Backend:  configfile.BackendDolt,
	}
	if err := cfg.Save(issuegraphDir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create the Dolt store via factory (which bootstraps the database)
	ctx := context.Background()
	doltPath := filepath.Join(issuegraphDir, "dolt")
	store, err := dolt.New(ctx, &dolt.Config{
		Path:            doltPath,
		Database:        "issuegraph",
		CreateIfMissing: true,
	})
	if err != nil {
		t.Fatalf("dolt.New: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("failed to close Dolt store: %v", err)
	}

	return dir
}

// TestFixMissingMetadata_DoltRepair verifies that FixMissingMetadata writes
// all 3 metadata fields to a Dolt database that has none.
// Covers FR-010, FR-011, SC-004.
func TestFixMissingMetadata_DoltRepair(t *testing.T) {
	dir := setupDoltWorkspace(t)

	// Run the fix with a known version
	err := FixMissingMetadata(dir, "0.49.6")
	if err != nil {
		t.Fatalf("FixMissingMetadata failed: %v", err)
	}

	// Verify metadata was written by opening the store
	ctx := context.Background()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	store, err := dolt.NewFromConfig(ctx, issuegraphDir)
	if err != nil {
		t.Fatalf("failed to reopen store for verification: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Check issuegraph_version
	issuegraphVersion, err := store.GetLocalMetadata(ctx, "issuegraph_version")
	if err != nil {
		t.Fatalf("GetLocalMetadata(issuegraph_version) error: %v", err)
	}
	if issuegraphVersion != "0.49.6" {
		t.Errorf("issuegraph_version = %q, want %q", issuegraphVersion, "0.49.6")
	}

	// Check repo_id (should be set since we have a git repo with remote)
	repoID, err := store.GetMetadata(ctx, "repo_id")
	if err != nil {
		t.Fatalf("GetMetadata(repo_id) error: %v", err)
	}
	if repoID == "" {
		t.Error("repo_id was not set")
	}

	// Check clone_id
	cloneID, err := store.GetMetadata(ctx, "clone_id")
	if err != nil {
		t.Fatalf("GetMetadata(clone_id) error: %v", err)
	}
	if cloneID == "" {
		t.Error("clone_id was not set")
	}
}

// TestFixMissingMetadata_DoltIdempotent verifies that running FixMissingMetadata
// on a Dolt database that already has all metadata is a no-op.
// Covers FR-012, T028.
func TestFixMissingMetadata_DoltIdempotent(t *testing.T) {
	dir := setupDoltWorkspace(t)

	// First run: set all metadata
	if err := FixMissingMetadata(dir, "0.49.6"); err != nil {
		t.Fatalf("first FixMissingMetadata failed: %v", err)
	}

	// Read the values that were set
	ctx := context.Background()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	store, err := dolt.NewFromConfig(ctx, issuegraphDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	origVersion, _ := store.GetLocalMetadata(ctx, "issuegraph_version")
	origRepoID, _ := store.GetMetadata(ctx, "repo_id")
	origCloneID, _ := store.GetMetadata(ctx, "clone_id")
	_ = store.Close()

	// Second run: should be a no-op
	if err := FixMissingMetadata(dir, "0.50.0"); err != nil {
		t.Fatalf("second FixMissingMetadata failed: %v", err)
	}

	// Verify values did not change (version should remain 0.49.6, not 0.50.0)
	store2, err := dolt.NewFromConfig(ctx, issuegraphDir)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer func() { _ = store2.Close() }()

	newVersion, _ := store2.GetLocalMetadata(ctx, "issuegraph_version")
	if newVersion != origVersion {
		t.Errorf("issuegraph_version changed from %q to %q (should be idempotent)", origVersion, newVersion)
	}

	newRepoID, _ := store2.GetMetadata(ctx, "repo_id")
	if newRepoID != origRepoID {
		t.Errorf("repo_id changed from %q to %q (should be idempotent)", origRepoID, newRepoID)
	}

	newCloneID, _ := store2.GetMetadata(ctx, "clone_id")
	if newCloneID != origCloneID {
		t.Errorf("clone_id changed from %q to %q (should be idempotent)", origCloneID, newCloneID)
	}
}

// TestFixMissingMetadata_DoltPartialRepair verifies that FixMissingMetadata
// only repairs missing fields and leaves existing ones untouched.
func TestFixMissingMetadata_DoltPartialRepair(t *testing.T) {
	dir := setupDoltWorkspace(t)

	// Pre-set only issuegraph_version
	ctx := context.Background()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	store, err := dolt.NewFromConfig(ctx, issuegraphDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	if err := store.SetLocalMetadata(ctx, "issuegraph_version", "0.48.0"); err != nil {
		t.Fatalf("failed to pre-set issuegraph_version: %v", err)
	}
	_ = store.Close()

	// Run the fix
	if err := FixMissingMetadata(dir, "0.49.6"); err != nil {
		t.Fatalf("FixMissingMetadata failed: %v", err)
	}

	// Verify: issuegraph_version should remain 0.48.0 (not overwritten)
	store2, err := dolt.NewFromConfig(ctx, issuegraphDir)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer func() { _ = store2.Close() }()

	issuegraphVersion, _ := store2.GetLocalMetadata(ctx, "issuegraph_version")
	if issuegraphVersion != "0.48.0" {
		t.Errorf("issuegraph_version = %q, want %q (should not overwrite existing)", issuegraphVersion, "0.48.0")
	}

	// repo_id and clone_id should have been set
	repoID, _ := store2.GetMetadata(ctx, "repo_id")
	if repoID == "" {
		t.Error("repo_id was not set during partial repair")
	}

	cloneID, _ := store2.GetMetadata(ctx, "clone_id")
	if cloneID == "" {
		t.Error("clone_id was not set during partial repair")
	}
}
