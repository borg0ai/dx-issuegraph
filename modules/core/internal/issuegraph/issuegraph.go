// Package issuegraph provides a minimal public API for extending issuegraph with custom orchestration.
//
// Most extensions should use direct SQL queries against issuegraph's database.
// This package exports only the essential types and functions needed for
// Go-based extensions that want to use issuegraph's storage layer programmatically.
//
// For a working extension example, see examples/issuegraph-example-extension-go.
package issuegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/git"
	"github.com/borg0ai/dx-issuegraph/internal/legacydir"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/storage/backends"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

// CanonicalDatabaseName is the required database filename for all issuegraph repositories
const CanonicalDatabaseName = "issuegraph.db"

// RedirectFileName is the name of the file that redirects to another .issuegraph directory
const RedirectFileName = "redirect"

// SourceDatabaseInfo contains the dolt_database name from a source .issuegraph/metadata.json,
// preserved across a redirect so that the source directory's database identity is not
// lost when the redirect target has a different dolt_database.
//
// When a .issuegraph/redirect points to a shared .issuegraph directory that serves multiple
// databases, the source's metadata.json may specify a different dolt_database than
// the target's. This struct captures the source database name so callers can
// restore it after redirect resolution.
type SourceDatabaseInfo struct {
	// SourceDir is the original .issuegraph directory (before redirect)
	SourceDir string
	// TargetDir is the resolved .issuegraph directory (after redirect)
	TargetDir string
	// WasRedirected is true if a redirect was followed
	WasRedirected bool
	// SourceDatabase is dolt_database from the source metadata.json (raw field,
	// NOT the env-var-aware GetDoltDatabase()). Empty if no source metadata exists
	// or the source has no dolt_database configured.
	SourceDatabase string
}

// ResolveRedirect follows a .issuegraph/redirect file and captures the source directory's
// dolt_database from metadata.json BEFORE following the redirect. This preserves
// the source database identity across redirects.
//
// The env var ISSUEGRAPH_DOLT_SERVER_DATABASE still takes highest priority (handled by
// GetDoltDatabase() in callers). This function only captures the raw config field
// so callers can use it as an override when the env var is not set.
//
// Returns SourceDatabaseInfo with WasRedirected=true if a redirect was followed,
// and SourceDatabase set to the source's dolt_database (if any).
func ResolveRedirect(issuegraphDir string) SourceDatabaseInfo {
	info := SourceDatabaseInfo{
		SourceDir: issuegraphDir,
		TargetDir: issuegraphDir,
	}

	// Read source metadata.json directly (NOT via configfile.Load which may trigger
	// Dolt connections or recursive FollowRedirect calls causing deadlocks).
	// We only need the raw dolt_database field.
	metadataPath := filepath.Join(issuegraphDir, "metadata.json")
	if data, err := os.ReadFile(metadataPath); err == nil {
		var raw struct {
			DoltDatabase string `json:"dolt_database"`
		}
		if json.Unmarshal(data, &raw) == nil {
			info.SourceDatabase = raw.DoltDatabase
		}
	}

	// Follow redirect
	resolved := FollowRedirect(issuegraphDir)
	if resolved != issuegraphDir {
		info.WasRedirected = true
		info.TargetDir = resolved
	}

	return info
}

// FollowRedirect checks if a .issuegraph directory contains a redirect file and follows it.
// If a redirect file exists, it returns the target .issuegraph directory path.
// If no redirect exists or there's an error, it returns the original path unchanged.
//
// The redirect file should contain a single path (relative or absolute) to the target
// .issuegraph directory. Relative paths are resolved from the parent directory of the
// original .issuegraph directory (i.e., the project root).
//
// Redirect chains are not followed - only one level of redirection is supported.
// This prevents infinite loops and keeps the behavior predictable.
func FollowRedirect(issuegraphDir string) string {
	redirectFile := filepath.Join(issuegraphDir, RedirectFileName)
	data, err := os.ReadFile(redirectFile)
	if err != nil {
		// No redirect file or can't read it - use original path
		return issuegraphDir
	}

	// Parse the redirect target (trim whitespace and handle comments)
	target := strings.TrimSpace(string(data))

	// Skip empty lines and comments to find the actual path
	lines := strings.Split(target, "\n")
	target = ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			target = line
			break
		}
	}

	if target == "" {
		return issuegraphDir
	}

	// Resolve relative paths from the parent of the .issuegraph directory (project root)
	if !filepath.IsAbs(target) {
		projectRoot := filepath.Dir(issuegraphDir)
		target = filepath.Join(projectRoot, target)
	}

	// Canonicalize the target path and prefer a stable branch worktree when the
	// redirect points at a detached snapshot checkout.
	target = canonicalizeIssueGraphDirPath(target)

	// Verify the target exists and is a directory
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		// Invalid redirect target - fall back to original
		fmt.Fprintf(os.Stderr, "Warning: redirect target does not exist or is not a directory: %s\n", target)
		return issuegraphDir
	}

	// Defense-in-depth (borg0ai/dx-issuegraph#4692): a redirect can end up
	// pointing at a directory with no database at all, e.g. a stray
	// worktree-depth redirect written by the "graceful server-to-embedded
	// fallback" path (related to the "issuegraph worktree create" write-site removed
	// in #3051). Following such a redirect silently lands issuegraph on an empty,
	// unrelated location and `issuegraph list`/`issuegraph show` report no issues even
	// though the real data is untouched elsewhere. docs/reference/advanced.md
	// ("Database Redirects") already documents the contract: "The target
	// directory must exist and contain a valid database" -- enforce that
	// here instead of trusting any redirect file blindly.
	//
	// This intentionally does NOT look at the source directory's own mode:
	// a server-mode source rig redirecting to a shared Gas Town root (each
	// supplying its own dolt_database via ResolveRedirect/fb51196f7) is a
	// documented, supported topology, not a staleness signal.
	//
	// hasIssueGraphProjectFiles treats bare presence of metadata.json in the
	// target as sufficient, even if it later fails to parse: a
	// present-but-corrupt metadata.json is a config problem, not a
	// missing-database problem, and store_factory.go's
	// newDoltStoreFromConfig already hard-errors loudly on an unloadable
	// metadata.json rather than silently falling back to the embedded store.
	if !hasIssueGraphProjectFiles(target) {
		warnInvalidRedirectTargetOnce(issuegraphDir, target)
		return issuegraphDir
	}

	// Prevent redirect chains - don't follow if target also has a redirect
	targetRedirect := filepath.Join(target, RedirectFileName)
	if _, err := os.Stat(targetRedirect); err == nil {
		fmt.Fprintf(os.Stderr, "Warning: redirect chains not allowed, ignoring redirect in %s\n", target)
	}

	if os.Getenv("ISSUEGRAPH_DEBUG_ROUTING") != "" {
		fmt.Fprintf(os.Stderr, "[routing] Followed redirect from %s -> %s\n", issuegraphDir, target)
	}

	return target
}

