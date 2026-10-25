// Package issuegraph context.go provides centralized repository context resolution.
//
// Problem: 50+ git commands across the codebase assume CWD is the repository root.
// When ISSUEGRAPH_DIR points to a different repo, or when running from a worktree,
// these commands execute in the wrong directory.
//
// Solution: RepoContext provides a single source of truth for repository paths,
// with methods that ensure git commands run in the correct repository.
//
// Usage:
//
//	rc, err := issuegraph.GetRepoContext()
//	if err != nil {
//	    return err
//	}
//	cmd := rc.GitCmd(ctx, "status")  // Runs in issuegraph repo, not CWD
//
// See engdocs/REPO_CONTEXT.md for detailed documentation.
package issuegraph

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/borg0ai/dx-issuegraph/internal/git"
)

// UserRole represents the user's relationship to a repository.
// Used to determine appropriate behaviors for fork contributors vs maintainers.
type UserRole string

// Role constants for user relationship to repository.
const (
	// Contributor indicates the user is contributing to a fork (not the maintainer).
	// ISSUEGRAPH_DIR redirection implies contributor role automatically.
	Contributor UserRole = "contributor"

	// Maintainer indicates the user owns/maintains the repository.
	Maintainer UserRole = "maintainer"
)

// ErrRoleNotConfigured is returned when issuegraph.role is not set in git config.
// This signals that the init prompt should be shown to configure the role.
var ErrRoleNotConfigured = errors.New("issuegraph.role not configured in git config")

// RepoContext holds resolved repository paths for issuegraph operations.
//
// The struct distinguishes between:
//   - RepoRoot: where .issuegraph/ lives (for git operations on issuegraph data)
//   - CWDRepoRoot: where user is working (for status display, etc.)
//
// These may differ when ISSUEGRAPH_DIR points to a different repository,
// or when running from a git worktree.
type RepoContext struct {
	// IssueGraphDir is the actual .issuegraph directory path (after following redirects).
	IssueGraphDir string

	// RepoRoot is the repository root containing IssueGraphDir.
	// Git commands for issuegraph operations should run here.
	RepoRoot string

	// CWDRepoRoot is the repository root containing the current working directory.
	// May differ from RepoRoot when ISSUEGRAPH_DIR points elsewhere.
	CWDRepoRoot string

	// IsRedirected is true if IssueGraphDir resolves to a different repository than CWD.
	// This covers explicit ISSUEGRAPH_DIR usage and redirect files.
	IsRedirected bool

	// IsWorktree is true if CWD is in a git worktree.
	IsWorktree bool
}

var (
	repoCtx     *RepoContext
	repoCtxOnce sync.Once
	repoCtxErr  error
)

// GetRepoContext returns the cached repository context, initializing it on first call.
//
// The context is cached because:
// 1. CWD doesn't change during command execution
// 2. ISSUEGRAPH_DIR doesn't change during command execution
// 3. Repeated filesystem access would be wasteful
//
// Returns an error if no .issuegraph directory can be found.
func GetRepoContext() (*RepoContext, error) {
	repoCtxOnce.Do(func() {
		repoCtx, repoCtxErr = buildRepoContext()
	})
	return repoCtx, repoCtxErr
}

// buildRepoContext constructs the RepoContext by resolving all paths.
// This is called once per process via sync.Once.
func buildRepoContext() (*RepoContext, error) {
	// 1. Find .issuegraph directory (respects ISSUEGRAPH_DIR env var)
	issuegraphDir := FindIssueGraphDir()
	if issuegraphDir == "" {
		return nil, fmt.Errorf("no .issuegraph directory found")
	}

	// 2. Security: Validate path boundary (SEC-003)
	if !isPathInSafeBoundary(issuegraphDir) {
		return nil, fmt.Errorf("ISSUEGRAPH_DIR points to unsafe location: %s", issuegraphDir)
	}

	// 3. Check for redirect file in the local repo
	redirectInfo := GetRedirectInfo()

	// 4. Determine RepoRoot based on external/redirect status
	var repoRoot string
	isExternal := redirectInfo.IsRedirected
	if !isExternal {
		if external, err := isExternalIssueGraphDir(issuegraphDir); err == nil {
			isExternal = external
		}
	}

	if isExternal {
		// IssueGraph dir is in a different repo - use that repo's root
		repoRoot = repoRootForIssueGraphDir(issuegraphDir)
	} else {
		// Normal case - find repo root via git
		var err error
		repoRoot, err = git.GetMainRepoRoot()
		if err != nil {
			return nil, fmt.Errorf("cannot determine repository root: %w", err)
		}
	}

	// 5. Get CWD's repo root (may differ from RepoRoot)
	cwdRepoRoot := git.GetRepoRoot() // Returns "" if not in git repo

	// 6. Check worktree status
	isWorktree := git.IsWorktree()

	return &RepoContext{
		IssueGraphDir: issuegraphDir,
		RepoRoot:      repoRoot,
		CWDRepoRoot:   cwdRepoRoot,
		IsRedirected:  isExternal,
		IsWorktree:    isWorktree,
	}, nil
}

