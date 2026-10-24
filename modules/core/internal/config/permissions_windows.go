//go:build windows

package config

import (
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

// CheckIssueGraphDirPermissions is a no-op on Windows where filesystem
// permissions use ACLs rather than Unix permission bits.
func CheckIssueGraphDirPermissions(path string) {}

// FixIssueGraphDirPermissions is a no-op on Windows where filesystem
// permissions use ACLs rather than Unix permission bits.
func FixIssueGraphDirPermissions(path string) (bool, error) { return false, nil }
