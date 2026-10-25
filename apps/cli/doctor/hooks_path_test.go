package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setGitConfig sets a git config value in dir, failing the test on error.
func setGitConfig(t *testing.T, dir, key, value string) {
	t.Helper()
	cmd := exec.Command("git", "config", key, value)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config %s %s failed: %v (%s)", key, value, err, strings.TrimSpace(string(out)))
	}
}

// unsetGitConfig unsets a git config value in dir. Ignores "not set" errors
// since the goal is just to make sure the key is absent.
func unsetGitConfig(t *testing.T, dir, key string) {
	t.Helper()
	cmd := exec.Command("git", "config", "--unset", key)
	cmd.Dir = dir
	_ = cmd.Run()
}

// getGitConfig reads a git config value in dir. Returns "" if unset.
func getGitConfig(t *testing.T, dir, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func TestCheckHooksPath_NotInGitRepo(t *testing.T) {
	dir := t.TempDir()
	runInDir(t, dir, func() {
		check := CheckHooksPath()
		if check.Status != StatusOK {
			t.Errorf("expected StatusOK, got %q: %s", check.Status, check.Message)
		}
		if !strings.Contains(check.Message, "N/A") {
			t.Errorf("expected N/A message, got %q", check.Message)
		}
	})
}

func TestCheckHooksPath_Unset(t *testing.T) {
	dir := t.TempDir()
	setupGitRepoInDir(t, dir)
	// setupGitRepoInDir sets core.hooksPath=.git/hooks for isolation; clear it
	// so we exercise the genuinely-unset case.
	unsetGitConfig(t, dir, "core.hooksPath")

	runInDir(t, dir, func() {
		check := CheckHooksPath()
		if check.Status != StatusOK {
			t.Errorf("expected StatusOK, got %q: %s", check.Status, check.Message)
		}
		if !strings.Contains(check.Message, "not set") {
			t.Errorf("expected message to mention core.hooksPath is not set, got %q", check.Message)
		}
	})
}

func TestCheckHooksPath_SetToExistingDir(t *testing.T) {
	dir := t.TempDir()
	setupGitRepoInDir(t, dir)
	// setupGitRepoInDir already points core.hooksPath at .git/hooks, which
	// always exists after git init.

	runInDir(t, dir, func() {
		check := CheckHooksPath()
		if check.Status != StatusOK {
			t.Errorf("expected StatusOK, got %q: %s", check.Status, check.Message)
		}
	})
}

func TestCheckHooksPath_MissingIssueGraphManagedPath(t *testing.T) {
	tests := []struct {
		name        string
		hooksPath   func(dir string) string
		wantAbsHint bool
	}{
		{
			name:      "relative .issuegraph/hooks",
			hooksPath: func(dir string) string { return ".issuegraph/hooks" },
		},
		{
			name:      "relative .issuegraph-hooks",
			hooksPath: func(dir string) string { return ".issuegraph-hooks" },
		},
		{
			name:        "absolute .issuegraph/hooks",
			hooksPath:   func(dir string) string { return filepath.Join(dir, ".issuegraph", "hooks") },
			wantAbsHint: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setupGitRepoInDir(t, dir)
			hooksPath := tt.hooksPath(dir)
			setGitConfig(t, dir, "core.hooksPath", hooksPath)

			runInDir(t, dir, func() {
				check := CheckHooksPath()
				if check.Status != StatusWarning {
					t.Fatalf("expected StatusWarning, got %q: %s", check.Status, check.Message)
				}
				if !strings.Contains(check.Fix, "issuegraph doctor --fix") {
					t.Errorf("expected issuegraph-managed fix hint, got %q", check.Fix)
				}
				if !strings.Contains(check.Detail, hooksPath) {
					t.Errorf("expected detail to mention configured value %q, got %q", hooksPath, check.Detail)
				}

				if err := FixHooksPath(); err != nil {
					t.Fatalf("FixHooksPath failed: %v", err)
				}
			})

			if got := getGitConfig(t, dir, "core.hooksPath"); got != "" {
				t.Errorf("expected core.hooksPath to be unset after FixHooksPath, got %q", got)
			}
		})
	}
}

func TestCheckHooksPath_MissingNonIssueGraphPath(t *testing.T) {
	dir := t.TempDir()
	setupGitRepoInDir(t, dir)
	setGitConfig(t, dir, "core.hooksPath", ".husky/_")

	runInDir(t, dir, func() {
		check := CheckHooksPath()
		if check.Status != StatusWarning {
			t.Fatalf("expected StatusWarning, got %q: %s", check.Status, check.Message)
		}
		if strings.Contains(check.Fix, "issuegraph doctor --fix") {
			t.Errorf("did not expect issuegraph auto-fix to be offered for a non-issuegraph-managed path, got %q", check.Fix)
		}
		if !strings.Contains(check.Fix, "not issuegraph-managed") {
			t.Errorf("expected fix text to say the path is not issuegraph-managed, got %q", check.Fix)
		}

		// The important guard: FixHooksPath must be a no-op for a third-party
		// (e.g. husky) hooksPath, even though its target is also missing.
		if err := FixHooksPath(); err != nil {
			t.Fatalf("FixHooksPath returned error: %v", err)
		}
	})

	if got := getGitConfig(t, dir, "core.hooksPath"); got != ".husky/_" {
		t.Errorf("expected core.hooksPath to remain %q after FixHooksPath, got %q", ".husky/_", got)
	}
}