// isExternalIssueGraphDir returns true if issuegraphDir is in a different git repo than CWD.
// Uses git common dir to correctly handle worktrees and bare repos.
func isExternalIssueGraphDir(issuegraphDir string) (bool, error) {
	cwdCommonDir, err := git.GetGitCommonDir()
	if err != nil {
		return false, err
	}

	issuegraphCommonDir, err := getGitCommonDirForPath(issuegraphDir)
	if err != nil {
		return false, err
	}

	return cwdCommonDir != issuegraphCommonDir, nil
}

// getGitCommonDirForPath returns the shared git directory for a path.
// For worktrees, this returns the shared git directory (common to all worktrees).
func getGitCommonDirForPath(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git common dir for %s: %w", path, err)
	}
	result := strings.TrimSpace(string(output))

	if !filepath.IsAbs(result) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path for %s: %w", path, err)
		}
		result = filepath.Join(absPath, result)
	}

	result = filepath.Clean(result)
	if resolved, err := filepath.EvalSymlinks(result); err == nil {
		result = resolved
	}

	return result, nil
}

// repoRootForIssueGraphDir returns the repository root for a issuegraph directory.
// Falls back to the issuegraphDir parent if git lookup fails.
func repoRootForIssueGraphDir(issuegraphDir string) string {
	repoRoot, err := getRepoRootFromPath(issuegraphDir)
	if err == nil && repoRoot != "" {
		return repoRoot
	}
	return filepath.Dir(issuegraphDir)
}

