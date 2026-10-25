package legacydir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvIssueGraphDir(t *testing.T) {
	t.Setenv("ISSUEGRAPH_DIR", "")
	t.Setenv("BEADS_DIR", "")
	if got := EnvIssueGraphDir(); got != "" {
		t.Fatalf("both unset: got %q", got)
	}
	t.Setenv("BEADS_DIR", "/legacy")
	if got := EnvIssueGraphDir(); got != "/legacy" {
		t.Fatalf("BEADS_DIR fallback: got %q", got)
	}
	t.Setenv("ISSUEGRAPH_DIR", "/new")
	if got := EnvIssueGraphDir(); got != "/new" {
		t.Fatalf("ISSUEGRAPH_DIR must win: got %q", got)
	}
}

func TestWorkspaceDir(t *testing.T) {
	root := t.TempDir()
	if got, want := WorkspaceDir(root), filepath.Join(root, ".issuegraph"); got != want {
		t.Fatalf("empty root: got %q want %q", got, want)
	}
	if err := os.Mkdir(filepath.Join(root, ".beads"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := WorkspaceDir(root), filepath.Join(root, ".beads"); got != want {
		t.Fatalf("legacy only: got %q want %q", got, want)
	}
	if err := os.Mkdir(filepath.Join(root, ".issuegraph"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := WorkspaceDir(root), filepath.Join(root, ".issuegraph"); got != want {
		t.Fatalf("both: got %q want %q", got, want)
	}
}
