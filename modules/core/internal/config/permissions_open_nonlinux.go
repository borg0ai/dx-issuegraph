//go:build !windows && !linux

package config

import "os"

// os.Open may follow a path swapped to a symlink on these platforms. The
// caller's descriptor identity check prevents chmod unless it still refers to
// the directory inspected with Lstat.
func openIssueGraphDirHandle(path string) (issuegraphDirHandle, error) {
	return os.Open(path)
}