// getRepoRootFromPath returns the git repository root for a given path.
func getRepoRootFromPath(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git root for %s: %w", path, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GitCmd creates an exec.Cmd configured to run git in the issuegraph repository.
//
// This method sets cmd.Dir to RepoRoot, ensuring git commands operate on
// the correct repository regardless of CWD.
//
// Security: Git hooks and templates are disabled to prevent code execution
// in potentially malicious repositories (SEC-001, SEC-002).
//
// Pattern:
//
//	cmd := rc.GitCmd(ctx, "add", ".issuegraph/")
//	output, err := cmd.Output()
//
// Equivalent to running: cd $RepoRoot && git add .issuegraph/
//
// GH#2538: When running from a git worktree, git may inherit environment
// variables that point to the worktree's .git instead of the main repo.
// We explicitly set GIT_DIR and GIT_WORK_TREE to ensure git operates on
// the correct repository (the one containing .issuegraph/).
func (rc *RepoContext) GitCmd(ctx context.Context, args ...string) *exec.Cmd {
	gitArgs := append([]string{"-c", "core.hooksPath="}, args...)
	cmd := exec.CommandContext(ctx, "git", gitArgs...)
	cmd.Dir = rc.RepoRoot

	// GH#2538: Ensure git uses the target repository, not the worktree we may be running from.
	// This fixes "pathspec outside repository" errors when issuegraph sync runs from a worktree.
	gitDir := filepath.Join(rc.RepoRoot, ".git")

	// Security: Disable git hooks and templates to prevent code execution
	// in potentially malicious repositories (SEC-001, SEC-002)
	cmd.Env = append(os.Environ(),
		"GIT_TEMPLATE_DIR=",          // Disable templates
		"GIT_DIR="+gitDir,            // Ensure git uses the correct .git directory
		"GIT_WORK_TREE="+rc.RepoRoot, // Ensure git uses the correct work tree
	)
	return cmd
}

// GitCmdCWD creates an exec.Cmd configured to run git in the user's working repository.
//
// Use this for git commands that should reflect the user's current context,
// such as showing status or checking for uncommitted changes in their working repo.
//
// If CWD is not in a git repository, cmd.Dir is left unset (uses process CWD).
func (rc *RepoContext) GitCmdCWD(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	if rc.CWDRepoRoot != "" {
		cmd.Dir = rc.CWDRepoRoot
	}
	return cmd
}

// RelPath returns the given absolute path relative to the issuegraph repository root.
//
// Useful for displaying paths to users in a consistent, repo-relative format.
// Returns an error if the path is not within the repository.
func (rc *RepoContext) RelPath(absPath string) (string, error) {
	return filepath.Rel(rc.RepoRoot, absPath)
}

// ResetCaches clears the cached RepoContext, forcing re-resolution on next call.
//
// This is intended for tests that need to change directory or ISSUEGRAPH_DIR
// between test cases. In production, the cache is safe because these
// values don't change during command execution.
//
// WARNING: Not thread-safe. Only call from single-threaded test contexts.
//
// Usage in tests:
//
//	t.Cleanup(func() {
//	    issuegraph.ResetCaches()
//	    git.ResetCaches()
//	})
func ResetCaches() {
	repoCtxOnce = sync.Once{}
	repoCtx = nil
	repoCtxErr = nil
}

// unsafePrefixes lists system directories that ISSUEGRAPH_DIR should never point to.
// This prevents path traversal attacks (SEC-003).
var unsafePrefixes = []string{
	"/etc", "/usr", "/var", "/root", "/System", "/Library",
	"/bin", "/sbin", "/opt", "/private",
}

// isPathInSafeBoundary validates that a path is not in sensitive system directories.
// Returns false if the path is in an unsafe location (SEC-003).
func isPathInSafeBoundary(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Allow OS-designated temp directories (e.g., /var/folders on macOS, which
	// symlinks to /private/var/folders). World-writable, so resolve symlinks
	// before admitting: a symlink planted under the temp dir whose target
	// escapes the boundary must be rejected, not followed into a system
	// directory — same treatment as the /Users/Shared carve-out below
	// (be-kghzr SEC-003 hardening).
	// The carve-out must admit both spellings of the temp root: os.TempDir()
	// itself (on macOS the symlinked /var/folders/... form) and its physical
	// resolution (/private/var/folders/...). A caller-supplied path that has
	// already been symlink-resolved arrives in the physical form and would
	// otherwise skip this branch and be rejected by the /private deny prefix
	// below.
	tempDir := strings.TrimSuffix(os.TempDir(), "/")
	physTempDir := strings.TrimSuffix(resolveLongestExistingAncestor(tempDir), "/")
	if absPath == tempDir || strings.HasPrefix(absPath, tempDir+"/") ||
		absPath == physTempDir || strings.HasPrefix(absPath, physTempDir+"/") {
		return resolvedPathWithinRoot(absPath, tempDir)
	}

	// Allow /var/home as a valid user home directory (Fedora Silverblue, Bluefin, etc.)
	if strings.HasPrefix(absPath, "/var/home/") {
		return true
	}

	// Allow /var/tmp as the FHS-standard secondary temp directory (persists across
	// reboots, unlike /tmp). This is distinct from the os.TempDir() carve-out
	// above: a machine's build tooling can set GOTMPDIR to redirect Go's own
	// test/compile temp dirs under /var/tmp even while os.TempDir() itself still
	// reports /tmp, so t.TempDir() in a test binary can land here without the
	// os.TempDir() check ever seeing it (be-odye4). Like /Users/Shared, /var/tmp is
	// world-writable (drwxrwxrwt), so resolve symlinks before admitting (SEC-003):
	// a symlink planted under it must not be followed into a rejected directory.
	if absPath == "/var/tmp" || strings.HasPrefix(absPath, "/var/tmp/") {
		return resolvedPathWithinRoot(absPath, "/var/tmp")
	}

	for _, prefix := range unsafePrefixes {
		if strings.HasPrefix(absPath, prefix+"/") || absPath == prefix {
			return false
		}
	}
	// macOS's /Users/Shared is the OS-designated shared directory, not a peer
	// user's home — allow it (and its subpaths) before the peer-home rejection
	// below. SEC-003 guards against path traversal into system directories; the
	// unsafePrefixes blocklist above stays authoritative, so this carve-out only
	// admits the shared dir, mirroring the /var/home/ allowance. /Users/Shared is
	// world-writable (drwxrwxrwt), so resolve symlinks before admitting: a symlink
	// planted under it whose target escapes the boundary must be rejected, not
	// followed into a system directory (be-vc1 SEC-003 hardening).
	if absPath == "/Users/Shared" || strings.HasPrefix(absPath, "/Users/Shared/") {
		return resolvedPathWithinRoot(absPath, "/Users/Shared")
	}

	// Also reject other users' home directories.
	if strings.HasPrefix(absPath, "/Users/") || strings.HasPrefix(absPath, "/home/") || strings.HasPrefix(absPath, "/var/home/") {
		// Resolve the current user's home from the account database, which is
		// not affected by $HOME manipulation. Fall back to $HOME when that
		// lookup is unavailable (e.g. CGO-free builds where the user is not in
		// /etc/passwd); leaving homeDir empty here would skip the check and
		// fail open, which is worse than trusting $HOME.
		homeDir := ""
		if u, err := user.Current(); err == nil {
			homeDir = u.HomeDir
		}
		if homeDir == "" {
			homeDir, _ = os.UserHomeDir()
		}
		if homeDir != "" {
			home := strings.TrimSuffix(homeDir, "/")
			// Compare on a path boundary so a sibling like /home/aliceXX is
			// not treated as inside /home/alice.
			if absPath != home && !strings.HasPrefix(absPath, home+"/") {
				return false
			}
		}
	}
	return true
}

// resolveLongestExistingAncestor canonicalizes path by resolving symlinks on its
// longest existing ancestor and re-appending the trailing segments that do not
// exist yet. Unlike a bare filepath.EvalSymlinks (which fails on a non-existent
// path and leaves it unresolved), this lets a not-yet-created ISSUEGRAPH_DIR still be
// canonicalized against a real, symlink-free root. The upward walk mirrors the
// filepath.Dir loops elsewhere in this package.
func resolveLongestExistingAncestor(path string) string {
	cur := filepath.Clean(path)
	remainder := ""
	for {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			if remainder == "" {
				return resolved
			}
			return filepath.Join(resolved, remainder)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			// Reached the filesystem root without resolving anything; return the
			// cleaned input unchanged (best effort).
			return filepath.Clean(path)
		}
		remainder = filepath.Join(filepath.Base(cur), remainder)
		cur = parent
	}
}

// resolvedPathWithinRoot reports whether absPath, after symlink resolution, still
// lies within root. Both sides are resolved via resolveLongestExistingAncestor so
// the comparison is symlink-safe and works for not-yet-created paths: a symlink
// under root whose target escapes root resolves outside and returns false, while
// a real (or not-yet-created) subpath of a non-symlinked root returns true.
//
// This hardens the /Users/Shared carve-out (be-vc1, SEC-003): /Users/Shared is
// world-writable, so a co-located user could plant a symlink there pointing at a
// system directory; matching on the unresolved path would admit it. Resolving
// first closes that path-traversal vector. Resolving root too is a no-op for the
// real /Users/Shared but is required for temp-dir-rooted tests on macOS, where
// the temp dir lives under the symlinked /var.
func resolvedPathWithinRoot(absPath, root string) bool {
	resolved := resolveLongestExistingAncestor(absPath)
	resolvedRoot := resolveLongestExistingAncestor(root)
	return resolved == resolvedRoot || strings.HasPrefix(resolved, resolvedRoot+string(filepath.Separator))
}

// GetRepoContextForWorkspace returns a fresh RepoContext for a specific workspace.
//
// Unlike GetRepoContext(), this function:
//   - Does NOT cache results (caller may handle multiple workspaces)
//   - Does NOT respect ISSUEGRAPH_DIR (workspace path is explicit)
//   - Resolves worktree relationships correctly
//
// This is designed for processes that need to handle
// multiple workspaces or detect context changes.
//
// The function temporarily changes to the workspace directory to resolve paths,
// then restores the original directory.
func GetRepoContextForWorkspace(workspacePath string) (*RepoContext, error) {
	// Normalize workspace path
	absWorkspace, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve workspace path %s: %w", workspacePath, err)
	}

	// Change to workspace directory temporarily
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(absWorkspace); err != nil {
		return nil, fmt.Errorf("cannot access workspace %s: %w", absWorkspace, err)
	}

	// Clear git caches for fresh resolution
	git.ResetCaches()

	// Build context fresh, specifically for this workspace (ignores ISSUEGRAPH_DIR)
	return buildRepoContextForWorkspace(absWorkspace)
}

