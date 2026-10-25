package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitignoreTemplate is the canonical .issuegraph/.gitignore content
const GitignoreTemplate = `# Dolt database (managed by Dolt, not git)
dolt/
embeddeddolt/
proxieddb/

# Runtime files
issuegraph.sock
issuegraph.sock.startlock
sync-state.json
last-touched
.exclusive-lock

# Daemon runtime (lock, log, pid)
daemon.*

# Push state (runtime, per-machine)
push-state.json

# Lock files (various runtime locks)
*.lock

# Credential key (encryption key for federation peer auth — never commit)
.issuegraph-credential-key

# Local version tracking (prevents upgrade notification spam after git ops)
.local_version

proxied_server_client_info.json

# Worktree redirect file (contains relative path to main repo's .issuegraph/)
# Must not be committed as paths would be wrong in other clones
redirect

# Sync state (local-only, per-machine)
# These files are machine-specific and should not be shared across clones
.sync.lock

# Workspace operation gate (internal/workspacegate): physical-root gate
# files live beside the guarded root inside .issuegraph (e.g. dolt.gate.lock)
*.gate.lock*
export-state/
export-state.json
last_pull

# Ephemeral store (SQLite - wisps/molecules, intentionally not versioned)
ephemeral.sqlite3
ephemeral.sqlite3-journal
ephemeral.sqlite3-wal
ephemeral.sqlite3-shm

# Dolt server management (auto-started by issuegraph)
dolt-server.pid
dolt-server.log
dolt-server.lock
dolt-server.port
dolt-server.activity
dolt-server-config.yaml

# Debug-mode pprof artifacts (written when dolt.debug: true in config.yaml)
dolt-pprof/

# Corrupt backup directories (created by issuegraph doctor --fix recovery)
*.corrupt.backup/

# Backup data (auto-exported JSONL, local-only)
backup/

# Per-project environment file (Dolt connection config, GH#2520)
.env

# Legacy files (from pre-Dolt versions)
*.db
*.db?*
*.db-journal
*.db-wal
*.db-shm
db.sqlite
issuegraph.db
# NOTE: Do NOT add negation patterns here.
# They would override fork protection in .git/info/exclude.
# Config files (metadata.json, config.yaml) are tracked by git by default
# since no pattern above ignores them.
`

// ProjectGitignorePatterns are patterns that should be in the project-root .gitignore
// to prevent accidentally committing Dolt database files and credential keys.
var ProjectGitignorePatterns = []string{
	".dolt/",
	"*.db",
	".issuegraph-credential-key",
	".issuegraph/proxieddb/",
	// Workspace-gate artifacts (internal/workspacegate): the workspace
	// gate file sits BESIDE .issuegraph in the project root, so .issuegraph/
	// patterns cannot cover it.
	"*.gate.lock*",
}

// ProjectGitignoreHeader is the section header added to the project .gitignore
const ProjectGitignoreHeader = "# IssueGraph / Dolt files (added by issuegraph init)"

// requiredPatterns are patterns that MUST be in .issuegraph/.gitignore
var requiredPatterns = []string{
	"*.db?*",
	".env",
	"redirect",
	"last-touched",
	"issuegraph.sock.startlock",
	".sync.lock",
	"*.gate.lock*",
	"export-state/",
	"export-state.json",
	"last_pull",
	"dolt/",
	"embeddeddolt/",
	"proxieddb/",
	"ephemeral.sqlite3",
	"dolt-server.pid",
	"dolt-server.log",
	"dolt-server.lock",
	"dolt-server.port",
	"dolt-server.activity",
	"dolt-server-config.yaml",
	"daemon.*",
	"*.lock",
	"*.corrupt.backup/",
	".issuegraph-credential-key",
	"proxied_server_client_info.json",
	".local_version",
	"backup/",
}

