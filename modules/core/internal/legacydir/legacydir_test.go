package legacydir

import (
	"path/filepath"
	"testing"
)

func TestEnvIssueGraphDir(t *testing.T) {
	t.Setenv("ISSUEGRAPH_DIR", "")
	if got := EnvIssueGraphDir(); got != "" {
		t.Fatalf("unset: got %q", got)
	}
	t.Setenv("ISSUEGRAPH_DIR", "/new")
	if got := EnvIssueGraphDir(); got != "/new" {
		t.Fatalf("ISSUEGRAPH_DIR: got %q", got)
	}
}

func TestWorkspaceDir(t *testing.T) {
	root := t.TempDir()
	if got, want := WorkspaceDir(root), filepath.Join(root, ".issuegraph"); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