// buildRepoContextForWorkspace constructs RepoContext for a specific workspace.
// Unlike buildRepoContext(), this ignores ISSUEGRAPH_DIR env var since the workspace
// path is explicitly provided.
func buildRepoContextForWorkspace(workspacePath string) (*RepoContext, error) {
	// 1. Determine if we're in a worktree and find the main repo root
	var repoRoot string
	var isWorktree bool

	if git.IsWorktree() {
		isWorktree = true
		var err error
		repoRoot, err = git.GetMainRepoRoot()
		if err != nil {
			return nil, fmt.Errorf("cannot determine main repository root: %w", err)
		}
	} else {
		isWorktree = false
		repoRoot = git.GetRepoRoot()
		if repoRoot == "" {
			return nil, fmt.Errorf("workspace %s is not in a git repository", workspacePath)
		}
	}

	// 2. Find .issuegraph directory in the appropriate location
	issuegraphDir := filepath.Join(repoRoot, ".issuegraph")

	// Check if .issuegraph exists
	if _, err := os.Stat(issuegraphDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no .issuegraph directory found at %s", issuegraphDir)
	}

	// 3. Follow redirect if present
	issuegraphDir = FollowRedirect(issuegraphDir)

	// 4. Security: Validate path boundary (SEC-003)
	if !isPathInSafeBoundary(issuegraphDir) {
		return nil, fmt.Errorf("issuegraph directory in unsafe location: %s", issuegraphDir)
	}

	// 5. Validate directory contains actual project files
	if !hasIssueGraphProjectFiles(issuegraphDir) {
		return nil, fmt.Errorf("issuegraph directory missing required files: %s", issuegraphDir)
	}

	// 6. Get CWD's repo root (same as workspace in this case)
	cwdRepoRoot := git.GetRepoRoot()

	return &RepoContext{
		IssueGraphDir: issuegraphDir,
		RepoRoot:      repoRoot,
		CWDRepoRoot:   cwdRepoRoot,
		IsRedirected:  false, // Workspace-specific context is never "redirected"
		IsWorktree:    isWorktree,
	}, nil
}