// CheckGitignore checks if .issuegraph/.gitignore is up to date.
// repoPath is the project root directory.
func CheckGitignore(repoPath string) DoctorCheck {
	gitignorePath := filepath.Join(ResolveIssueGraphDirForRepo(repoPath), ".gitignore")

	// Check if file exists
	content, err := os.ReadFile(gitignorePath) // #nosec G304 -- path is constructed from known parts
	if err != nil {
		return DoctorCheck{
			Name:    "Gitignore",
			Status:  "warning",
			Message: ".issuegraph/.gitignore not found",
			Fix:     "Run: issuegraph init (safe to re-run) or issuegraph doctor --fix",
		}
	}

	// Check for required patterns
	contentStr := string(content)
	missing := missingGitignorePatterns(contentStr)
	if len(missing) > 0 {
		return DoctorCheck{
			Name:    "Gitignore",
			Status:  "warning",
			Message: "Outdated .issuegraph/.gitignore (missing required patterns)",
			Detail:  "Missing: " + strings.Join(missing, ", "),
			Fix:     "Run: issuegraph doctor --fix or issuegraph init (safe to re-run)",
		}
	}

	return DoctorCheck{
		Name:    "Gitignore",
		Status:  "ok",
		Message: "Up to date",
	}
}

// EnsureGitignoreForIssueGraphDir writes the canonical .issuegraph/.gitignore when it is
// missing or outdated. If the file does not exist, it writes the full template.
// If it exists but is outdated, it safely appends missing required patterns so
// local additions are preserved.
func EnsureGitignoreForIssueGraphDir(issuegraphDir string) error {
	gitignorePath := filepath.Join(issuegraphDir, ".gitignore")

	content, err := os.ReadFile(gitignorePath) // #nosec G304 -- caller supplies the active .issuegraph dir
	if os.IsNotExist(err) {
		return writeGitignoreTemplate(gitignorePath)
	}
	if err != nil {
		return fmt.Errorf("read .issuegraph/.gitignore: %w", err)
	}

	missing := missingGitignorePatterns(string(content))
	if len(missing) == 0 {
		return nil
	}

	if info, err := os.Stat(gitignorePath); err == nil {
		if info.Mode().Perm()&0200 == 0 {
			if err := os.Chmod(gitignorePath, 0600); err != nil {
				return fmt.Errorf("chmod .issuegraph/.gitignore: %w", err)
			}
		}
	}

	existingContent := string(content)
	newContent := existingContent
	if len(newContent) > 0 && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	newContent += "\n# Added by issuegraph (missing required patterns)\n"
	for _, pattern := range missing {
		newContent += pattern + "\n"
	}

	if err := os.WriteFile(gitignorePath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("ensure .issuegraph/.gitignore: %w", err)
	}

	// Tighten permissions on pre-existing files: os.WriteFile's mode argument
	// only applies at creation, and the file may predate the 0600 policy.
	if err := os.Chmod(gitignorePath, 0600); err != nil {
		return fmt.Errorf("chmod .issuegraph/.gitignore: %w", err)
	}

	return nil
}

// FixGitignore brings .issuegraph/.gitignore up to date: the full template when
// the file is missing, append-only for missing required patterns otherwise.
// It must never rewrite an existing file wholesale — local rules (e.g.
// keep-exports-off-master negations) live in this file too, and the old
// full-template rewrite destroyed them (bd-kaaz3).
// If a redirect exists, it writes to the redirect target's .gitignore instead.
// repoPath is the project root directory.
func FixGitignore(repoPath string) error {
	return EnsureGitignoreForIssueGraphDir(ResolveIssueGraphDirForRepo(repoPath))
}

func missingGitignorePatterns(content string) []string {
	var missing []string
	for _, pattern := range requiredPatterns {
		if !containsGitignorePattern(content, pattern) {
			missing = append(missing, pattern)
		}
	}
	return missing
}

