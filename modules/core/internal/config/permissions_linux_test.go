//go:build linux

package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestFixIssueGraphDirPermissions_UnreadableDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".issuegraph")
	if err := os.Mkdir(path, 0o111); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o111); err != nil {
		t.Fatal(err)
	}

	fixed, err := FixIssueGraphDirPermissions(path)
	if errors.Is(err, unix.EOPNOTSUPP) {
		t.Skip("kernel does not support fchmodat2 with AT_EMPTY_PATH")
	}
	if err != nil {
		t.Fatalf("FixIssueGraphDirPermissions() error = %v", err)
	}
	if !fixed {
		t.Fatal("expected unreadable insecure directory to be fixed")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != IssueGraphDirPerm {
		t.Fatalf("permissions after fix = %04o, want %04o", got, IssueGraphDirPerm)
	}
}

func TestLinuxPathDirHandle_ChmodUnreadableDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".issuegraph")
	if err := os.Mkdir(path, 0o111); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o111); err != nil {
		t.Fatal(err)
	}

	handle, err := openIssueGraphDirPathHandle(path)
	if err != nil {
		t.Fatalf("openIssueGraphDirPathHandle() error = %v", err)
	}
	defer handle.Close()
	if err := handle.Chmod(IssueGraphDirPerm); err != nil {
		if errors.Is(err, unix.EOPNOTSUPP) {
			t.Skip("kernel does not support fchmodat2 with AT_EMPTY_PATH")
		}
		t.Fatalf("descriptor chmod error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != IssueGraphDirPerm {
		t.Fatalf("permissions after descriptor chmod = %04o, want %04o", got, IssueGraphDirPerm)
	}
}
