//go:build !windows

package config

import (
	"fmt"
	"io/fs"
	"os"
)

const (
	// IssueGraphDirPerm is the permission mode for .issuegraph/ directories (owner-only).
	IssueGraphDirPerm fs.FileMode = 0700
	// IssueGraphFilePerm is the permission mode for state files inside .issuegraph/ (owner-only).
	IssueGraphFilePerm fs.FileMode = 0600
)

// EnsureIssueGraphDir creates the .issuegraph directory with secure permissions.
func EnsureIssueGraphDir(path string) error {
	return os.MkdirAll(path, IssueGraphDirPerm)
}

// CheckIssueGraphDirPermissions warns to stderr if the .issuegraph directory has
// group or world-accessible permissions. The check is non-fatal.
func CheckIssueGraphDirPermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return // directory doesn't exist yet
	}
	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: %s has permissions %04o (recommended: 0700). Run: chmod 700 %s\n", path, perm, path)
	}
}

// FixIssueGraphDirPermissions sets the .issuegraph directory to IssueGraphDirPerm when it
// has group or world-accessible bits. Returns true if permissions changed.
func FixIssueGraphDirPermissions(path string) (bool, error) {
	return fixIssueGraphDirPermissions(path, openIssueGraphDirHandle)
}

type issuegraphDirHandle interface {
	Stat() (os.FileInfo, error)
	Chmod(os.FileMode) error
	Close() error
}

func fixIssueGraphDirPermissions(path string, openDir func(string) (issuegraphDirHandle, error)) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // directory doesn't exist yet
		}
		return false, fmt.Errorf("failed to inspect %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("refusing to chmod %s: path is a symbolic link", path)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("refusing to chmod %s: path is not a directory", path)
	}
	perm := info.Mode().Perm()
	if perm&0077 == 0 {
		return false, nil // no group or world-accessible bits
	}

	dir, err := openDir(path)
	if err != nil {
		return false, fmt.Errorf("failed to open %s securely: %w", path, err)
	}
	defer func() { _ = dir.Close() }()

	openedInfo, err := dir.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to inspect opened directory %s: %w", path, err)
	}
	if !openedInfo.IsDir() || !os.SameFile(info, openedInfo) {
		return false, fmt.Errorf("refusing to chmod %s: path changed during permission repair", path)
	}
	if err := dir.Chmod(IssueGraphDirPerm); err != nil {
		return false, fmt.Errorf("failed to chmod %s to %04o: %w", path, IssueGraphDirPerm, err)
	}
	return true, nil
}