// invalidRedirectTargetWarned tracks source issuegraphDir paths that have already
// received the "ignoring redirect: invalid target" warning in this process,
// so warnInvalidRedirectTargetOnce doesn't spam stderr. FollowRedirect is
// called many times per issuegraph invocation (CWD walk, routing, worktree
// fallbacks), all for the same source directory.
var invalidRedirectTargetWarned sync.Map

// warnInvalidRedirectTargetOnce emits the invalid-redirect-target warning at
// most once per source issuegraphDir per process.
func warnInvalidRedirectTargetOnce(issuegraphDir, target string) {
	if _, alreadyWarned := invalidRedirectTargetWarned.LoadOrStore(issuegraphDir, struct{}{}); alreadyWarned {
		return
	}
	fmt.Fprintf(os.Stderr, "Warning: ignoring redirect from %s to %s because the target has no database or metadata.json; fix or delete the redirect file\n", issuegraphDir, target)
}

func canonicalizeIssueGraphDirPath(issuegraphDir string) string {
	canonical := utils.CanonicalizePath(issuegraphDir)
	if stable := preferStableBranchWorktreeIssueGraphDir(canonical); stable != "" {
		return stable
	}
	return canonical
}

type worktreeInfo struct {
	Path     string
	Head     string
	Branch   string
	Detached bool
	Bare     bool
}

func preferStableBranchWorktreeIssueGraphDir(issuegraphDir string) string {
	if filepath.Base(issuegraphDir) != ".issuegraph" {
		return ""
	}

	repoRoot := filepath.Dir(issuegraphDir)
	if !isDetachedCommitWorktreePath(repoRoot) {
		return ""
	}

	branch, err := gitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch != "HEAD" {
		return ""
	}

	head, err := gitOutput(repoRoot, "rev-parse", "HEAD")
	if err != nil || head == "" {
		return ""
	}

	worktrees, err := listWorktrees(repoRoot)
	if err != nil {
		return ""
	}

	var candidates []worktreeInfo
	for _, wt := range worktrees {
		if wt.Bare || wt.Detached || wt.Branch == "" {
			continue
		}
		if wt.Head != head || utils.PathsEqual(wt.Path, repoRoot) {
			continue
		}
		candidates = append(candidates, wt)
	}

	if len(candidates) == 0 {
		return ""
	}

	sort.Slice(candidates, func(i, j int) bool {
		iStable := !isDetachedCommitWorktreePath(candidates[i].Path)
		jStable := !isDetachedCommitWorktreePath(candidates[j].Path)
		if iStable != jStable {
			return iStable
		}
		return candidates[i].Path < candidates[j].Path
	})

	stableIssueGraphDir := filepath.Join(candidates[0].Path, ".issuegraph")
	if info, err := os.Stat(stableIssueGraphDir); err == nil && info.IsDir() {
		return utils.CanonicalizePath(stableIssueGraphDir)
	}

	return ""
}

// isDetachedCommitWorktreePath checks if a path follows the megarepo convention
// of placing detached worktrees under refs/commits/<sha>.
func isDetachedCommitWorktreePath(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/refs/commits/")
}

func gitOutput(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...) //nolint:gosec // args are internal, not user-supplied
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func listWorktrees(repoRoot string) ([]worktreeInfo, error) {
	output, err := gitOutput(repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []worktreeInfo
	var current *worktreeInfo

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &worktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		case current == nil:
			continue
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "detached":
			current.Detached = true
		case line == "bare":
			current.Bare = true
		}
	}

	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, nil
}

// RedirectInfo contains information about a issuegraph directory redirect.
type RedirectInfo struct {
	// IsRedirected is true if the local .issuegraph has a redirect file
	IsRedirected bool
	// LocalDir is the local .issuegraph directory (the one with the redirect file)
	LocalDir string
	// TargetDir is the actual .issuegraph directory being used (after following redirect)
	TargetDir string
}

// GetRedirectInfo checks if the current issuegraph directory is redirected.
// It searches for the local .issuegraph/ directory and checks if it contains a redirect file.
// Returns RedirectInfo with IsRedirected=true if a redirect is active.
//
// bd-wayc3: This function now also checks the git repo's local .issuegraph directory even when
// ISSUEGRAPH_DIR is set. This handles the case where ISSUEGRAPH_DIR is pre-set to the redirect target
// (e.g., by shell environment or tooling), but we still need to detect that a redirect exists.
func GetRedirectInfo() RedirectInfo {
	// First, always check the git repo's local .issuegraph directory for redirects
	// This handles the case where ISSUEGRAPH_DIR is pre-set to the redirect target
	if localIssueGraphDir := findLocalBdsDirInRepo(); localIssueGraphDir != "" {
		if info := checkRedirectInDir(localIssueGraphDir); info.IsRedirected {
			return info
		}
	}

	// Fall back to original logic for non-git-repo cases
	if localIssueGraphDir := findLocalIssueGraphDir(); localIssueGraphDir != "" {
		return checkRedirectInDir(localIssueGraphDir)
	}

	return RedirectInfo{}
}

// checkRedirectInDir checks if a issuegraph directory has a redirect file and returns redirect info.
// Returns RedirectInfo with IsRedirected=true if a valid redirect exists.
func checkRedirectInDir(issuegraphDir string) RedirectInfo {
	info := RedirectInfo{LocalDir: issuegraphDir}

	// Check if this directory has a redirect file
	redirectFile := filepath.Join(issuegraphDir, RedirectFileName)
	if _, err := os.Stat(redirectFile); err != nil {
		// No redirect file
		return info
	}

	// There's a redirect - find the target
	targetDir := FollowRedirect(issuegraphDir)
	if targetDir == issuegraphDir {
		// Redirect file exists but failed to resolve (invalid target)
		return info
	}

	info.IsRedirected = true
	info.TargetDir = targetDir
	return info
}