func writeGitignoreTemplate(gitignorePath string) error {
	// If file exists and is read-only, fix permissions first
	if info, err := os.Stat(gitignorePath); err == nil {
		if info.Mode().Perm()&0200 == 0 { // No write permission for owner
			if err := os.Chmod(gitignorePath, 0600); err != nil {
				return err
			}
		}
	}

	// Write canonical template with secure file permissions
	if err := os.WriteFile(gitignorePath, []byte(GitignoreTemplate), 0600); err != nil {
		return err
	}

	// Ensure permissions are set correctly (some systems respect umask)
	if err := os.Chmod(gitignorePath, 0600); err != nil {
		return err
	}

	return nil
}

// CheckRedirectNotTracked verifies that .issuegraph/redirect is NOT tracked by git.
// Redirect files contain relative paths that only work in the original worktree.
// If committed, they cause warnings in other clones where the path is invalid.
// repoPath is the project root directory.
func CheckRedirectNotTracked(repoPath string) DoctorCheck {
	redirectPath := filepath.Join(repoPath, ".issuegraph", "redirect")

	// First check if the file exists
	if _, err := os.Stat(redirectPath); os.IsNotExist(err) {
		// File doesn't exist - nothing to check
		return DoctorCheck{
			Name:    "Redirect Tracking",
			Status:  StatusOK,
			Message: "No redirect file present",
		}
	}

	// Check if git considers this file tracked
	// git ls-files exits 0 and outputs the filename if tracked, empty if untracked
	cmd := exec.Command("git", "ls-files", redirectPath) // #nosec G204 - args are hardcoded paths
	output, err := cmd.Output()
	if err != nil {
		// Not in a git repo or git error - skip check
		return DoctorCheck{
			Name:    "Redirect Tracking",
			Status:  StatusOK,
			Message: "N/A (not a git repository)",
		}
	}

	trackedPath := strings.TrimSpace(string(output))
	if trackedPath == "" {
		// File exists but is not tracked - this is correct
		return DoctorCheck{
			Name:    "Redirect Tracking",
			Status:  StatusOK,
			Message: "redirect file not tracked (correct)",
		}
	}

	// File is tracked - this is a problem
	return DoctorCheck{
		Name:    "Redirect Tracking",
		Status:  StatusWarning,
		Message: "redirect file is tracked by git",
		Detail:  "The .issuegraph/redirect file contains a relative path that only works in this worktree. When committed, it causes warnings in other clones.",
		Fix:     "Run 'issuegraph doctor --fix' to untrack, or manually: git rm --cached .issuegraph/redirect",
	}
}

// FixRedirectTracking untracks the .issuegraph/redirect file from git.
// repoPath is the project root directory.
func FixRedirectTracking(repoPath string) error {
	redirectPath := filepath.Join(repoPath, ".issuegraph", "redirect")

	// Check if file is actually tracked first
	cmd := exec.Command("git", "ls-files", redirectPath) // #nosec G204 - args are hardcoded paths
	output, err := cmd.Output()
	if err != nil {
		return nil // Not a git repo, nothing to do
	}

	trackedPath := strings.TrimSpace(string(output))
	if trackedPath == "" {
		return nil // Not tracked, nothing to do
	}

	// Untrack the file (keeps the local copy)
	cmd = exec.Command("git", "rm", "--cached", redirectPath) // #nosec G204 - args are hardcoded paths
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to untrack redirect file: %w", err)
	}

	return nil
}

// parseRedirectTarget extracts the first non-comment, non-empty redirect target.
// It also strips a UTF-8 BOM if present.
func parseRedirectTarget(data []byte) string {
	content := strings.TrimSpace(string(data))
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "\ufeff")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}

	return ""
}

// resolveRedirectTarget resolves a redirect target relative to the .issuegraph parent.
// Absolute targets are cleaned and returned as-is.
func resolveRedirectTarget(issuegraphDir string, target string) string {
	if target == "" {
		return ""
	}

	resolvedTarget := target
	if !filepath.IsAbs(target) {
		projectRoot := filepath.Dir(issuegraphDir)
		resolvedTarget = filepath.Join(projectRoot, target)
	}
	resolvedTarget = filepath.Clean(resolvedTarget)
	if absPath, err := filepath.Abs(resolvedTarget); err == nil {
		resolvedTarget = absPath
	}

	return resolvedTarget
}

