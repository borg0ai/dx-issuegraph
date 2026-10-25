package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

// TestResolveIssueGraphDirForDBPath_EmbeddedNonDotIssueGraph guards GH#4574: an embedded
// store stores its data under <issuegraphDir>/embeddeddolt/, but Config.DatabasePath
// returns <issuegraphDir>/dolt. When the issuegraph dir is not literally named ".issuegraph"
// the early-return in resolveIssueGraphDirForDBPath does not fire, so resolution
// must still map <issuegraphDir>/embeddeddolt back to <issuegraphDir> via the parent
// relationship — otherwise commands silently fall back to an ancestor ~/.issuegraph
// and report zero issues.
func TestResolveIssueGraphDirForDBPath_EmbeddedNonDotIssueGraph(t *testing.T) {
	root := t.TempDir()
	// leaf is intentionally NOT ".issuegraph" (mirrors an embedding tool that keeps
	// its store at e.g. ~/.local/state/<app>/issuegraph).
	issuegraphDir := filepath.Join(root, "issuegraph")
	dbPath := filepath.Join(issuegraphDir, "embeddeddolt")
	if err := os.MkdirAll(dbPath, 0o755); err != nil {
		t.Fatalf("mkdir embeddeddolt dir: %v", err)
	}

	cfg := &configfile.Config{
		Database:     "dolt",
		Backend:      configfile.BackendDolt,
		DoltMode:     "embedded",
		DoltDatabase: "mystore",
	}
	if err := cfg.Save(issuegraphDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	if got := resolveIssueGraphDirForDBPath(dbPath); !utils.PathsEqual(got, issuegraphDir) {
		t.Fatalf("resolveIssueGraphDirForDBPath(%q) = %q, want %q", dbPath, got, issuegraphDir)
	}
}