// findLocalBdsDirInRepo finds the .issuegraph directory relative to the git repo root.
// This ignores ISSUEGRAPH_DIR to find the "true local" .issuegraph for redirect detection.
// bd-wayc3: Added to detect redirects even when ISSUEGRAPH_DIR is pre-set.
func findLocalBdsDirInRepo() string {
	// Get git repo root
	repoRoot := git.GetRepoRoot()
	if repoRoot == "" {
		return ""
	}

	issuegraphDir := legacydir.WorkspaceDir(repoRoot)
	if info, err := os.Stat(issuegraphDir); err == nil && info.IsDir() {
		return issuegraphDir
	}

	return ""
}

// findLocalIssueGraphDir finds the local .issuegraph directory without following redirects.
// This is used to detect if a redirect is configured.
func findLocalIssueGraphDir() string {
	// Check ISSUEGRAPH_DIR environment variable first
	if issuegraphDir := legacydir.EnvIssueGraphDir(); issuegraphDir != "" {
		return canonicalizeIssueGraphDirPath(issuegraphDir)
	}

	// For worktrees, check worktree-local redirect first (per-worktree override).
	// Returns the raw worktree .issuegraph dir (not the resolved target) since
	// findLocalIssueGraphDir doesn't follow redirects — callers use FollowRedirect.
	if git.IsWorktree() {
		if root := git.GetRepoRoot(); root != "" {
			wt := legacydir.WorkspaceDir(root)
			// Check for redirect file first
			if _, err := os.Stat(filepath.Join(wt, "redirect")); err == nil {
				return wt
			}
			// Check for worktree's own .issuegraph with project files (separate-DB mode)
			if info, err := os.Stat(wt); err == nil && info.IsDir() {
				if hasIssueGraphProjectFiles(wt) {
					return wt
				}
			}
		}
	}

	if issuegraphDir := GetWorktreeFallbackIssueGraphDir(); issuegraphDir != "" {
		if info, err := os.Stat(issuegraphDir); err == nil && info.IsDir() {
			return issuegraphDir
		}
	}

	// Walk up directory tree
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	for dir := cwd; dir != "/" && dir != "."; {
		issuegraphDir := legacydir.WorkspaceDir(dir)
		if info, err := os.Stat(issuegraphDir); err == nil && info.IsDir() {
			return issuegraphDir
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root (works on both Unix and Windows)
			// On Unix: filepath.Dir("/") returns "/"
			// On Windows: filepath.Dir("C:\\") returns "C:\\"
			break
		}
		dir = parent
	}

	return ""
}

// findDatabaseInIssueGraphDir searches for a database within a .issuegraph directory.
// Checks metadata.json for the selected implementation's database path. For
// server mode, no local path needs to exist yet: authoritative
// metadata is enough to route the caller without falling through to Dolt.
// Embedded Dolt checks both embeddeddolt/ and the legacy dolt/ path. Returns
// empty string if no database is configured or found.
func findDatabaseInIssueGraphDir(issuegraphDir string, _ bool) string {
	// Check for metadata.json first (single source of truth)
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		// A present but unreadable/invalid metadata file is authoritative. Do
		// not bypass it by discovering a leftover local Dolt directory; callers
		// that need the detailed error should use OpenBestAvailable.
		return ""
	}
	if cfg != nil {
		if !configfile.IsSupportedBackend(cfg.Backend) {
			// Unknown and removed backend markers are not Dolt aliases. Let the
			// storage-selection layer report the metadata error without routing
			// through a leftover local Dolt directory.
			return ""
		}
		// A registered backend whose workspace is the .issuegraph directory has no
		// separately discoverable local database; metadata.json plus its remote
		// store identify the workspace. Return the .issuegraph dir itself so extension
		// callers discover it, mirroring the CLI's registered-workspace path in
		// apps/cli/main.go instead of falling through to Dolt-only discovery.
		if backends.WorkspaceIsIssueGraphDir(cfg.GetBackend()) {
			return issuegraphDir
		}
		// For Dolt server mode, database is on the server - no local directory required
		if cfg.IsDoltServerMode() {
			return cfg.DatabasePath(issuegraphDir)
		}
		// For embedded Dolt, the engine stores data under .issuegraph/embeddeddolt/,
		// not .issuegraph/dolt/. Check the actual embedded data directory first.
		embeddedPath := filepath.Join(issuegraphDir, "embeddeddolt")
		if info, err := os.Stat(embeddedPath); err == nil && info.IsDir() {
			return embeddedPath
		}
		// Fall back to configured database path (e.g. .issuegraph/dolt/ for
		// server-mode installs or legacy setups that pre-date embeddeddolt).
		doltPath := cfg.DatabasePath(issuegraphDir)
		if info, err := os.Stat(doltPath); err == nil && info.IsDir() {
			return doltPath
		}
	}

	// Fall back: check if embeddeddolt or dolt directory exists without metadata.json
	embeddedPath := filepath.Join(issuegraphDir, "embeddeddolt")
	if info, err := os.Stat(embeddedPath); err == nil && info.IsDir() {
		return embeddedPath
	}
	doltPath := filepath.Join(issuegraphDir, "dolt")
	if info, err := os.Stat(doltPath); err == nil && info.IsDir() {
		return doltPath
	}

	return ""
}

// Storage provides the minimal interface for extension orchestration
type Storage = storage.Storage

// Transaction provides atomic multi-operation support within a database transaction.
// Use Storage.RunInTransaction() to obtain a Transaction instance.
type Transaction = storage.Transaction