func TestCheckHooksPath_ExistingNonIssueGraphPath(t *testing.T) {
	dir := t.TempDir()
	setupGitRepoInDir(t, dir)
	huskyDir := filepath.Join(dir, ".husky", "_")
	if err := os.MkdirAll(huskyDir, 0755); err != nil {
		t.Fatalf("failed to create husky dir: %v", err)
	}
	setGitConfig(t, dir, "core.hooksPath", ".husky/_")

	runInDir(t, dir, func() {
		check := CheckHooksPath()
		if check.Status != StatusOK {
			t.Errorf("expected StatusOK for an existing third-party hooksPath, got %q: %s", check.Status, check.Message)
		}
	})
}

func TestFixHooksPath_NoOpWhenTargetExists(t *testing.T) {
	dir := t.TempDir()
	setupGitRepoInDir(t, dir)
	issuegraphHooksDir := filepath.Join(dir, ".issuegraph", "hooks")
	if err := os.MkdirAll(issuegraphHooksDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph/hooks: %v", err)
	}
	setGitConfig(t, dir, "core.hooksPath", ".issuegraph/hooks")

	runInDir(t, dir, func() {
		if err := FixHooksPath(); err != nil {
			t.Fatalf("FixHooksPath failed: %v", err)
		}
	})

	if got := getGitConfig(t, dir, "core.hooksPath"); got != ".issuegraph/hooks" {
		t.Errorf("expected core.hooksPath to remain %q when target exists, got %q", ".issuegraph/hooks", got)
	}
}

// TestCheckHooksPath_SymlinkedRepoRoot pins the fix for the macOS-red build at
// 868dd077a: git canonicalizes the repo root it reports, but the configured
// core.hooksPath is whatever string was written, so an absolute issuegraph-managed
// path reached through a symlink used to string-compare unequal and be
// misreported as third-party — leaving FixHooksPath refusing to unset it.
//
// macOS hits this on every temp dir (/var is a symlink to /private/var), which
// is how CI caught it, but nothing about it is macOS-specific: an explicit
// symlink reproduces it anywhere git resolves the root.
func TestCheckHooksPath_SymlinkedRepoRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires elevated privileges on Windows")
	}

	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatalf("failed to create real repo dir: %v", err)
	}
	linkDir := filepath.Join(base, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("failed to symlink %s -> %s: %v", linkDir, realDir, err)
	}
	setupGitRepoInDir(t, realDir)

	// The value a user (or an older issuegraph) wrote: absolute, and carrying the
	// unresolved symlink prefix. Its target does not exist, so it is dangling.
	hooksPath := filepath.Join(linkDir, ".issuegraph", "hooks")
	setGitConfig(t, realDir, "core.hooksPath", hooksPath)

	runInDir(t, realDir, func() {
		check := CheckHooksPath()
		if check.Status != StatusWarning {
			t.Fatalf("expected StatusWarning, got %q: %s", check.Status, check.Message)
		}
		if !strings.Contains(check.Fix, "issuegraph doctor --fix") {
			t.Errorf("expected issuegraph-managed fix hint for a symlinked repo root, got %q", check.Fix)
		}
		if err := FixHooksPath(); err != nil {
			t.Fatalf("FixHooksPath failed: %v", err)
		}
	})

	if got := getGitConfig(t, realDir, "core.hooksPath"); got != "" {
		t.Errorf("expected core.hooksPath to be unset after FixHooksPath, got %q", got)
	}
}

// TestIsIssueGraphManagedHooksPath_LeavesThirdPartyAlone guards the widened
// symlink-resolving match against over-reach: resolving paths must not make
// an unrelated hooks directory look issuegraph-managed, since a false positive here
// means issuegraph unsets a hooksPath it does not own (husky, lefthook).
func TestIsIssueGraphManagedHooksPath_LeavesThirdPartyAlone(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name      string
		hooksPath string
		want      bool
	}{
		{"relative issuegraph hooks", ".issuegraph/hooks", true},
		{"relative shared issuegraph hooks", ".issuegraph-hooks", true},
		{"absolute issuegraph hooks", filepath.Join(root, ".issuegraph", "hooks"), true},
		{"absolute shared issuegraph hooks", filepath.Join(root, ".issuegraph-hooks"), true},
		{"relative husky", ".husky/_", false},
		{"relative githooks", ".githooks", false},
		{"absolute husky", filepath.Join(root, ".husky", "_"), false},
		{"issuegraph hooks under a different repo", filepath.Join(t.TempDir(), ".issuegraph", "hooks"), false},
		{"deeper path below issuegraph hooks", filepath.Join(root, ".issuegraph", "hooks", "pre-commit"), false},
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsIssueGraphManagedHooksPath(root, tc.hooksPath); got != tc.want {
				t.Errorf("IsIssueGraphManagedHooksPath(%q, %q) = %v, want %v", root, tc.hooksPath, got, tc.want)
			}
		})
	}
}
