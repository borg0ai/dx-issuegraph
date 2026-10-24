package fix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStaleClosedIssues_NoDatabase(t *testing.T) {
	// Create temp directory with .issuegraph but no database
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph dir: %v", err)
	}

	// Should succeed without database
	err := StaleClosedIssues(tmpDir)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStaleClosedIssues_NoIssueGraphDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Should fail without .issuegraph directory
	err := StaleClosedIssues(tmpDir)
	if err == nil {
		t.Error("expected error for missing .issuegraph directory")
	}
}
