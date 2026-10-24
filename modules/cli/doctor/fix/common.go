package fix

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
)

// ErrTestBinary is returned when getBdBinary detects it's running as a test binary.
// This prevents fork bombs when tests call functions that execute issuegraph subcommands.
var ErrTestBinary = fmt.Errorf("running as test binary - cannot execute issuegraph subcommands")

type workspaceIssueGraphDirs struct {
	local    string
	resolved string
}

func newBdCmd(issuegraphBinary string, args ...string) *exec.Cmd {
	cmd := exec.Command(issuegraphBinary, args...) // #nosec G204 -- issuegraphBinary from validated executable path
	return cmd
}

// getBdBinary returns the path to the issuegraph binary to use for fix operations.
// It prefers the current executable to avoid command injection attacks.
// Returns ErrTestBinary if running as a test binary to prevent fork bombs.
func getBdBinary() (string, error) {
	// Prefer current executable for security
	exe, err := os.Executable()
	if err == nil {
		// Resolve symlinks to get the real binary path
		realPath, err := filepath.EvalSymlinks(exe)
		if err == nil {
			exe = realPath
		}

		// Check if we're running as a test binary - this prevents fork bombs
		// when tests call functions that execute issuegraph subcommands
		baseName := filepath.Base(exe)
		if strings.HasSuffix(baseName, ".test") || strings.Contains(baseName, ".test.") {
			return "", ErrTestBinary
		}

		return exe, nil
	}

	// Fallback to PATH lookup with validation
	issuegraphPath, err := exec.LookPath("issuegraph")
	if err != nil {
		return "", fmt.Errorf("issuegraph binary not found in PATH: %w", err)
	}

	return issuegraphPath, nil
}

// validateIssueGraphWorkspace ensures the path is a valid issuegraph workspace before
// attempting any fix operations. This prevents path traversal attacks.
func validateIssueGraphWorkspace(path string) error {
	_, err := resolveWorkspaceIssueGraphDirs(path)
	return err
}

func resolveWorkspaceIssueGraphDirs(path string) (workspaceIssueGraphDirs, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return workspaceIssueGraphDirs{}, fmt.Errorf("invalid path: %w", err)
	}

	dirs := workspaceIssueGraphDirs{
		local:    filepath.Join(absPath, ".issuegraph"),
		resolved: issuegraph.ResolveIssueGraphDirForRepo(absPath),
	}

	if info, err := os.Stat(dirs.resolved); err != nil || !info.IsDir() {
		return workspaceIssueGraphDirs{}, fmt.Errorf("not a issuegraph workspace: .issuegraph directory not found for %s", absPath)
	}

	return dirs, nil
}

func resolvedWorkspaceIssueGraphDir(path string) (string, error) {
	dirs, err := resolveWorkspaceIssueGraphDirs(path)
	if err != nil {
		return "", err
	}

	return dirs.resolved, nil
}

func localWorkspaceIssueGraphDir(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	return filepath.Join(absPath, ".issuegraph"), nil
}

// safeWorkspacePath resolves relPath within the workspace root and ensures it
// cannot escape the workspace via path traversal.
func safeWorkspacePath(root, relPath string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("invalid workspace path: %w", err)
	}

	cleanRel := filepath.Clean(relPath)
	if filepath.IsAbs(cleanRel) {
		return "", fmt.Errorf("expected relative path, got absolute: %s", relPath)
	}

	joined := filepath.Join(absRoot, cleanRel)
	rel, err := filepath.Rel(absRoot, joined)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes workspace: %s", relPath)
	}

	return joined, nil
}

// isWithinWorkspace reports whether candidate resides within the workspace root.
func isWithinWorkspace(root, candidate string) bool {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	cleanCandidate := filepath.Clean(candidate)
	rel, err := filepath.Rel(cleanRoot, cleanCandidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

// resolveIssueGraphDir follows .issuegraph/redirect files to find the actual issuegraph directory.
// If no redirect exists, returns the original path unchanged.
// This is a wrapper around issuegraph.FollowRedirect for use within the fix package.
func resolveIssueGraphDir(issuegraphDir string) string {
	return issuegraph.FollowRedirect(issuegraphDir)
}