// CheckRedirectTargetValid verifies that the redirect target exists and has a valid issuegraph database.
// This catches cases where the redirect points to a non-existent directory or one without a database.
// repoPath is the project root directory.
func CheckRedirectTargetValid(repoPath string) DoctorCheck {
	redirectPath := filepath.Join(repoPath, ".issuegraph", "redirect")

	// Check if redirect file exists
	data, err := os.ReadFile(redirectPath) // #nosec G304 - path is hardcoded
	if os.IsNotExist(err) {
		return DoctorCheck{
			Name:    "Redirect Target Valid",
			Status:  StatusOK,
			Message: "No redirect configured",
		}
	}
	if err != nil {
		return DoctorCheck{
			Name:    "Redirect Target Valid",
			Status:  StatusWarning,
			Message: "Cannot read redirect file",
			Detail:  err.Error(),
		}
	}

	// Parse redirect target
	target := parseRedirectTarget(data)
	if target == "" {
		return DoctorCheck{
			Name:    "Redirect Target Valid",
			Status:  StatusWarning,
			Message: "Redirect file is empty",
			Fix:     "Remove the empty redirect file or add a valid path",
		}
	}

	issuegraphDir := filepath.Dir(redirectPath)
	resolvedTarget := resolveRedirectTarget(issuegraphDir, target)

	// Check if target directory exists
	info, err := os.Stat(resolvedTarget)
	if os.IsNotExist(err) {
		return DoctorCheck{
			Name:    "Redirect Target Valid",
			Status:  StatusError,
			Message: "Redirect target does not exist",
			Detail:  fmt.Sprintf("Target: %s", resolvedTarget),
			Fix:     "Fix the redirect path or create the target directory",
		}
	}
	if err != nil {
		return DoctorCheck{
			Name:    "Redirect Target Valid",
			Status:  StatusWarning,
			Message: "Cannot access redirect target",
			Detail:  err.Error(),
		}
	}
	if !info.IsDir() {
		return DoctorCheck{
			Name:    "Redirect Target Valid",
			Status:  StatusError,
			Message: "Redirect target is not a directory",
			Detail:  fmt.Sprintf("Target: %s", resolvedTarget),
		}
	}

	// Check for valid issuegraph database in target
	// First check for Dolt backend via metadata.json — Dolt server mode has no local .db file
	metadataPath := filepath.Join(resolvedTarget, "metadata.json")
	metadataData, metaErr := os.ReadFile(metadataPath) // #nosec G304 -- constructed from known path
	if metaErr == nil && strings.Contains(string(metadataData), `"backend"`) &&
		strings.Contains(string(metadataData), `"dolt"`) {
		return DoctorCheck{
			Name:    "Redirect Target Valid",
			Status:  StatusOK,
			Message: fmt.Sprintf("Redirect target valid (dolt backend): %s", resolvedTarget),
		}
	}

	// Legacy: check for Dolt database directory or SQLite .db file
	doltDir := filepath.Join(resolvedTarget, "dolt")
	if info, statErr := os.Stat(doltDir); statErr == nil && info.IsDir() {
		return DoctorCheck{
			Name:    "Redirect Target Valid",
			Status:  StatusOK,
			Message: fmt.Sprintf("Redirect target valid (dolt directory): %s", resolvedTarget),
		}
	}

	dbPath := filepath.Join(resolvedTarget, "issuegraph.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Also check for any .db file (legacy SQLite)
		matches, _ := filepath.Glob(filepath.Join(resolvedTarget, "*.db"))
		if len(matches) == 0 {
			return DoctorCheck{
				Name:    "Redirect Target Valid",
				Status:  StatusWarning,
				Message: "Redirect target has no issuegraph database",
				Detail:  fmt.Sprintf("Target: %s (no dolt directory or .db file found)", resolvedTarget),
				Fix:     "Run 'issuegraph init' in the target directory or check redirect path",
			}
		}
	}

	return DoctorCheck{
		Name:    "Redirect Target Valid",
		Status:  StatusOK,
		Message: fmt.Sprintf("Redirect target valid (legacy): %s", resolvedTarget),
	}
}

