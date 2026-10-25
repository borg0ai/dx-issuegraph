//go:build cgo

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/types"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
	"github.com/spf13/cobra"
)

func TestWithStorage_ReopensUsingMetadata(t *testing.T) {
	ctx := context.Background()
	testDBPath := filepath.Join(t.TempDir(), "dolt")
	newTestStoreIsolatedDB(t, testDBPath, "cfg")

	var gotPrefix string
	err := withStorage(ctx, nil, testDBPath, func(s storage.DoltStorage) error {
		var err error
		gotPrefix, err = s.GetConfig(ctx, "issue_prefix")
		return err
	})
	if err != nil {
		t.Fatalf("withStorage() error = %v", err)
	}
	if gotPrefix != "cfg" {
		t.Fatalf("issue_prefix = %q, want %q", gotPrefix, "cfg")
	}
}

func TestResolveIssueGraphDirForDBPath_UsesRawIssueGraphDirForSymlinkedDBPath(t *testing.T) {
	repoDir := t.TempDir()
	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	actualDBPath := filepath.Join(repoDir, "external-dolt")
	linkDBPath := filepath.Join(issuegraphDir, "dolt")

	if err := os.MkdirAll(issuegraphDir, 0o755); err != nil {
		t.Fatalf("mkdir issuegraph dir: %v", err)
	}
	if err := os.MkdirAll(actualDBPath, 0o755); err != nil {
		t.Fatalf("mkdir external dolt dir: %v", err)
	}
	if err := os.Symlink(actualDBPath, linkDBPath); err != nil {
		t.Fatalf("symlink db path: %v", err)
	}

	cfg := &configfile.Config{
		Database: "dolt",
		Backend:  configfile.BackendDolt,
	}
	if err := cfg.Save(issuegraphDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	if got := resolveIssueGraphDirForDBPath(linkDBPath); !utils.PathsEqual(got, issuegraphDir) {
		t.Fatalf("resolveIssueGraphDirForDBPath(%q) = %q, want %q", linkDBPath, got, issuegraphDir)
	}
}

func TestIssueIDCompletion_UsesMetadataWhenStoreNil(t *testing.T) {
	originalStore := store
	originalDBPath := dbPath
	originalRootCtx := rootCtx
	defer func() {
		store = originalStore
		dbPath = originalDBPath
		rootCtx = originalRootCtx
	}()

	ctx := context.Background()
	rootCtx = ctx

	testDBPath := filepath.Join(t.TempDir(), "dolt")
	testStore := newTestStoreIsolatedDB(t, testDBPath, "cfg")
	if err := testStore.CreateIssue(ctx, &types.Issue{
		ID:        "cfg-abc1",
		Title:     "Completion target",
		Status:    types.StatusOpen,
		Priority:  1,
		IssueType: types.TypeTask,
	}, "test"); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}

	store = nil
	dbPath = testDBPath

	completions, directive := issueIDCompletion(&cobra.Command{}, nil, "cfg-a")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %d, want %d", directive, cobra.ShellCompDirectiveNoFileComp)
	}
	if len(completions) != 1 {
		t.Fatalf("len(completions) = %d, want 1 (%v)", len(completions), completions)
	}
	if len(completions[0]) < len("cfg-abc1") || completions[0][:len("cfg-abc1")] != "cfg-abc1" {
		t.Fatalf("completion = %q, want prefix %q", completions[0], "cfg-abc1")
	}
}

func TestResolveCommandIssueGraphDir_NoCWDFallbackForExplicitPath(t *testing.T) {
	// Set up project A with metadata so FindIssueGraphDir() discovers it from CWD.
	projectA := t.TempDir()
	issuegraphDirA := filepath.Join(projectA, ".issuegraph")
	if err := os.MkdirAll(filepath.Join(issuegraphDirA, "dolt"), 0o755); err != nil {
		t.Fatalf("mkdir issuegraph dir A: %v", err)
	}
	cfgA := &configfile.Config{
		Database:     "dolt",
		Backend:      configfile.BackendDolt,
		DoltDatabase: "project_a_db",
	}
	if err := cfgA.Save(issuegraphDirA); err != nil {
		t.Fatalf("save metadata A: %v", err)
	}

	// Project B: .issuegraph/dolt exists but metadata.json is missing.
	// This triggers the bug: filepath.Dir(dbPath) gives the correct
	// .issuegraph dir but configfile.Load returns nil, so the old code falls
	// through to FindIssueGraphDir() which discovers project A instead.
	projectB := t.TempDir()
	issuegraphDirB := filepath.Join(projectB, ".issuegraph")
	if err := os.MkdirAll(filepath.Join(issuegraphDirB, "dolt"), 0o755); err != nil {
		t.Fatalf("mkdir issuegraph dir B: %v", err)
	}

	// CWD is inside project A so FindIssueGraphDir() discovers A
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectA); err != nil {
		t.Fatalf("chdir to project A: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Simulate --db pointing to project B's database path
	dbPathB := filepath.Join(issuegraphDirB, "dolt")
	got := resolveCommandIssueGraphDir(dbPathB)

	// Must resolve to project B's .issuegraph, NOT project A's.
	// The old code falls back to FindIssueGraphDir() and returns issuegraphDirA.
	if !utils.PathsEqual(got, issuegraphDirB) {
		t.Fatalf("resolveCommandIssueGraphDir(%q) = %q, want %q", dbPathB, got, issuegraphDirB)
	}
}

func TestGetGitHubConfigValue_UsesMetadataWhenStoreNil(t *testing.T) {
	// github.token is now a YAML-only key (not stored in Dolt DB) to avoid
	// leaking secrets when pushing to remotes. Test that the env-var fallback
	// still works when the store is nil.
	originalStore := store
	originalDBPath := dbPath
	defer func() {
		store = originalStore
		dbPath = originalDBPath
	}()

	ctx := context.Background()
	store = nil
	dbPath = ""

	t.Setenv("GITHUB_TOKEN", "ghp_test_token")

	if got := getGitHubConfigValue(ctx, "github.token"); got != "ghp_test_token" {
		t.Fatalf("getGitHubConfigValue() = %q, want %q", got, "ghp_test_token")
	}
}