// FindDatabasePath discovers the issuegraph database path using issuegraph's standard search order:
//  1. $ISSUEGRAPH_DIR environment variable (points to .issuegraph directory)
//  2. $ISSUEGRAPH_DB environment variable (points directly to database file, deprecated)
//  3. .issuegraph/*.db in current directory or ancestors
//
// Redirect files are supported: if a .issuegraph/redirect file exists, its contents
// are used as the actual .issuegraph directory path.
//
// Returns empty string if no database is found.
func FindDatabasePath() string {
	// 1. Check ISSUEGRAPH_DIR environment variable (preferred)
	if issuegraphDir := legacydir.EnvIssueGraphDir(); issuegraphDir != "" {
		// Canonicalize the path to prevent nested .issuegraph directories
		absIssueGraphDir := canonicalizeIssueGraphDirPath(issuegraphDir)

		// Follow redirect if present
		absIssueGraphDir = FollowRedirect(absIssueGraphDir)

		// Use helper to find database (no warnings for ISSUEGRAPH_DIR - user explicitly set it)
		if dbPath := findDatabaseInIssueGraphDir(absIssueGraphDir, false); dbPath != "" {
			return dbPath
		}

		// ISSUEGRAPH_DIR is set but no database found - this is OK for --no-db mode
		// Return empty string and let the caller handle it
		return ""
	}

	// 2. Check ISSUEGRAPH_DB environment variable (deprecated but still supported)
	if envDB := os.Getenv("ISSUEGRAPH_DB"); envDB != "" {
		absDB := utils.CanonicalizePath(envDB)
		// If ISSUEGRAPH_DB points to a directory rather than a file, treat it
		// like ISSUEGRAPH_DIR to avoid filepath.Dir() resolving one level too
		// high in the caller (apps/cli/main.go). See GH#2548.
		if info, err := os.Stat(absDB); err == nil && info.IsDir() {
			if dbPath := findDatabaseInIssueGraphDir(absDB, false); dbPath != "" {
				return dbPath
			}
		}
		return absDB
	}

	// 3. Search for .issuegraph/*.db in current directory and ancestors
	if foundDB := findDatabaseInTree(); foundDB != "" {
		return utils.CanonicalizePath(foundDB)
	}

	// No fallback to ~/.issuegraph - return empty string
	return ""
}