// Validate checks if the cached context is still valid.
//
// Returns an error if IssueGraphDir or RepoRoot no longer exist. This is useful
// for long-running processes that need to detect when context becomes stale (DMN-002).
func (rc *RepoContext) Validate() error {
	if _, err := os.Stat(rc.IssueGraphDir); os.IsNotExist(err) {
		return fmt.Errorf("IssueGraphDir no longer exists: %s", rc.IssueGraphDir)
	}
	if _, err := os.Stat(rc.RepoRoot); os.IsNotExist(err) {
		return fmt.Errorf("RepoRoot no longer exists: %s", rc.RepoRoot)
	}
	return nil
}

// GitOutput runs a git command in the issuegraph repository and returns its output.
//
// This is a convenience wrapper around GitCmd that captures stdout.
// Returns an error if the command fails or produces no output.
//
// Pattern:
//
//	output, err := rc.GitOutput(ctx, "config", "--get", "issuegraph.role")
//	if err != nil {
//	    // Config key not set or git error
//	}
func (rc *RepoContext) GitOutput(ctx context.Context, args ...string) (string, error) {
	cmd := rc.GitCmd(ctx, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// Role reads issuegraph.role from git config (fresh each call, ~1ms).
//
// If ISSUEGRAPH_DIR is set (IsRedirected), returns Contributor implicitly
// because external repo mode always indicates a contributor workflow.
//
// Returns ("", false) if role is not configured and not redirected.
// The bool return indicates whether a role was determined.
func (rc *RepoContext) Role() (UserRole, bool) {
	// ISSUEGRAPH_DIR implies contributor (external repo mode)
	if rc.IsRedirected {
		return Contributor, true
	}

	output, err := rc.GitOutput(context.Background(), "config", "--get", "issuegraph.role")
	if err != nil {
		return "", false // Not configured
	}
	return UserRole(strings.TrimSpace(output)), true
}

// IsContributor returns true if user is configured as contributor.
//
// This includes both explicit configuration (git config issuegraph.role contributor)
// and implicit detection (ISSUEGRAPH_DIR redirect active).
func (rc *RepoContext) IsContributor() bool {
	role, ok := rc.Role()
	return ok && role == Contributor
}

// IsMaintainer returns true if user is configured as maintainer.
//
// Only returns true for explicit configuration (git config issuegraph.role maintainer).
// ISSUEGRAPH_DIR redirect always implies contributor, never maintainer.
func (rc *RepoContext) IsMaintainer() bool {
	role, ok := rc.Role()
	return ok && role == Maintainer
}

// RequireRole returns error if role not configured (forces init prompt).
//
// Use this at command entry points that need role-aware behavior.
// If ISSUEGRAPH_DIR is set, role is implicitly determined (contributor),
// so this will not return an error.
func (rc *RepoContext) RequireRole() error {
	if _, ok := rc.Role(); !ok {
		return ErrRoleNotConfigured
	}
	return nil
}
