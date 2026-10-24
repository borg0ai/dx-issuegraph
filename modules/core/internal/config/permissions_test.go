//go:build !windows

package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestIssueGraphDirPermConstants(t *testing.T) {
	if IssueGraphDirPerm != 0700 {
		t.Errorf("IssueGraphDirPerm = %04o, want 0700", IssueGraphDirPerm)
	}
	if IssueGraphFilePerm != 0600 {
		t.Errorf("IssueGraphFilePerm = %04o, want 0600", IssueGraphFilePerm)
	}
}

func TestEnsureIssueGraphDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".issuegraph")
	if err := EnsureIssueGraphDir(dir); err != nil {
		t.Fatalf("EnsureIssueGraphDir(%q) = %v", dir, err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(%q) = %v", dir, err)
	}
	perm := info.Mode().Perm()
	if perm != IssueGraphDirPerm {
		t.Errorf("directory permissions = %04o, want %04o", perm, IssueGraphDirPerm)
	}
}

func TestEnsureIssueGraphDirNested(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".issuegraph", "dolt")
	if err := EnsureIssueGraphDir(dir); err != nil {
		t.Fatalf("EnsureIssueGraphDir(%q) = %v", dir, err)
	}
	// Both parent and child should exist
	for _, d := range []string{filepath.Dir(dir), dir} {
		info, err := os.Stat(d)
		if err != nil {
			t.Fatalf("Stat(%q) = %v", d, err)
		}
		perm := info.Mode().Perm()
		if perm != IssueGraphDirPerm {
			t.Errorf("%s permissions = %04o, want %04o", d, perm, IssueGraphDirPerm)
		}
	}
}

func TestCheckIssueGraphDirPermissions_Secure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".issuegraph")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	CheckIssueGraphDirPermissions(dir)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	if buf.Len() != 0 {
		t.Errorf("expected no warning for 0700 dir, got: %s", buf.String())
	}
}

func TestCheckIssueGraphDirPermissions_Permissive(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".issuegraph")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	CheckIssueGraphDirPermissions(dir)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	want := fmt.Sprintf("Warning: %s has permissions 0755 (recommended: 0700). Run: chmod 700 %s\n", dir, dir)
	if buf.String() != want {
		t.Errorf("warning = %q, want %q", buf.String(), want)
	}
}

func TestCheckIssueGraphDirPermissions_Nonexistent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "no-such-dir")
	// Capture stderr — should produce no output
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	CheckIssueGraphDirPermissions(dir)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	if buf.Len() != 0 {
		t.Errorf("expected no output for nonexistent dir, got: %s", buf.String())
	}
}

func TestFixIssueGraphDirPermissions(t *testing.T) {
	tests := []struct {
		name      string
		startPerm os.FileMode
		wantFixed bool
		wantPerm  os.FileMode
	}{
		{"world_readable_0755", 0755, true, 0700},
		{"world_writable_0777", 0777, true, 0700},
		{"world_only_0707", 0707, true, 0700},
		{"group_only_0770", 0770, true, 0700},
		{"already_secure_0700", 0700, false, 0700},
		{"owner_only_0600", 0600, false, 0600},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), ".issuegraph")
			if err := os.Mkdir(dir, tt.startPerm); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(dir, tt.startPerm); err != nil {
				t.Fatal(err)
			}

			fixed, err := FixIssueGraphDirPermissions(dir)
			if err != nil {
				t.Fatalf("FixIssueGraphDirPermissions() error = %v", err)
			}
			if fixed != tt.wantFixed {
				t.Errorf("fixed = %v, want %v", fixed, tt.wantFixed)
			}

			info, err := os.Stat(dir)
			if err != nil {
				t.Fatalf("Stat() error = %v", err)
			}
			got := info.Mode().Perm()
			if got != tt.wantPerm {
				t.Errorf("permissions after fix = %04o, want %04o", got, tt.wantPerm)
			}
		})
	}
}

func TestFixIssueGraphDirPermissions_Nonexistent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "no-such-dir")
	fixed, err := FixIssueGraphDirPermissions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fixed {
		t.Error("expected fixed=false for nonexistent directory")
	}
}

func TestFixIssueGraphDirPermissions_RejectsSymlink(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(tmp, ".issuegraph")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	fixed, err := FixIssueGraphDirPermissions(link)
	if err == nil {
		t.Fatal("expected error for symbolic link")
	}
	if fixed {
		t.Error("expected fixed=false for symbolic link")
	}
}

func TestFixIssueGraphDirPermissions_RejectsNonDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".issuegraph")
	if err := os.WriteFile(path, []byte("not a directory"), 0o755); err != nil {
		t.Fatal(err)
	}

	fixed, err := FixIssueGraphDirPermissions(path)
	if err == nil {
		t.Fatal("expected error for non-directory")
	}
	if fixed {
		t.Error("expected fixed=false for non-directory")
	}
}

func TestFixIssueGraphDirPermissions_RejectsSymlinkSwap(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".issuegraph")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(tmp, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0o755); err != nil {
		t.Fatal(err)
	}

	fixed, err := fixIssueGraphDirPermissions(path, func(path string) (issuegraphDirHandle, error) {
		if err := os.Remove(path); err != nil {
			t.Fatalf("remove validated directory: %v", err)
		}
		if err := os.Symlink(target, path); err != nil {
			t.Fatalf("replace directory with symlink: %v", err)
		}
		return openIssueGraphDirHandle(path)
	})
	if err == nil {
		t.Fatal("expected error when directory is replaced with a symlink")
	}
	if fixed {
		t.Error("expected fixed=false after symlink swap")
	}
	info, statErr := os.Stat(target)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("symlink target permissions = %04o, want unchanged 0755", got)
	}
}

func TestFixIssueGraphDirPermissions_RejectsDirectorySwap(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".issuegraph")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	replacement := filepath.Join(tmp, "replacement")
	if err := os.Mkdir(replacement, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(replacement, 0o755); err != nil {
		t.Fatal(err)
	}

	fixed, err := fixIssueGraphDirPermissions(path, func(path string) (issuegraphDirHandle, error) {
		if err := os.Remove(path); err != nil {
			t.Fatalf("remove validated directory: %v", err)
		}
		if err := os.Rename(replacement, path); err != nil {
			t.Fatalf("replace validated directory: %v", err)
		}
		return openIssueGraphDirHandle(path)
	})
	if err == nil {
		t.Fatal("expected error when directory is replaced")
	}
	if fixed {
		t.Error("expected fixed=false after directory swap")
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("replacement directory permissions = %04o, want unchanged 0755", got)
	}
}

func TestFixIssueGraphDirPermissions_ChmodUnsupportedLeavesDirectoryUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".issuegraph")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	unsupported := errors.New("descriptor chmod unsupported")

	fixed, err := fixIssueGraphDirPermissions(path, func(path string) (issuegraphDirHandle, error) {
		file, openErr := os.Open(path)
		if openErr != nil {
			return nil, openErr
		}
		return &chmodErrorHandle{File: file, err: unsupported}, nil
	})
	if !errors.Is(err, unsupported) {
		t.Fatalf("error = %v, want wrapped unsupported error", err)
	}
	if fixed {
		t.Error("expected fixed=false when descriptor chmod is unsupported")
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("permissions after failed descriptor chmod = %04o, want unchanged 0755", got)
	}
}

type chmodErrorHandle struct {
	*os.File
	err error
}

func (h *chmodErrorHandle) Chmod(os.FileMode) error {
	return h.err
}