// FindIssueGraphDirFrom finds the effective .issuegraph/ directory as if discovery
// started from startDir, without changing the process working directory.
func FindIssueGraphDirFrom(startDir string) string {
	if startDir == "" {
		return ""
	}

	info, err := os.Stat(startDir)
	if err != nil || !info.IsDir() {
		return ""
	}

	startDir = utils.CanonicalizePath(startDir)
	repoRoot := ""
	if out, err := gitOutput(startDir, "rev-parse", "--show-toplevel"); err == nil {
		repoRoot = utils.CanonicalizePath(out)
	}

	jjSecondaryRoot := ""
	jjPrimaryIssueGraphDir := ""
	jjPrimaryHasDB := false
	if root, ok := git.JJSecondaryWorkspaceRootFrom(startDir); ok {
		jjSecondaryRoot = utils.CanonicalizePath(root)
		if primaryRoot, err := git.GetJJPrimaryWorkspaceRootFrom(startDir); err == nil && primaryRoot != "" {
			primaryIssueGraphDir := legacydir.WorkspaceDir(primaryRoot)
			if info, err := os.Stat(primaryIssueGraphDir); err == nil && info.IsDir() {
				resolved := FollowRedirect(primaryIssueGraphDir)
				if hasIssueGraphProjectFiles(resolved) {
					jjPrimaryIssueGraphDir = resolved
					jjPrimaryHasDB = hasIssueGraphDatabase(resolved)
				}
			}
		}
	}

	fallbackIssueGraphDir := ""
	fallbackHasDB := false
	if repoRoot != "" {
		fallbackIssueGraphDir = worktreeFallbackIssueGraphDirForRepo(repoRoot)
		if fallbackIssueGraphDir != "" {
			if fbInfo, err := os.Stat(fallbackIssueGraphDir); err == nil && fbInfo.IsDir() {
				fallbackHasDB = hasIssueGraphDatabase(FollowRedirect(fallbackIssueGraphDir))
			}
		}
	}

	for dir := startDir; dir != "/" && dir != "."; {
		issuegraphDir := legacydir.WorkspaceDir(dir)
		if info, err := os.Stat(issuegraphDir); err == nil && info.IsDir() {
			resolved := FollowRedirect(issuegraphDir)
			isWorktreeRoot := repoRoot != "" && utils.PathsEqual(dir, repoRoot)
			isJJSecondaryRoot := jjSecondaryRoot != "" && utils.PathsEqual(dir, jjSecondaryRoot)
			if isWorktreeRoot && fallbackHasDB && !hasIssueGraphDatabase(resolved) {
				// A worktree root can contain tracked .issuegraph metadata without
				// owning the ignored database directory. Match FindIssueGraphDir by
				// preferring the shared worktree database in that case.
			} else if isJJSecondaryRoot && jjPrimaryHasDB && !hasIssueGraphDatabase(resolved) {
				// A jj secondary workspace can likewise contain inherited
				// .issuegraph metadata without the ignored database directory.
				// Match FindIssueGraphDir by preferring the primary workspace DB.
			} else if hasIssueGraphProjectFiles(resolved) {
				return resolved
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if fallbackIssueGraphDir != "" {
		if info, err := os.Stat(fallbackIssueGraphDir); err == nil && info.IsDir() {
			resolved := FollowRedirect(fallbackIssueGraphDir)
			if hasIssueGraphProjectFiles(resolved) {
				return resolved
			}
		}
	}

	if jjPrimaryIssueGraphDir != "" {
		return jjPrimaryIssueGraphDir
	}

	return ""
}

// hasIssueGraphProjectFiles checks if a .issuegraph directory contains actual project files.
// Returns true if the directory contains any of:
// - metadata.json or config.yaml (project configuration)
// - Any *.db file (excluding backups and vc.db)
// - A dolt/ directory (Dolt database)
//
// Returns false for directories that only contain legacy registry files.
// This prevents FindIssueGraphDir from returning ~/.issuegraph/ which only has registry.json.
func hasIssueGraphProjectFiles(issuegraphDir string) bool {
	// Check for project configuration files
	if _, err := os.Stat(filepath.Join(issuegraphDir, "metadata.json")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(issuegraphDir, "config.yaml")); err == nil {
		return true
	}

	// Check for Dolt database directory (server mode uses dolt/, embedded uses embeddeddolt/)
	if info, err := os.Stat(filepath.Join(issuegraphDir, "dolt")); err == nil && info.IsDir() {
		return true
	}
	if info, err := os.Stat(filepath.Join(issuegraphDir, "embeddeddolt")); err == nil && info.IsDir() {
		return true
	}

	// Check for database files (excluding backups and vc.db)
	dbMatches, _ := filepath.Glob(filepath.Join(issuegraphDir, "*.db"))
	for _, match := range dbMatches {
		baseName := filepath.Base(match)
		if !strings.Contains(baseName, ".backup") && baseName != "vc.db" {
			return true
		}
	}

	return false
}

// hasIssueGraphDatabase is the strict counterpart to hasIssueGraphProjectFiles: it
// returns true only when issuegraphDir contains an actual database — a dolt/
// directory, an embeddeddolt/ directory, or a non-backup *.db file. Mere
// presence of metadata.json / config.yaml / issues.jsonl does not count.
//
// Used by FindIssueGraphDir's worktree-separate-DB branch to distinguish a
// genuine separate-database worktree (which owns its own Dolt data) from
// a worktree that has inherited tracked .issuegraph/ artifacts through a git
// checkout of the parent repo's working-tree snapshot. Without this strict
// check, the separate-DB branch would match on inherited metadata.json and
// return a broken directory, short-circuiting the shared-DB fallback.
func hasIssueGraphDatabase(issuegraphDir string) bool {
	if info, err := os.Stat(filepath.Join(issuegraphDir, "dolt")); err == nil && info.IsDir() {
		return true
	}
	if info, err := os.Stat(filepath.Join(issuegraphDir, "embeddeddolt")); err == nil && info.IsDir() {
		return true
	}
	dbMatches, _ := filepath.Glob(filepath.Join(issuegraphDir, "*.db"))
	for _, match := range dbMatches {
		baseName := filepath.Base(match)
		if !strings.Contains(baseName, ".backup") && baseName != "vc.db" {
			return true
		}
	}
	return false
}

// FindIssueGraphDir finds the .issuegraph/ directory in the current directory tree.
// Returns empty string if not found.
//
// Resolution order:
//  1. ISSUEGRAPH_DIR environment variable (highest priority)
//  2. Walk up from CWD toward repo root boundary, checking each directory
//     for .issuegraph/ with valid project files. For worktrees, stops at the
//     worktree root; for non-worktrees, stops at the git root.
//  3. Worktree-specific fallback: per-worktree redirect, worktree's own
//     .issuegraph (separate-DB mode), shared .issuegraph via git-common-dir.
//  4. Extended walk from the boundary to the main repo root (worktrees)
//     or checks the git root itself (non-worktrees).
//
// Validates that directories contain actual project files (metadata.json,
// config.yaml, dolt/, embeddeddolt/, or *.db).
// Redirect files are supported: if a .issuegraph/redirect file exists, its
// contents are used as the actual .issuegraph directory path.
func FindIssueGraphDir() string {
	// 1. Check ISSUEGRAPH_DIR environment variable (preferred)
	if issuegraphDir := legacydir.EnvIssueGraphDir(); issuegraphDir != "" {
		absIssueGraphDir := canonicalizeIssueGraphDirPath(issuegraphDir)

		// Follow redirect if present
		absIssueGraphDir = FollowRedirect(absIssueGraphDir)

		if info, err := os.Stat(absIssueGraphDir); err == nil && info.IsDir() {
			// Validate directory contains actual project files
			if hasIssueGraphProjectFiles(absIssueGraphDir) {
				return absIssueGraphDir
			}
		}
	}

	// 2. Walk up from CWD toward the repo root, checking each directory for .issuegraph/.
	// This replaces the former step 1b (CWD-only check) with a proper ancestor walk,
	// fixing the case where CWD is a subdirectory within a rig (not the rig root itself).
	// For worktrees, the walk stops at the worktree root boundary to avoid finding
	// git-tracked .issuegraph/ at the worktree root that has metadata but no database.
	// The worktree-specific fallback logic (step 3) handles worktree root resolution.
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	gitRoot := findGitRoot()

	// Determine the walk-up boundary: worktree root for worktrees, git root otherwise.
	// We stop BEFORE the boundary so worktree fallback logic can handle the root's .issuegraph/.
	isWt := git.IsWorktree()
	// A jj secondary workspace is inside the primary's git tree but is not a git
	// worktree. git.GetRepoRoot() returns the PRIMARY workspace root in that case,
	// so without correction the walk would cross the secondary workspace boundary and
	// find the secondary's git-tracked .issuegraph/ (which has config files but no DB).
	// Treat the jj secondary workspace root as the walk boundary, then step 3 below
	// handles the fallback to the primary's .issuegraph/ exactly like git worktrees do.
	var jjSecondaryRoot string
	var isJJSecondary bool
	if !isWt {
		jjSecondaryRoot, isJJSecondary = git.JJSecondaryWorkspaceRoot()
	}
	walkBoundary := gitRoot
	if isWt {
		// For worktrees, stop the walk at the worktree root.
		// The worktree root's .issuegraph/ may be git-tracked metadata without a real database;
		// the worktree fallback logic (step 3) handles this correctly.
		walkBoundary = git.GetRepoRoot()
	} else if isJJSecondary {
		walkBoundary = jjSecondaryRoot
	}

	// Canonicalize both walk start and walk boundary so the `dir == walkBoundary`
	// comparison below works even when the two come from different sources
	// (os.Getwd() often returns unresolved symlinks like /var/... on macOS
	// while git rev-parse returns the canonical /private/var/... form). Without
	// this, the boundary check silently never matches and the walk overshoots
	// the worktree root — finding an inherited .issuegraph/ directory there and
	// short-circuiting the worktree-fallback logic in step 3.
	cwdCanonical := utils.CanonicalizePath(cwd)
	walkBoundaryCanonical := ""
	if walkBoundary != "" {
		walkBoundaryCanonical = utils.CanonicalizePath(walkBoundary)
	}
	for dir := cwdCanonical; dir != "/" && dir != "."; {
		// Stop at the walk boundary (exclusive — don't check this directory).
		// For worktrees: stops before worktree root so step 3 handles it.
		// For non-worktrees: stops before git root (which is checked below in the
		// post-worktree walk, step 4).
		if walkBoundaryCanonical != "" && dir == walkBoundaryCanonical {
			break
		}

		issuegraphDir := legacydir.WorkspaceDir(dir)
		if info, err := os.Stat(issuegraphDir); err == nil && info.IsDir() {
			issuegraphDir = FollowRedirect(issuegraphDir)
			if hasIssueGraphProjectFiles(issuegraphDir) {
				return issuegraphDir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// 3. Worktree-specific fallback: redirect, own .issuegraph, shared .issuegraph.
	// This runs after the walk-up so that rig subdirectories win, but before
	// the extended walk (step 4) so worktree-aware logic is preferred.
	var mainRepoRoot string
	if isWt {
		// 3a. Per-worktree redirect override
		if target := worktreeRedirectTarget(); target != "" {
			if info, err := os.Stat(target); err == nil && info.IsDir() {
				if hasIssueGraphProjectFiles(target) {
					return target
				}
			}
		}

		// 3b. Worktree's own .issuegraph (separate-DB mode, no redirect).
		//
		// Only accept the worktree-local .issuegraph/ as separate-DB if it owns an
		// actual database (dolt/, embeddeddolt/, or a *.db file). A worktree
		// that only has metadata.json / config.yaml / issues.jsonl is almost
		// certainly carrying tracked artifacts from the parent repo's
		// working-tree snapshot — `git worktree add` checks them out, but the
		// dolt/ data directory is gitignored and therefore absent. Returning
		// such a directory short-circuits the shared-DB fallback (3c) and
		// causes issuegraph to spawn a sidecar Dolt server against an empty data
		// directory, which cannot serve the project's database.
		//
		// If no fallback is available (non-worktree edge case, or the main
		// repo itself has no .issuegraph/), fall back to hasIssueGraphProjectFiles so a
		// fresh `issuegraph init` can still locate the nascent project directory.
		if worktreeRoot := git.GetRepoRoot(); worktreeRoot != "" {
			worktreeIssueGraphDir := legacydir.WorkspaceDir(worktreeRoot)
			if info, err := os.Stat(worktreeIssueGraphDir); err == nil && info.IsDir() {
				if hasIssueGraphDatabase(worktreeIssueGraphDir) {
					return worktreeIssueGraphDir
				}
				// Lenient acceptance only when there is no shared .issuegraph with
				// a real database to fall back to.
				fallback := GetWorktreeFallbackIssueGraphDir()
				fallbackHasDB := false
				if fallback != "" {
					if fbInfo, err := os.Stat(fallback); err == nil && fbInfo.IsDir() {
						resolved := FollowRedirect(fallback)
						fallbackHasDB = hasIssueGraphDatabase(resolved)
					}
				}
				if !fallbackHasDB && hasIssueGraphProjectFiles(worktreeIssueGraphDir) {
					return worktreeIssueGraphDir
				}
			}
		}

		// 3c. Fall back to the canonical shared .issuegraph for this worktree.
		if fallbackIssueGraphDir := GetWorktreeFallbackIssueGraphDir(); fallbackIssueGraphDir != "" {
			if info, err := os.Stat(fallbackIssueGraphDir); err == nil && info.IsDir() {
				fallbackIssueGraphDir = FollowRedirect(fallbackIssueGraphDir)
				if hasIssueGraphProjectFiles(fallbackIssueGraphDir) {
					return fallbackIssueGraphDir
				}
			}
		}

		var err error
		mainRepoRoot, err = git.GetMainRepoRoot()
		if err != nil {
			mainRepoRoot = ""
		}
	} else if isJJSecondary {
		// 3'. JJ secondary fallback: mirror of step 3 for git worktrees
		// (no 3'a — jj has no per-workspace redirect equivalent).
		jjPrimaryRoot, jjPrimaryErr := git.GetJJPrimaryWorkspaceRoot()

		// 3'b. Only accept the secondary's own .issuegraph/ if it owns a real database.
		// Otherwise it's git-tracked config inherited from the primary; fall through.
		if jjSecondaryRoot != "" {
			secondaryIssueGraphDir := legacydir.WorkspaceDir(jjSecondaryRoot)
			if info, err := os.Stat(secondaryIssueGraphDir); err == nil && info.IsDir() {
				if hasIssueGraphDatabase(secondaryIssueGraphDir) {
					return secondaryIssueGraphDir
				}
				// Lenient acceptance only when the primary has no DB to fall back to.
				primaryFallbackHasDB := false
				if jjPrimaryErr == nil && jjPrimaryRoot != "" {
					primaryIssueGraphDir := legacydir.WorkspaceDir(jjPrimaryRoot)
					if pInfo, pErr := os.Stat(primaryIssueGraphDir); pErr == nil && pInfo.IsDir() {
						primaryFallbackHasDB = hasIssueGraphDatabase(FollowRedirect(primaryIssueGraphDir))
					}
				}
				if !primaryFallbackHasDB && hasIssueGraphProjectFiles(secondaryIssueGraphDir) {
					return secondaryIssueGraphDir
				}
			}
		}

		// 3'c.
		if jjPrimaryErr == nil && jjPrimaryRoot != "" {
			primaryIssueGraphDir := legacydir.WorkspaceDir(jjPrimaryRoot)
			if info, err := os.Stat(primaryIssueGraphDir); err == nil && info.IsDir() {
				resolved := FollowRedirect(primaryIssueGraphDir)
				if hasIssueGraphProjectFiles(resolved) {
					return resolved
				}
			}
			mainRepoRoot = jjPrimaryRoot
		}
	}

	// 4. Extended walk: from walk boundary to git/main-repo root.
	// For non-worktrees, this checks the git root itself (the walk-up in step 2
	// stopped before it). For worktrees, this walks from worktree root to main
	// repo root, handling edge cases where .issuegraph/ is between the two.
	// Skip if there was no walk boundary (step 2 already searched everything).
	if walkBoundary != "" {
		extendedRoot := gitRoot
		if (isWt || isJJSecondary) && mainRepoRoot != "" {
			extendedRoot = mainRepoRoot
		}
		// Canonicalize the extended-root so the `dir == extendedRoot` check
		// matches when extendedRoot came from a git helper (canonical) and
		// the starting `dir` came from walkBoundary (also canonicalized
		// above). Keeps the walk bounded on macOS-style /var → /private/var.
		extendedRootCanonical := ""
		if extendedRoot != "" {
			extendedRootCanonical = utils.CanonicalizePath(extendedRoot)
		}

		for dir := walkBoundaryCanonical; dir != "/" && dir != "."; {
			issuegraphDir := legacydir.WorkspaceDir(dir)
			if info, err := os.Stat(issuegraphDir); err == nil && info.IsDir() {
				issuegraphDir = FollowRedirect(issuegraphDir)
				if hasIssueGraphProjectFiles(issuegraphDir) {
					return issuegraphDir
				}
			}

			// Stop at the extended root
			if extendedRootCanonical != "" && dir == extendedRootCanonical {
				break
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return ""
}

// DatabaseInfo contains information about a discovered issuegraph database
type DatabaseInfo struct {
	Path          string // Full path to the .db file
	IssueGraphDir string // Parent .issuegraph directory
	IssueCount    int    // Number of issues (-1 if unknown)
}

// findGitRoot returns the root directory of the current git repository,
// or empty string if not in a git repository. Used to limit directory
// tree walking to within the current git repo.
//
// This function delegates to git.GetRepoRoot() which is worktree-aware
// and handles Windows path normalization.
func findGitRoot() string {
	return git.GetRepoRoot()
}

// GetWorktreeFallbackIssueGraphDir returns the canonical shared .issuegraph location for
// the current git worktree when no local redirect or worktree-local .issuegraph is present.
func GetWorktreeFallbackIssueGraphDir() string {
	if !git.IsWorktree() {
		return ""
	}

	commonDir, err := git.GetGitCommonDir()
	if err != nil || commonDir == "" {
		return ""
	}

	commonDir = utils.CanonicalizePath(commonDir)
	if filepath.Base(commonDir) == ".git" {
		return filepath.Join(filepath.Dir(commonDir), ".issuegraph")
	}

	return legacydir.WorkspaceDir(commonDir)
}

// ResolveIssueGraphDirForRepo returns the effective .issuegraph directory for a repo path.
// It prefers a local .issuegraph directory and otherwise falls back to the shared
// worktree location derived from git-common-dir.
//
// Unlike FindIssueGraphDir, this helper does not use ISSUEGRAPH_DIR and does not walk up
// from CWD. Callers that care about nested rig directories should resolve those
// before falling back to this repo-scoped helper.
func ResolveIssueGraphDirForRepo(repoPath string) string {
	repoPath = utils.CanonicalizePath(repoPath)
	localIssueGraphDir := legacydir.WorkspaceDir(repoPath)
	if info, err := os.Stat(localIssueGraphDir); err == nil && info.IsDir() {
		return FollowRedirect(localIssueGraphDir)
	}

	if fallback := worktreeFallbackIssueGraphDirForRepo(repoPath); fallback != "" {
		return FollowRedirect(fallback)
	}

	return FollowRedirect(localIssueGraphDir)
}

func worktreeFallbackIssueGraphDirForRepo(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--git-dir", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return ""
	}

	gitDir := gitPathForRepo(repoPath, strings.TrimSpace(lines[0]))
	commonDir := gitPathForRepo(repoPath, strings.TrimSpace(lines[1]))
	if gitDir == "" || commonDir == "" || utils.PathsEqual(gitDir, commonDir) {
		return ""
	}

	if filepath.Base(commonDir) == ".git" {
		return filepath.Join(filepath.Dir(commonDir), ".issuegraph")
	}

	return legacydir.WorkspaceDir(commonDir)
}

func gitPathForRepo(repoPath, path string) string {
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(repoPath, path)
	}
	return utils.CanonicalizePath(path)
}

// worktreeRedirectTarget returns the resolved redirect target for the current
// worktree's .issuegraph/redirect file, or empty string if not in a worktree or no
// redirect exists. This centralizes the per-worktree redirect override logic
// used by findLocalIssueGraphDir, FindIssueGraphDir, and findDatabaseInTree.
func worktreeRedirectTarget() string {
	if !git.IsWorktree() {
		return ""
	}
	worktreeRoot := git.GetRepoRoot()
	if worktreeRoot == "" {
		return ""
	}
	worktreeIssueGraphDir := legacydir.WorkspaceDir(worktreeRoot)
	redirectFile := filepath.Join(worktreeIssueGraphDir, "redirect")
	if _, err := os.Stat(redirectFile); err != nil {
		return ""
	}
	target := FollowRedirect(worktreeIssueGraphDir)
	if target == worktreeIssueGraphDir {
		// Redirect file exists but FollowRedirect returned the original path
		// (empty/invalid content). Return the raw .issuegraph dir so callers that
		// only need to know a redirect *exists* (findLocalIssueGraphDir) still work.
		return worktreeIssueGraphDir
	}
	return target
}

// findDatabaseInTree walks up the directory tree looking for .issuegraph/*.db
// Stops at the git repository root to avoid finding unrelated databases.
// For worktrees, searches the main repository root first, then falls back to worktree.
// Prefers config.json, falls back to issuegraph.db, and warns if multiple .db files exist.
// Redirect files are supported: if a .issuegraph/redirect file exists, its contents
// are used as the actual .issuegraph directory path.
func findDatabaseInTree() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Canonicalize the starting directory so the gitRoot boundary comparison
	// below matches. utils.CanonicalizePath resolves symlinks AND normalizes
	// case on macOS/Windows — the same form git helpers return — so the
	// eventual `dir == gitRoot` check doesn't silently overshoot on
	// /var → /private/var or case-insensitive filesystems. Matches the
	// canonicalization strategy in FindIssueGraphDir.
	dir = utils.CanonicalizePath(dir)

	isWt := git.IsWorktree()
	var jjSecondaryRoot string
	var isJJSecondary bool
	if !isWt {
		jjSecondaryRoot, isJJSecondary = git.JJSecondaryWorkspaceRoot()
	}

	// Check cwd first — a rig subdirectory with its own .issuegraph/ takes
	// priority over the git root's .issuegraph/ (same fix as FindIssueGraphDir step 1b).
	//
	// Skip the CWD check at the worktree root (git worktree) or jj secondary
	// workspace root: those directories may contain git-tracked metadata
	// (metadata.json, config.yaml) without a real database. In server mode,
	// findDatabaseInIssueGraphDir returns a path regardless of whether the data
	// directory exists, so without this skip issuegraph would open an empty DB.
	{
		skipCwdCheck := (isWt && dir == utils.CanonicalizePath(git.GetRepoRoot())) ||
			(isJJSecondary && dir == utils.CanonicalizePath(jjSecondaryRoot))
		if !skipCwdCheck {
			cwdIssueGraphDir := legacydir.WorkspaceDir(dir)
			if info, err := os.Stat(cwdIssueGraphDir); err == nil && info.IsDir() {
				cwdIssueGraphDir = FollowRedirect(cwdIssueGraphDir)
				if dbPath := findDatabaseInIssueGraphDir(cwdIssueGraphDir, true); dbPath != "" {
					return dbPath
				}
			}
		}
	}

	// Check if we're in a git worktree or jj secondary workspace.
	var mainRepoRoot string
	if isWt {
		// Per-worktree redirect override
		if target := worktreeRedirectTarget(); target != "" {
			if dbPath := findDatabaseInIssueGraphDir(target, true); dbPath != "" {
				return dbPath
			}
		}

		// Worktree's own .issuegraph (separate-DB mode, no redirect).
		// Only use it if the worktree has an actual database (dolt/, embeddeddolt/,
		// or *.db). A worktree that only has git-tracked metadata (metadata.json
		// with dolt_mode=server, config.yaml, etc.) should fall through to the
		// shared fallback below. This mirrors FindIssueGraphDir step 3b's
		// hasIssueGraphDatabase guard and prevents duplicate server spawns.
		if worktreeRoot := git.GetRepoRoot(); worktreeRoot != "" {
			worktreeIssueGraphDir := legacydir.WorkspaceDir(worktreeRoot)
			if info, err := os.Stat(worktreeIssueGraphDir); err == nil && info.IsDir() {
				if hasIssueGraphDatabase(worktreeIssueGraphDir) {
					if dbPath := findDatabaseInIssueGraphDir(worktreeIssueGraphDir, true); dbPath != "" {
						return dbPath
					}
				}
			}
		}

		// Fall back: search the canonical shared .issuegraph for this worktree.
		if fallbackIssueGraphDir := GetWorktreeFallbackIssueGraphDir(); fallbackIssueGraphDir != "" {
			if info, err := os.Stat(fallbackIssueGraphDir); err == nil && info.IsDir() {
				fallbackIssueGraphDir = FollowRedirect(fallbackIssueGraphDir)
				if dbPath := findDatabaseInIssueGraphDir(fallbackIssueGraphDir, true); dbPath != "" {
					return dbPath
				}
			}
		}
		var err error
		mainRepoRoot, err = git.GetMainRepoRoot()
		if err != nil {
			mainRepoRoot = ""
		}
		// If not found in main repo, fall back to worktree search below
	} else if isJJSecondary {
		jjPrimaryRoot, jjPrimaryErr := git.GetJJPrimaryWorkspaceRoot()

		// Separate-DB mode: only honor secondary's .issuegraph/ if it owns a real DB;
		// otherwise it's inherited git-tracked config and we want the primary.
		if jjSecondaryRoot != "" {
			secondaryIssueGraphDir := legacydir.WorkspaceDir(jjSecondaryRoot)
			if info, err := os.Stat(secondaryIssueGraphDir); err == nil && info.IsDir() {
				if hasIssueGraphDatabase(secondaryIssueGraphDir) {
					if dbPath := findDatabaseInIssueGraphDir(secondaryIssueGraphDir, true); dbPath != "" {
						return dbPath
					}
				}
			}
		}

		if jjPrimaryErr == nil && jjPrimaryRoot != "" {
			primaryIssueGraphDir := legacydir.WorkspaceDir(jjPrimaryRoot)
			if info, err := os.Stat(primaryIssueGraphDir); err == nil && info.IsDir() {
				primaryIssueGraphDir = FollowRedirect(primaryIssueGraphDir)
				if dbPath := findDatabaseInIssueGraphDir(primaryIssueGraphDir, true); dbPath != "" {
					return dbPath
				}
			}
			mainRepoRoot = jjPrimaryRoot
		}
	}

	// Find git root to limit the search
	gitRoot := findGitRoot()
	if (isWt || isJJSecondary) && mainRepoRoot != "" {
		gitRoot = mainRepoRoot
	}
	// Canonicalize the boundary so the `dir == gitRoot` comparison is robust
	// against symlink or case-form mismatches between git helpers and os.Getwd.
	gitRootCanonical := ""
	if gitRoot != "" {
		gitRootCanonical = utils.CanonicalizePath(gitRoot)
	}

	// Walk up directory tree (regular repository or worktree fallback)
	for {
		issuegraphDir := legacydir.WorkspaceDir(dir)
		if info, err := os.Stat(issuegraphDir); err == nil && info.IsDir() {
			// Follow redirect if present
			issuegraphDir = FollowRedirect(issuegraphDir)

			// Use helper to find database (with warnings for auto-discovery)
			if dbPath := findDatabaseInIssueGraphDir(issuegraphDir, true); dbPath != "" {
				return dbPath
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}

		// Stop at git root to avoid finding unrelated databases
		if gitRootCanonical != "" && dir == gitRootCanonical {
			break
		}

		dir = parent
	}

	return ""
}

// FindAllDatabases scans the directory hierarchy for the closest .issuegraph directory.
// Returns a slice with at most one DatabaseInfo - the closest database to CWD.
// Stops searching upward as soon as a .issuegraph directory is found,
// because in multi-workspace setups, nested .issuegraph directories
// are intentional and separate - parent directories are out of scope.
// Redirect files are supported: if a .issuegraph/redirect file exists, its contents
// are used as the actual .issuegraph directory path.
func FindAllDatabases() []DatabaseInfo {
	databases := []DatabaseInfo{} // Initialize to empty slice, never return nil
	seen := make(map[string]bool) // Track canonical paths to avoid duplicates

	dir, err := os.Getwd()
	if err != nil {
		return databases
	}

	// Find git root to limit the search
	gitRoot := findGitRoot()

	// Walk up directory tree
	for {
		issuegraphDir := legacydir.WorkspaceDir(dir)
		if info, err := os.Stat(issuegraphDir); err == nil && info.IsDir() {
			// Follow redirect if present
			issuegraphDir = FollowRedirect(issuegraphDir)

			// Look for database: dolt directory first, then legacy *.db files
			dbPath := ""
			doltDir := filepath.Join(issuegraphDir, "dolt")
			if dInfo, dErr := os.Stat(doltDir); dErr == nil && dInfo.IsDir() {
				dbPath = doltDir
			} else {
				// Legacy: check for *.db files (pre-migration)
				matches, err := filepath.Glob(filepath.Join(issuegraphDir, "*.db"))
				if err == nil && len(matches) > 0 {
					dbPath = matches[0]
				}
			}

			if dbPath != "" {
				// Resolve symlinks to get canonical path for deduplication
				canonicalPath := dbPath
				if resolved, err := filepath.EvalSymlinks(dbPath); err == nil {
					canonicalPath = resolved
				}

				// Skip if we've already seen this database (via symlink or other path)
				if seen[canonicalPath] {
					// Move up one directory
					parent := filepath.Dir(dir)
					if parent == dir {
						break
					}
					dir = parent
					continue
				}
				seen[canonicalPath] = true

				databases = append(databases, DatabaseInfo{
					Path:          dbPath,
					IssueGraphDir: issuegraphDir,
					IssueCount:    -1,
				})

				// Stop searching upward - the closest .issuegraph is the one to use
				break
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}

		// Stop at git root to avoid finding unrelated databases
		if gitRoot != "" && dir == gitRoot {
			break
		}

		dir = parent
	}

	return databases
}