// CheckRedirectTargetSyncWorktree verifies that the redirect target has a working issuegraph-sync worktree.
// This is important for repos using sync-branch mode with redirects.
// repoPath is the project root directory.
func CheckRedirectTargetSyncWorktree(repoPath string) DoctorCheck {
	redirectPath := filepath.Join(repoPath, ".issuegraph", "redirect")

	// Check if redirect file exists
	data, err := os.ReadFile(redirectPath) // #nosec G304 - path is hardcoded
	if os.IsNotExist(err) {
		return DoctorCheck{
			Name:    "Redirect Target Sync",
			Status:  StatusOK,
			Message: "No redirect configured",
		}
	}
	if err != nil {
		return DoctorCheck{
			Name:    "Redirect Target Sync",
			Status:  StatusOK, // Don't warn if we can't read - other check handles that
			Message: "N/A (cannot read redirect)",
		}
	}

	target := parseRedirectTarget(data)
	if target == "" {
		return DoctorCheck{
			Name:    "Redirect Target Sync",
			Status:  StatusOK,
			Message: "N/A (empty redirect)",
		}
	}

	// Resolve the target path
	issuegraphDir := filepath.Dir(redirectPath)
	resolvedTarget := resolveRedirectTarget(issuegraphDir, target)

	// Check if the target has a sync-branch configured in config.yaml
	configPath := filepath.Join(resolvedTarget, "config.yaml")
	configData, err := os.ReadFile(configPath) // #nosec G304 - constructed from known path
	if err != nil {
		// No config.yaml means no sync-branch, which is fine
		return DoctorCheck{
			Name:    "Redirect Target Sync",
			Status:  StatusOK,
			Message: "N/A (target not using sync-branch mode)",
		}
	}

	// Simple check for sync-branch in config
	if !strings.Contains(string(configData), "sync-branch:") {
		return DoctorCheck{
			Name:    "Redirect Target Sync",
			Status:  StatusOK,
			Message: "N/A (target not using sync-branch mode)",
		}
	}

	// Target uses sync-branch - check for issuegraph-sync worktree in the repo containing the target
	// The target is inside a .issuegraph dir, so the repo is the parent of .issuegraph
	targetRepoRoot := filepath.Dir(resolvedTarget)

	// Check for issuegraph-sync worktree
	worktreePath := filepath.Join(targetRepoRoot, ".issuegraph-sync")
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return DoctorCheck{
			Name:    "Redirect Target Sync",
			Status:  StatusWarning,
			Message: "Redirect target missing issuegraph-sync worktree",
			Detail:  fmt.Sprintf("Expected worktree at: %s", worktreePath),
			Fix:     fmt.Sprintf("Run 'issuegraph init' in %s to set up issuegraph", targetRepoRoot),
		}
	}

	return DoctorCheck{
		Name:    "Redirect Target Sync",
		Status:  StatusOK,
		Message: "Redirect target has issuegraph-sync worktree",
	}
}

