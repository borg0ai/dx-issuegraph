package legacydir

import (
	"os"
	"path/filepath"
)

// EnvIssueGraphDir returns ISSUEGRAPH_DIR.
func EnvIssueGraphDir() string {
	return os.Getenv("ISSUEGRAPH_DIR")
}

// WorkspaceDir returns root/.issuegraph.
func WorkspaceDir(root string) string {
	return filepath.Join(root, ".issuegraph")
}
