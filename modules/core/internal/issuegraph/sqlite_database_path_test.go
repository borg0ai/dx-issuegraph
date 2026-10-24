package issuegraph

import (
	"os"
	"path/filepath"
	"testing"
)

// The sqlite backend is a removed-backend tombstone: discovery must return ""
// so store selection reports the metadata error, and it must never create the
// database file the removed backend would have provisioned.
func TestFindDatabasePathSQLiteMetadataOutsideCWD(t *testing.T) {
	tests := []struct {
		name         string
		metadata     string
		relativePath string
	}{
		{
			name:         "default path",
			metadata:     `{"backend":"sqlite","sqlite_path":"issuegraph.db"}`,
			relativePath: "issuegraph.db",
		},
		{
			name:         "custom relative path",
			metadata:     `{"backend":"sqlite","sqlite_path":"data/custom.db"}`,
			relativePath: filepath.Join("data", "custom.db"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			cwd := filepath.Join(root, "working-directory")
			issuegraphDir := filepath.Join(root, "external-workspace", ".issuegraph")
			if err := os.MkdirAll(cwd, 0o750); err != nil {
				t.Fatalf("create working directory: %v", err)
			}
			if err := os.MkdirAll(issuegraphDir, 0o750); err != nil {
				t.Fatalf("create issuegraph directory: %v", err)
			}
			if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(tt.metadata), 0o600); err != nil {
				t.Fatalf("write metadata.json: %v", err)
			}

			t.Chdir(cwd)
			t.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
			t.Setenv("ISSUEGRAPH_DB", "")
			t.Setenv("ISSUEGRAPH_DB", "")

			dbPath := filepath.Join(issuegraphDir, tt.relativePath)
			if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
				t.Fatalf("SQLite database unexpectedly exists before discovery: %v", err)
			}

			if got := FindDatabasePath(); got != "" {
				t.Errorf("FindDatabasePath() = %q, want \"\" for the removed sqlite backend", got)
			}
			if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
				t.Fatalf("FindDatabasePath created SQLite database path %q: %v", dbPath, err)
			}
		})
	}
}

func TestFindDatabasePathDoesNotBypassCorruptMetadata(t *testing.T) {
	root := t.TempDir()
	issuegraphDir := filepath.Join(root, "selected", ".issuegraph")
	leftoverDolt := filepath.Join(issuegraphDir, "embeddeddolt")
	if err := os.MkdirAll(leftoverDolt, 0o750); err != nil {
		t.Fatalf("create leftover Dolt directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("write corrupt metadata.json: %v", err)
	}
	ambiguousCWD := filepath.Join(root, "ambient")
	if err := os.MkdirAll(filepath.Join(ambiguousCWD, ".issuegraph", "embeddeddolt"), 0o750); err != nil {
		t.Fatalf("create ambient Dolt workspace: %v", err)
	}
	t.Chdir(ambiguousCWD)

	t.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
	t.Setenv("ISSUEGRAPH_DB", "")
	t.Setenv("ISSUEGRAPH_DB", "")

	if got := FindDatabasePath(); got != "" {
		t.Fatalf("FindDatabasePath() = %q with corrupt metadata, want empty path instead of leftover Dolt", got)
	}
}