// CheckNoVestigialSyncWorktrees detects issuegraph-sync worktrees in redirected repos that are unused.
// When a repo uses .issuegraph/redirect, it doesn't need its own issuegraph-sync worktree since
// sync operations happen in the redirect target. These vestigial worktrees waste space.
// repoPath is the project root directory.
func CheckNoVestigialSyncWorktrees(repoPath string) DoctorCheck {
	redirectPath := filepath.Join(repoPath, ".issuegraph", "redirect")

	// Check if redirect file exists
	if _, err := os.Stat(redirectPath); os.IsNotExist(err) {
		// No redirect - this check doesn't apply
		return DoctorCheck{
			Name:    "Vestigial Sync Worktrees",
			Status:  StatusOK,
			Message: "N/A (no redirect configured)",
		}
	}

	// Use repoPath to find git root instead of walking up from cwd
	gitRoot := repoPath
	for {
		if _, err := os.Stat(filepath.Join(gitRoot, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(gitRoot)
		if parent == gitRoot {
			// Reached filesystem root, not in a git repo
			return DoctorCheck{
				Name:    "Vestigial Sync Worktrees",
				Status:  StatusOK,
				Message: "N/A (not in git repository)",
			}
		}
		gitRoot = parent
	}

	// Check for .issuegraph-sync worktree
	syncWorktreePath := filepath.Join(gitRoot, ".issuegraph-sync")
	if _, err := os.Stat(syncWorktreePath); os.IsNotExist(err) {
		// No local worktree - good
		return DoctorCheck{
			Name:    "Vestigial Sync Worktrees",
			Status:  StatusOK,
			Message: "No vestigial sync worktrees found",
		}
	}

	// Found a local .issuegraph-sync but we have a redirect - this is vestigial
	return DoctorCheck{
		Name:    "Vestigial Sync Worktrees",
		Status:  StatusWarning,
		Message: "Vestigial .issuegraph-sync worktree found",
		Detail:  fmt.Sprintf("This repo uses redirect but has unused worktree at: %s", syncWorktreePath),
		Fix:     fmt.Sprintf("Remove with: rm -rf %s", syncWorktreePath),
	}
}

// CheckLastTouchedNotTracked verifies that .issuegraph/last-touched is NOT tracked by git.
// The last-touched file is local runtime state that should never be committed.
// If committed, it causes spurious diffs in other clones.
// repoPath is the project root directory.
func CheckLastTouchedNotTracked(repoPath string) DoctorCheck {
	lastTouchedPath := filepath.Join(repoPath, ".issuegraph", "last-touched")

	// First check if the file exists
	if _, err := os.Stat(lastTouchedPath); os.IsNotExist(err) {
		// File doesn't exist - nothing to check
		return DoctorCheck{
			Name:    "Last-Touched Tracking",
			Status:  StatusOK,
			Message: "No last-touched file present",
		}
	}

	// Check if git considers this file tracked
	// git ls-files exits 0 and outputs the filename if tracked, empty if untracked
	cmd := exec.Command("git", "ls-files", lastTouchedPath) // #nosec G204 - args are hardcoded paths
	output, err := cmd.Output()
	if err != nil {
		// Not in a git repo or git error - skip check
		return DoctorCheck{
			Name:    "Last-Touched Tracking",
			Status:  StatusOK,
			Message: "N/A (not a git repository)",
		}
	}

	trackedPath := strings.TrimSpace(string(output))
	if trackedPath == "" {
		// File exists but is not tracked - this is correct
		return DoctorCheck{
			Name:    "Last-Touched Tracking",
			Status:  StatusOK,
			Message: "last-touched file not tracked (correct)",
		}
	}

	// File is tracked - this is a problem
	return DoctorCheck{
		Name:    "Last-Touched Tracking",
		Status:  StatusWarning,
		Message: "last-touched file is tracked by git",
		Detail:  "The .issuegraph/last-touched file is local runtime state that should never be committed.",
		Fix:     "Run 'issuegraph doctor --fix' to untrack, or manually: git rm --cached .issuegraph/last-touched",
	}
}

// FixLastTouchedTracking untracks the .issuegraph/last-touched file from git.
// repoPath is the project root directory.
func FixLastTouchedTracking(repoPath string) error {
	lastTouchedPath := filepath.Join(repoPath, ".issuegraph", "last-touched")

	// Check if file is actually tracked first
	cmd := exec.Command("git", "ls-files", lastTouchedPath) // #nosec G204 - args are hardcoded paths
	output, err := cmd.Output()
	if err != nil {
		return nil // Not a git repo, nothing to do
	}

	trackedPath := strings.TrimSpace(string(output))
	if trackedPath == "" {
		return nil // Not tracked, nothing to do
	}

	// Untrack the file (keeps the local copy)
	cmd = exec.Command("git", "rm", "--cached", lastTouchedPath) // #nosec G204 - args are hardcoded paths
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to untrack last-touched file: %w", err)
	}

	return nil
}

// CheckProjectGitignore checks if the project-root .gitignore contains patterns
// to prevent accidentally committing Dolt database files and credential keys.
// repoPath is the project root directory.
func CheckProjectGitignore(repoPath string) DoctorCheck {
	gitignorePath := filepath.Join(repoPath, ".gitignore")

	content, err := os.ReadFile(gitignorePath) // #nosec G304 -- path is hardcoded
	if err != nil {
		if os.IsNotExist(err) {
			return DoctorCheck{
				Name:    "Project Gitignore",
				Status:  StatusWarning,
				Message: "No project .gitignore found — Dolt/credential files may be committed accidentally",
				Fix:     "Run: issuegraph init (safe to re-run) or issuegraph doctor --fix",
			}
		}
		return DoctorCheck{
			Name:    "Project Gitignore",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Cannot read project .gitignore: %v", err),
		}
	}

	contentStr := string(content)
	var missing []string
	for _, pattern := range ProjectGitignorePatterns {
		if !containsGitignorePattern(contentStr, pattern) {
			missing = append(missing, pattern)
		}
	}

	if len(missing) > 0 {
		return DoctorCheck{
			Name:    "Project Gitignore",
			Status:  StatusWarning,
			Message: "Project .gitignore missing required exclusion patterns",
			Detail:  "Missing: " + strings.Join(missing, ", "),
			Fix:     "Run: issuegraph doctor --fix or issuegraph init (safe to re-run)",
		}
	}

	return DoctorCheck{
		Name:    "Project Gitignore",
		Status:  StatusOK,
		Message: "Dolt and credential files excluded",
	}
}

// EnsureProjectGitignore adds .dolt/, *.db, and .issuegraph-credential-key patterns
// to the project-root .gitignore if they are not already present. Creates the
// file if it doesn't exist. This prevents users from accidentally committing
// Dolt database files or the credential encryption key.
// repoPath is the project root directory.
func EnsureProjectGitignore(repoPath string) error {
	gitignorePath := filepath.Join(repoPath, ".gitignore")

	var existingContent string
	// #nosec G304 -- path is hardcoded
	if content, err := os.ReadFile(gitignorePath); err == nil {
		existingContent = string(content)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	var toAdd []string
	for _, pattern := range ProjectGitignorePatterns {
		if !containsGitignorePattern(existingContent, pattern) {
			toAdd = append(toAdd, pattern)
		}
	}

	if len(toAdd) == 0 {
		return nil // All patterns already present
	}

	newContent := existingContent
	if len(newContent) > 0 && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	newContent += "\n" + ProjectGitignoreHeader + "\n"
	for _, pattern := range toAdd {
		newContent += pattern + "\n"
	}

	// #nosec G306 -- gitignore needs to be readable by git and collaborators
	if err := os.WriteFile(gitignorePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	return nil
}

// FixProjectGitignore is an alias for EnsureProjectGitignore, used by issuegraph doctor --fix.
// repoPath is the project root directory.
func FixProjectGitignore(repoPath string) error {
	return EnsureProjectGitignore(repoPath)
}

// containsGitignorePattern checks if a gitignore file content contains the given pattern.
// It checks for the pattern as a standalone line (ignoring leading/trailing whitespace).
func containsGitignorePattern(content, pattern string) bool {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == pattern {
			return true
		}
	}
	return false
}
