package legacydir

import (
	"os"
	"path/filepath"
)

// EnvIssueGraphDir returns ISSUEGRAPH_DIR, falling back to the legacy
// BEADS_DIR (RFC 0002 migration period).
func EnvIssueGraphDir() string {
	if v := os.Getenv("ISSUEGRAPH_DIR"); v != "" {
		return v
	}
	return os.Getenv("BEADS_DIR")
}

// WorkspaceDir returns root/.issuegraph, or root/.beads when only the legacy
// directory exists. Existing workspaces are read and written in place.
func WorkspaceDir(root string) string {
	dir := filepath.Join(root, ".issuegraph")
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	legacy := filepath.Join(root, ".beads")
	if info, err := os.Stat(legacy); err == nil && info.IsDir() {
		return legacy
	}
	return dir
}
