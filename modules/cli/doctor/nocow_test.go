package doctor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCheckBtrfsNoCOW_NonLinux ensures the check short-circuits to OK on
// non-Linux platforms. The flag doesn't exist there so there is nothing to
// report.
func TestCheckBtrfsNoCOW_NonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("non-Linux short-circuit only applies off Linux")
	}
	result := CheckBtrfsNoCOW(t.TempDir())
	if result.Status != StatusOK {
		t.Errorf("expected StatusOK on non-Linux, got %q: %s", result.Status, result.Message)
	}
}

// TestCheckBtrfsNoCOW_MissingIssueGraphDir reports OK when there is no
// .issuegraph/ directory to check — the check has nothing to examine.
func TestCheckBtrfsNoCOW_MissingIssueGraphDir(t *testing.T) {
	result := CheckBtrfsNoCOW(t.TempDir())
	if result.Status != StatusOK {
		t.Errorf("expected StatusOK when .issuegraph/ is missing, got %q: %s", result.Status, result.Message)
	}
}

// TestCheckBtrfsNoCOW_NotBtrfs ensures we report OK when the .issuegraph/ dir
// exists but lives on a non-btrfs filesystem. We don't want to nag users
// on ext4/xfs/tmpfs where the flag is a no-op.
func TestCheckBtrfsNoCOW_NotBtrfs(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only filesystem detection")
	}
	root := t.TempDir()
	issuegraphDir := filepath.Join(root, ".issuegraph")
	if err := os.MkdirAll(filepath.Join(issuegraphDir, "dolt"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	onBtrfs, err := isBtrfs(issuegraphDir)
	if err != nil {
		t.Fatalf("isBtrfs: %v", err)
	}
	if onBtrfs {
		t.Skip("temp dir is on btrfs; this test targets non-btrfs")
	}
	result := CheckBtrfsNoCOW(root)
	if result.Status != StatusOK {
		t.Errorf("expected StatusOK on non-btrfs, got %q: %s", result.Status, result.Message)
	}
	if !strings.Contains(result.Message, "btrfs") && !strings.Contains(result.Message, "non-Linux") {
		// Accept any of the OK messages; this is a weak check but
		// documents intent.
		t.Logf("non-btrfs OK message: %s", result.Message)
	}
}

// TestCheckBtrfsNoCOW_BtrfsMissingFlag verifies that on a btrfs dolt
// directory without the flag, the check reports a warning. Skipped unless
// the test environment is actually on btrfs.
func TestCheckBtrfsNoCOW_BtrfsMissingFlag(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only")
	}
	root := t.TempDir()
	issuegraphDir := filepath.Join(root, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	onBtrfs, err := isBtrfs(issuegraphDir)
	if err != nil {
		t.Fatalf("isBtrfs: %v", err)
	}
	if !onBtrfs {
		t.Skip("test requires btrfs")
	}

	// Freshly-created subdir should NOT have the flag (unless inherited
	// from a parent that already had it set).
	set, err := hasNoCOW(issuegraphDir)
	if err != nil {
		t.Fatalf("hasNoCOW: %v", err)
	}
	if set {
		t.Skip("temp .issuegraph dir inherited FS_NOCOW_FL from parent; can't test missing state")
	}

	result := CheckBtrfsNoCOW(root)
	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning when flag is missing on btrfs, got %q: %s", result.Status, result.Message)
	}
	if !strings.Contains(result.Fix, "doctor --fix") {
		t.Errorf("expected fix hint to mention 'doctor --fix', got %q", result.Fix)
	}
}

// TestCheckBtrfsNoCOW_BtrfsFlagSet verifies that after applying the flag
// to a btrfs .issuegraph/ directory, the check reports OK. Skipped unless the
// test environment is actually on btrfs.
func TestCheckBtrfsNoCOW_BtrfsFlagSet(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only")
	}
	root := t.TempDir()
	issuegraphDir := filepath.Join(root, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	onBtrfs, err := isBtrfs(issuegraphDir)
	if err != nil {
		t.Fatalf("isBtrfs: %v", err)
	}
	if !onBtrfs {
		t.Skip("test requires btrfs")
	}

	if err := applyNoCOW(issuegraphDir); err != nil {
		t.Fatalf("applyNoCOW: %v", err)
	}

	result := CheckBtrfsNoCOW(root)
	if result.Status != StatusOK {
		t.Errorf("expected StatusOK after applying flag, got %q: %s", result.Status, result.Message)
	}
}

// TestFixBtrfsNoCOW_NonLinux verifies FixBtrfsNoCOW is a no-op off Linux.
func TestFixBtrfsNoCOW_NonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("non-Linux branch only")
	}
	msg, err := FixBtrfsNoCOW(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "not on Linux") {
		t.Errorf("expected skip message, got %q", msg)
	}
}

// TestFixBtrfsNoCOW_MissingIssueGraphDir returns an error when there is no
// .issuegraph/ to fix.
func TestFixBtrfsNoCOW_MissingIssueGraphDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only")
	}
	_, err := FixBtrfsNoCOW(t.TempDir())
	if err == nil {
		t.Errorf("expected error for missing .issuegraph directory, got nil")
	}
}

// TestFixBtrfsNoCOW_Btrfs verifies the fix path applies the flag and
// returns a warning about relocating existing files. Skipped off btrfs.
func TestFixBtrfsNoCOW_Btrfs(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only")
	}
	root := t.TempDir()
	issuegraphDir := filepath.Join(root, ".issuegraph")
	if err := os.MkdirAll(filepath.Join(issuegraphDir, "dolt"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	onBtrfs, err := isBtrfs(issuegraphDir)
	if err != nil {
		t.Fatalf("isBtrfs: %v", err)
	}
	if !onBtrfs {
		t.Skip("test requires btrfs")
	}

	msg, err := FixBtrfsNoCOW(root)
	if err != nil {
		t.Fatalf("FixBtrfsNoCOW: %v", err)
	}
	if !strings.Contains(msg, "FS_NOCOW_FL") {
		t.Errorf("expected fix message to mention FS_NOCOW_FL, got %q", msg)
	}
	if !strings.Contains(msg, "WARNING") {
		t.Errorf("expected fix message to warn about existing files, got %q", msg)
	}

	set, err := hasNoCOW(issuegraphDir)
	if err != nil {
		t.Fatalf("hasNoCOW: %v", err)
	}
	if !set {
		t.Errorf("expected FS_NOCOW_FL to be set after fix")
	}
}
