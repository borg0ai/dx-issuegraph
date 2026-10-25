package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph"
)

// TestExampleCompiles ensures the example code compiles and basic API works
func TestExampleCompiles(t *testing.T) {
	// Create temporary database for testing
	tmpDir, err := os.MkdirTemp("", "issuegraph-example-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o750); err != nil {
		t.Fatalf("Failed to create .issuegraph dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(`{"backend":"dolt"}`), 0o600); err != nil {
		t.Fatalf("Failed to create metadata.json: %v", err)
	}

	// Open storage
	ctx := context.Background()
	store, err := issuegraph.OpenBestAvailable(ctx, issuegraphDir)
	if err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer store.Close()

	// A fresh workspace needs an issue prefix before issues can be created
	// (issuegraph init normally seeds this).
	if err := store.SetConfig(ctx, "issue_prefix", "ex"); err != nil {
		t.Fatalf("Failed to seed issue prefix: %v", err)
	}

	// Create an issue (from example code)
	newIssue := &issuegraph.Issue{
		ID:          "",
		Title:       "Test library-created issue",
		Description: "This verifies the library example works",
		Status:      issuegraph.StatusOpen,
		Priority:    2,
		IssueType:   issuegraph.TypeTask,
	}

	if err := store.CreateIssue(ctx, newIssue, "test"); err != nil {
		t.Fatalf("Failed to create issue: %v", err)
	}

	if newIssue.ID == "" {
		t.Error("Issue ID should be auto-generated")
	}

	// Get ready work (from example code)
	ready, err := store.GetReadyWork(ctx, issuegraph.WorkFilter{
		Status: issuegraph.StatusOpen,
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("Failed to get ready work: %v", err)
	}

	if len(ready) == 0 {
		t.Error("Expected at least one ready issue")
	}

	// Add label (from example code)
	if err := store.AddLabel(ctx, newIssue.ID, "test-label", "test"); err != nil {
		t.Fatalf("Failed to add label: %v", err)
	}

	labels, err := store.GetLabels(ctx, newIssue.ID)
	if err != nil {
		t.Fatalf("Failed to get labels: %v", err)
	}

	if len(labels) != 1 || labels[0] != "test-label" {
		t.Errorf("Expected label 'test-label', got %v", labels)
	}

	// Update status (from example code)
	updates := map[string]interface{}{
		"status": issuegraph.StatusInProgress,
	}
	if err := store.UpdateIssue(ctx, newIssue.ID, updates, "test"); err != nil {
		t.Fatalf("Failed to update issue: %v", err)
	}

	// Get statistics (from example code)
	stats, err := store.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}

	if stats.TotalIssues == 0 {
		t.Error("Expected at least one total issue")
	}

	// Close issue (from example code)
	if err := store.CloseIssue(ctx, newIssue.ID, "Test complete", "test", ""); err != nil {
		t.Fatalf("Failed to close issue: %v", err)
	}

	// Verify closed
	closed, err := store.GetIssue(ctx, newIssue.ID)
	if err != nil {
		t.Fatalf("Failed to get issue: %v", err)
	}

	if closed.Status != issuegraph.StatusClosed {
		t.Errorf("Expected status closed, got %v", closed.Status)
	}

	t.Log("✅ All example operations work correctly")
}

// TestDependencyConstants ensures all constants are accessible
func TestDependencyConstants(t *testing.T) {
	// Test that all constants from the example are accessible
	_ = issuegraph.StatusOpen
	_ = issuegraph.StatusInProgress
	_ = issuegraph.StatusClosed
	_ = issuegraph.StatusBlocked

	_ = issuegraph.TypeBug
	_ = issuegraph.TypeFeature
	_ = issuegraph.TypeTask
	_ = issuegraph.TypeEpic
	_ = issuegraph.TypeChore

	_ = issuegraph.DepBlocks
	_ = issuegraph.DepRelated
	_ = issuegraph.DepParentChild
	_ = issuegraph.DepDiscoveredFrom
}

// TestFindDatabasePath tests database discovery
func TestFindDatabasePath(t *testing.T) {
	// Create temp directory with .issuegraph
	tmpDir, err := os.MkdirTemp("", "issuegraph-finddb-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o755); err != nil {
		t.Fatalf("Failed to create .issuegraph dir: %v", err)
	}

	dbPath := filepath.Join(issuegraphDir, "test.db")
	f, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Failed to create db file: %v", err)
	}
	f.Close()

	// Change to temp directory
	t.Chdir(tmpDir)

	// Test FindDatabasePath
	found := issuegraph.FindDatabasePath()
	if found == "" {
		t.Error("Expected to find database, got empty string")
	}

	t.Logf("Found database: %s", found)
}
