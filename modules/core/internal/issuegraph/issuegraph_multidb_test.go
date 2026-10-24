//go:build integration
// +build integration

package issuegraph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindAllDatabases(t *testing.T) {
	// Create a temporary directory structure with multiple .issuegraph databases
	tmpDir, err := os.MkdirTemp("", "issuegraph-multidb-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Resolve symlinks (macOS /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create nested directory structure:
	// tmpDir/
	//   .issuegraph/test.db
	//   project1/
	//     .issuegraph/project1.db
	//     subdir/
	//       (working directory here)

	// Root .issuegraph
	rootIssueGraph := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(rootIssueGraph, 0750); err != nil {
		t.Fatal(err)
	}
	rootDB := filepath.Join(rootIssueGraph, "test.db")
	if err := os.WriteFile(rootDB, []byte("fake db"), 0600); err != nil {
		t.Fatal(err)
	}

	// Project1 .issuegraph
	project1Dir := filepath.Join(tmpDir, "project1")
	project1IssueGraph := filepath.Join(project1Dir, ".issuegraph")
	if err := os.MkdirAll(project1IssueGraph, 0750); err != nil {
		t.Fatal(err)
	}
	project1DB := filepath.Join(project1IssueGraph, "project1.db")
	if err := os.WriteFile(project1DB, []byte("fake db"), 0600); err != nil {
		t.Fatal(err)
	}

	// Subdir for working directory
	subdir := filepath.Join(project1Dir, "subdir")
	if err := os.MkdirAll(subdir, 0750); err != nil {
		t.Fatal(err)
	}

	// Change to subdir and test FindAllDatabases
	t.Chdir(subdir)

	databases := FindAllDatabases()

	// Should find only the closest database (gt-bzd: stop searching when .issuegraph found)
	// Parent .issuegraph directories are out of scope in multi-workspace setups
	if len(databases) != 1 {
		t.Fatalf("expected 1 database (closest only), got %d", len(databases))
	}

	// Database should be project1 (closest to CWD)
	if databases[0].Path != project1DB {
		t.Errorf("expected database to be %s, got %s", project1DB, databases[0].Path)
	}
	if databases[0].IssueGraphDir != project1IssueGraph {
		t.Errorf("expected issuegraph dir to be %s, got %s", project1IssueGraph, databases[0].IssueGraphDir)
	}

	// Root database should NOT be found - it's out of scope (parent of closest .issuegraph)
	_ = rootDB         // referenced but not expected in results
	_ = rootIssueGraph // referenced but not expected in results
}

func TestFindAllDatabases_Single(t *testing.T) {
	// Create a temporary directory with only one database
	tmpDir, err := os.MkdirTemp("", "issuegraph-single-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Resolve symlinks (macOS /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create .issuegraph directory with database
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0750); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(issuegraphDir, "test.db")
	if err := os.WriteFile(dbPath, []byte("fake db"), 0600); err != nil {
		t.Fatal(err)
	}

	// Change to tmpDir and test
	t.Chdir(tmpDir)

	databases := FindAllDatabases()

	// Should find exactly one database
	if len(databases) != 1 {
		t.Fatalf("expected 1 database, got %d", len(databases))
	}

	if databases[0].Path != dbPath {
		t.Errorf("expected database path %s, got %s", dbPath, databases[0].Path)
	}
}

func TestFindAllDatabases_None(t *testing.T) {
	// Create a temporary directory with no databases
	tmpDir, err := os.MkdirTemp("", "issuegraph-none-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to tmpDir and test
	t.Chdir(tmpDir)

	databases := FindAllDatabases()

	// Should find no databases
	if len(databases) != 0 {
		t.Fatalf("expected 0 databases, got %d", len(databases))
	}
}
