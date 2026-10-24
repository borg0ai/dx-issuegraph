package embeddeddolt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasRepository(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, issuegraphDir string)
		want  bool
	}{
		{name: "missing root", want: false},
		{
			name: "empty root",
			setup: func(t *testing.T, issuegraphDir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(issuegraphDir, "embeddeddolt"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
			want: false,
		},
		{
			name: "empty repository marker",
			setup: func(t *testing.T, issuegraphDir string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(issuegraphDir, "embeddeddolt", "issuegraph", ".dolt"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
			want: false,
		},
		{
			name: "nonempty repository marker",
			setup: func(t *testing.T, issuegraphDir string) {
				t.Helper()
				marker := filepath.Join(issuegraphDir, "embeddeddolt", "issuegraph", ".dolt")
				if err := os.MkdirAll(marker, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(marker, "opaque-entry"), []byte("not inspected"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			want: true,
		},
		{
			name: "symlink marker is refused",
			setup: func(t *testing.T, issuegraphDir string) {
				t.Helper()
				databaseDir := filepath.Join(issuegraphDir, "embeddeddolt", "issuegraph")
				if err := os.MkdirAll(databaseDir, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(issuegraphDir, filepath.Join(databaseDir, ".dolt")); err != nil {
					t.Fatal(err)
				}
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuegraphDir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, issuegraphDir)
			}
			if got := HasRepository(issuegraphDir); got != tt.want {
				t.Fatalf("HasRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}
