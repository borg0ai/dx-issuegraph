package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/git"
	"github.com/borg0ai/dx-issuegraph/internal/types"
	"github.com/borg0ai/dx-issuegraph/apps/cli/doctor/fix"
)

// gitCmdTimeout is the timeout for git subprocess commands in doctor checks.
// Prevents doctor checks from blocking indefinitely if git hangs.
const gitCmdTimeout = 30 * time.Second

const (
	hooksExamplesURL = "https://github.com/borg0ai/dx-issuegraph/tree/main/examples/git-hooks"
	hooksUpgradeURL  = "https://github.com/borg0ai/dx-issuegraph/issues/615"
)

// issuegraphShimMarker identifies issuegraph shim hooks (GH#946)
const issuegraphShimMarker = "# bd-shim"

// issuegraphInlineHookMarker identifies inline hooks created by issuegraph init (GH#1120)
// These hooks have the logic embedded directly rather than calling issuegraph hooks run
const issuegraphInlineHookMarker = "# issuegraph (issuegraph)"

// issuegraphSectionMarkerPrefix identifies marker-managed hooks (GH#1380)
// These use "# --- BEGIN ISSUEGRAPH INTEGRATION vX.Y.Z ---" section markers.
const issuegraphSectionMarkerPrefix = "# --- BEGIN ISSUEGRAPH INTEGRATION"

// issuegraphHooksRunPattern matches hooks that call issuegraph hooks run
var issuegraphHooksRunPattern = regexp.MustCompile(`\bbd\s+hooks\s+run\b`)

// CheckGitHooks verifies that recommended git hooks are installed.
func CheckGitHooks(cliVersion string) DoctorCheck {
	// Check if we're in a git repository using worktree-aware detection
	hooksDir, err := git.GetGitHooksDir()
	if err != nil {
		return DoctorCheck{
			Name:    "Git Hooks",
			Status:  StatusOK,
			Message: "N/A (not a git repository)",
		}
	}

	// Recommended hooks and their purposes
	recommendedHooks := map[string]string{
		"pre-commit": "Syncs pending issuegraph changes before commit",
		"post-merge": "Syncs database after git pull/merge",
		"pre-push":   "Validates database state before push",
	}
	var missingHooks []string
	var installedHooks []string

	for hookName := range recommendedHooks {
		hookPath := filepath.Join(hooksDir, hookName)
		if _, err := os.Stat(hookPath); os.IsNotExist(err) {
			missingHooks = append(missingHooks, hookName)
		} else {
			installedHooks = append(installedHooks, hookName)
		}
	}

	// Get repo root for external manager detection
	repoRoot := git.GetRepoRoot()

	// Check for external hook managers (lefthook, husky, etc.)
	externalManagers := fix.DetectExternalHookManagers(repoRoot)
	if len(externalManagers) > 0 {
		// First, check if issuegraph shims are installed (GH#946)
		// If the actual hooks are issuegraph shims, they're calling issuegraph regardless of what
		// the external manager config says (user may have leftover config files)
		if hasBdShims, issuegraphHooks := areBdShimsInstalled(hooksDir); hasBdShims {
			if outdated, oldest := findOutdatedIssueGraphHookVersions(hooksDir, issuegraphHooks, cliVersion); len(outdated) > 0 {
				return DoctorCheck{
					Name:    "Git Hooks",
					Status:  StatusWarning,
					Message: "Installed issuegraph hooks are outdated",
					Detail: fmt.Sprintf(
						"Outdated: %s (oldest: %s, current: %s)",
						strings.Join(outdated, ", "),
						oldest,
						cliVersion,
					),
					Fix: "Run 'issuegraph hooks install --force' to update hooks",
				}
			}
			return DoctorCheck{
				Name:    "Git Hooks",
				Status:  StatusOK,
				Message: "issuegraph shims installed (ignoring external manager config)",
				Detail:  fmt.Sprintf("issuegraph hooks run: %s", strings.Join(issuegraphHooks, ", ")),
			}
		}

		// External manager detected - check if it's configured to call issuegraph
		integration := fix.CheckExternalHookManagerIntegration(repoRoot)
		if integration != nil {
			// Detection-only managers - we can't verify their config
			if integration.DetectionOnly {
				return DoctorCheck{
					Name:    "Git Hooks",
					Status:  StatusOK,
					Message: fmt.Sprintf("%s detected (cannot verify issuegraph integration)", integration.Manager),
					Detail:  "Ensure your hook config calls 'issuegraph hooks run <hook>'",
				}
			}

			if integration.Configured {
				// Check if any hooks are missing issuegraph integration
				if len(integration.HooksWithoutBd) > 0 {
					return DoctorCheck{
						Name:    "Git Hooks",
						Status:  StatusWarning,
						Message: fmt.Sprintf("%s hooks not calling issuegraph", integration.Manager),
						Detail:  fmt.Sprintf("Missing issuegraph: %s", strings.Join(integration.HooksWithoutBd, ", ")),
						Fix:     "Add or upgrade to 'issuegraph hooks run <hook>'. See " + hooksUpgradeURL,
					}
				}

				// All hooks calling issuegraph - success
				return DoctorCheck{
					Name:    "Git Hooks",
					Status:  StatusOK,
					Message: fmt.Sprintf("All hooks via %s", integration.Manager),
					Detail:  fmt.Sprintf("issuegraph hooks run: %s", strings.Join(integration.HooksWithBd, ", ")),
				}
			}

			// External manager exists but doesn't call issuegraph at all
			return DoctorCheck{
				Name:    "Git Hooks",
				Status:  StatusWarning,
				Message: fmt.Sprintf("%s not calling issuegraph", fix.ManagerNames(externalManagers)),
				Detail:  "Configure hooks to call issuegraph commands",
				Fix:     "Add or upgrade to 'issuegraph hooks run <hook>'. See " + hooksUpgradeURL,
			}
		}
	}

	if len(missingHooks) == 0 {
		if outdated, oldest := findOutdatedIssueGraphHookVersions(hooksDir, installedHooks, cliVersion); len(outdated) > 0 {
			return DoctorCheck{
				Name:    "Git Hooks",
				Status:  StatusWarning,
				Message: "Installed issuegraph hooks are outdated",
				Detail: fmt.Sprintf(
					"Outdated: %s (oldest: %s, current: %s)",
					strings.Join(outdated, ", "),
					oldest,
					cliVersion,
				),
				Fix: "Run 'issuegraph hooks install --force' to update hooks",
			}
		}
		return DoctorCheck{
			Name:    "Git Hooks",
			Status:  StatusOK,
			Message: "All recommended hooks installed",
			Detail:  fmt.Sprintf("Installed: %s", strings.Join(installedHooks, ", ")),
		}
	}

	hookInstallMsg := "Install hooks with 'issuegraph hooks install'. See " + hooksExamplesURL

	if len(installedHooks) > 0 {
		return DoctorCheck{
			Name:    "Git Hooks",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Missing %d recommended hook(s)", len(missingHooks)),
			Detail:  fmt.Sprintf("Missing: %s", strings.Join(missingHooks, ", ")),
			Fix:     hookInstallMsg,
		}
	}

	return DoctorCheck{
		Name:    "Git Hooks",
		Status:  StatusWarning,
		Message: "No recommended git hooks installed",
		Detail:  fmt.Sprintf("Recommended: %s", strings.Join([]string{"pre-commit", "post-merge", "pre-push"}, ", ")),
		Fix:     hookInstallMsg,
	}
}

func findOutdatedIssueGraphHookVersions(
	hooksDir string,
	hookNames []string,
	cliVersion string,
) ([]string, string) {
	if !IsValidSemver(cliVersion) {
		return nil, ""
	}
	var outdated []string
	var oldest string
	for _, hookName := range hookNames {
		hookPath := filepath.Join(hooksDir, hookName)
		content, err := os.ReadFile(hookPath)
		if err != nil {
			continue
		}
		contentStr := string(content)
		hookVersion, ok := parseIssueGraphHookVersion(contentStr)
		if !ok || !IsValidSemver(hookVersion) {
			// No version comment found. If this is a issuegraph hook (has shim marker,
			// inline marker, or calls issuegraph hooks run), treat it as outdated since
			// all current hook templates include a version comment. (GH#1466)
			if isBdHookContent(contentStr) {
				outdated = append(outdated, fmt.Sprintf("%s@unknown", hookName))
				if oldest == "" {
					oldest = "0.0.0"
				}
			}
			continue
		}
		if CompareVersions(hookVersion, cliVersion) < 0 {
			outdated = append(outdated, fmt.Sprintf("%s@%s", hookName, hookVersion))
			if oldest == "" || CompareVersions(hookVersion, oldest) < 0 {
				oldest = hookVersion
			}
		}
	}
	return outdated, oldest
}

// isBdHookContent checks if hook content is a issuegraph hook (shim, inline, section-marker, or calls issuegraph hooks run).
func isBdHookContent(content string) bool {
	return strings.Contains(content, issuegraphShimMarker) ||
		strings.Contains(content, issuegraphInlineHookMarker) ||
		strings.Contains(content, issuegraphSectionMarkerPrefix) ||
		issuegraphHooksRunPattern.MatchString(content)
}

func parseIssueGraphHookVersion(content string) (string, bool) {
	for _, line := range strings.Split(content, "\n") {
		// Check for section marker: "# --- BEGIN ISSUEGRAPH INTEGRATION v0.57.0 ---"
		// This is the current hook format used by marker-managed installs (GH#1380).
		if strings.HasPrefix(line, issuegraphSectionMarkerPrefix) {
			after := strings.TrimPrefix(line, issuegraphSectionMarkerPrefix)
			after = strings.TrimSpace(after)
			after = strings.TrimPrefix(after, "v")
			after = strings.TrimSuffix(after, "---")
			version := strings.TrimSpace(after)
			if version != "" {
				return version, true
			}
		}
		// Check for legacy version comment: "# bd-hooks-version: 0.55.0"
		if strings.Contains(line, "bd-hooks-version:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			version := strings.TrimSpace(parts[1])
			if version != "" {
				return version, true
			}
		}
	}
	return "", false
}

// areBdShimsInstalled checks if the installed hooks are issuegraph shims, call issuegraph hooks run,
// or are inline issuegraph hooks created by issuegraph init.
// This helps detect when issuegraph hooks are installed directly but an external manager config exists.
// Returns (true, installedHooks) if issuegraph hooks are detected, (false, nil) otherwise.
// (GH#946, GH#1120)
func areBdShimsInstalled(hooksDir string) (bool, []string) {
	hooks := []string{"pre-commit", "post-merge", "pre-push"}
	var issuegraphHooks []string

	for _, hookName := range hooks {
		hookPath := filepath.Join(hooksDir, hookName)
		content, err := os.ReadFile(hookPath)
		if err != nil {
			continue
		}
		contentStr := string(content)
		// Check for bd-shim marker, issuegraph hooks run call, or inline issuegraph hook marker (from issuegraph init)
		if strings.Contains(contentStr, issuegraphShimMarker) ||
			strings.Contains(contentStr, issuegraphInlineHookMarker) ||
			issuegraphHooksRunPattern.MatchString(contentStr) {
			issuegraphHooks = append(issuegraphHooks, hookName)
		}
	}

	return len(issuegraphHooks) > 0, issuegraphHooks
}

// CheckGitWorkingTree checks if the git working tree is clean.
// This helps prevent leaving work stranded (AGENTS.md: keep git state clean).
func CheckGitWorkingTree(path string) DoctorCheck {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return DoctorCheck{
			Name:    "Git Working Tree",
			Status:  StatusOK,
			Message: "N/A (not a git repository)",
		}
	}

	cmd = exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return DoctorCheck{
			Name:    "Git Working Tree",
			Status:  StatusWarning,
			Message: "Unable to check git status",
			Detail:  err.Error(),
			Fix:     "Run 'git status' and commit/stash changes before syncing",
		}
	}

	status := strings.TrimSpace(string(out))
	if status == "" {
		return DoctorCheck{
			Name:    "Git Working Tree",
			Status:  StatusOK,
			Message: "Clean",
		}
	}

	// Parse raw porcelain lines preserving leading spaces for correct XY parsing.
	// strings.TrimSpace above strips the leading space from the first " D ..."
	// line, corrupting porcelain format. Use the raw output for line parsing.
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")

	// In redirect worktrees (.issuegraph/redirect exists), deleted .issuegraph/ files
	// are expected — the actual data lives at the redirect target (the rig).
	// Filter these out so they don't trigger a false warning.
	redirectPath := filepath.Join(path, ".issuegraph", "redirect")
	if _, err := os.Stat(redirectPath); err == nil {
		var filtered []string
		for _, line := range lines {
			if isExpectedRedirectChange(line) {
				continue
			}
			filtered = append(filtered, line)
		}
		if len(filtered) == 0 {
			return DoctorCheck{
				Name:    "Git Working Tree",
				Status:  StatusOK,
				Message: "Clean (redirect worktree, .issuegraph/ deletions expected)",
			}
		}
		lines = filtered
	}

	// Show a small sample of paths for quick debugging.
	maxLines := 8
	if len(lines) > maxLines {
		lines = append(lines[:maxLines], "…")
	}

	return DoctorCheck{
		Name:    "Git Working Tree",
		Status:  StatusWarning,
		Message: "Uncommitted changes present",
		Detail:  strings.Join(lines, "\n"),
		Fix:     "Commit or stash changes, then follow AGENTS.md: git pull --rebase && git push",
	}
}

// isExpectedRedirectChange returns true if a git status --porcelain line
// represents an expected change in a redirect worktree: deleted .issuegraph/ files
// or the untracked .issuegraph/redirect file itself.
// Porcelain format: XY PATH where X=index status, Y=worktree status.
// Deletions show as " D .issuegraph/..." (unstaged) or "D  .issuegraph/..." (staged).
func isExpectedRedirectChange(line string) bool {
	if len(line) < 4 {
		return false
	}
	xy := line[:2]
	filePath := line[3:]
	if !strings.HasPrefix(filePath, ".issuegraph/") {
		return false
	}
	// Deleted .issuegraph/ files (expected: data lives at redirect target)
	if xy == " D" || xy == "D " || xy == "DD" {
		return true
	}
	// Untracked .issuegraph/redirect file (expected: the redirect marker itself)
	if xy == "??" && filePath == ".issuegraph/redirect" {
		return true
	}
	return false
}

// CheckGitUpstream checks whether the current branch is up to date with its upstream.
// This catches common "forgot to pull/push" failure modes (AGENTS.md: pull --rebase, push).
func CheckGitUpstream(path string) DoctorCheck {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return DoctorCheck{
			Name:    "Git Upstream",
			Status:  StatusOK,
			Message: "N/A (not a git repository)",
		}
	}

	// Detect detached HEAD.
	cmd = exec.CommandContext(ctx, "git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = path
	branchOut, err := cmd.Output()
	if err != nil {
		return DoctorCheck{
			Name:    "Git Upstream",
			Status:  StatusWarning,
			Message: "Detached HEAD (no branch)",
			Fix:     "Check out a branch before syncing",
		}
	}
	branch := strings.TrimSpace(string(branchOut))

	// Check if any remotes exist — no point warning about upstream if there's no remote
	remoteCmd := exec.CommandContext(ctx, "git", "remote")
	remoteCmd.Dir = path
	remoteOut, err := remoteCmd.Output()
	if err != nil || strings.TrimSpace(string(remoteOut)) == "" {
		return DoctorCheck{
			Name:    "Git Upstream",
			Status:  StatusOK,
			Message: "N/A — no remotes configured",
		}
	}

	cmd = exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	cmd.Dir = path
	upOut, err := cmd.Output()
	if err != nil {
		return DoctorCheck{
			Name:    "Git Upstream",
			Status:  StatusWarning,
			Message: fmt.Sprintf("No upstream configured for %s", branch),
			Fix:     fmt.Sprintf("Set upstream then push: git push -u origin %s", branch),
		}
	}
	upstream := strings.TrimSpace(string(upOut))

	ahead, aheadErr := gitRevListCount(ctx, path, "@{u}..HEAD")
	behind, behindErr := gitRevListCount(ctx, path, "HEAD..@{u}")
	if aheadErr != nil || behindErr != nil {
		detailParts := []string{}
		if aheadErr != nil {
			detailParts = append(detailParts, "ahead: "+aheadErr.Error())
		}
		if behindErr != nil {
			detailParts = append(detailParts, "behind: "+behindErr.Error())
		}
		return DoctorCheck{
			Name:    "Git Upstream",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Unable to compare with upstream (%s)", upstream),
			Detail:  strings.Join(detailParts, "; "),
			Fix:     "Run 'git fetch' then check: git status -sb",
		}
	}

	if ahead == 0 && behind == 0 {
		return DoctorCheck{
			Name:    "Git Upstream",
			Status:  StatusOK,
			Message: fmt.Sprintf("Up to date (%s)", upstream),
			Detail:  fmt.Sprintf("Branch: %s", branch),
		}
	}

	if ahead > 0 && behind == 0 {
		return DoctorCheck{
			Name:    "Git Upstream",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Ahead of upstream by %d commit(s)", ahead),
			Detail:  fmt.Sprintf("Branch: %s, upstream: %s", branch, upstream),
			Fix:     "Run 'git push' (AGENTS.md: git pull --rebase && git push)",
		}
	}

	if behind > 0 && ahead == 0 {
		return DoctorCheck{
			Name:    "Git Upstream",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Behind upstream by %d commit(s)", behind),
			Detail:  fmt.Sprintf("Branch: %s, upstream: %s", branch, upstream),
			Fix:     "Run 'git pull --rebase' (then re-run issuegraph doctor)",
		}
	}

	return DoctorCheck{
		Name:    "Git Upstream",
		Status:  StatusWarning,
		Message: fmt.Sprintf("Diverged from upstream (ahead %d, behind %d)", ahead, behind),
		Detail:  fmt.Sprintf("Branch: %s, upstream: %s", branch, upstream),
		Fix:     "Run 'git pull --rebase' then 'git push'",
	}
}

func gitRevListCount(ctx context.Context, path string, rangeExpr string) (int, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", rangeExpr) // #nosec G204 -- fixed args
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	countStr := strings.TrimSpace(string(out))
	if countStr == "" {
		return 0, nil
	}

	var n int
	if _, err := fmt.Sscanf(countStr, "%d", &n); err != nil {
		return 0, err
	}
	return n, nil
}

// staleBdHookPattern matches the removed "issuegraph hook <name>" command (not "issuegraph hooks run").
// This was removed in v0.58.0 and replaced by "issuegraph hooks run".
var staleBdHookPattern = regexp.MustCompile(`\bbd\s+hook\s+(?:pre-commit|post-merge|pre-push|post-checkout|prepare-commit-msg)\b`)

// CheckStaleLegacyHooks detects *.legacy sidecar hooks (created by Python's pre-commit
// framework) that still call the removed "issuegraph hook" command. These cause "unknown command"
// errors at runtime even though "issuegraph hooks list" and "issuegraph doctor" show green. (GH#2398)
func CheckStaleLegacyHooks() DoctorCheck {
	hooksDir, err := git.GetGitHooksDir()
	if err != nil {
		return DoctorCheck{
			Name:    "Stale Legacy Hooks",
			Status:  StatusOK,
			Message: "N/A (not a git repository)",
		}
	}

	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		return DoctorCheck{
			Name:    "Stale Legacy Hooks",
			Status:  StatusOK,
			Message: "N/A (cannot read hooks directory)",
		}
	}

	var staleFiles []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".legacy") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(hooksDir, name))
		if err != nil {
			continue
		}
		if staleBdHookPattern.Match(content) {
			staleFiles = append(staleFiles, name)
		}
	}

	if len(staleFiles) == 0 {
		return DoctorCheck{
			Name:    "Stale Legacy Hooks",
			Status:  StatusOK,
			Message: "No stale legacy hook sidecars",
		}
	}

	return DoctorCheck{
		Name:    "Stale Legacy Hooks",
		Status:  StatusWarning,
		Message: fmt.Sprintf("%d stale .legacy hook(s) calling removed 'issuegraph hook' command", len(staleFiles)),
		Detail:  fmt.Sprintf("Files: %s", strings.Join(staleFiles, ", ")),
		Fix:     "Remove or update these files: rm " + strings.Join(staleFilePaths(hooksDir, staleFiles), " "),
	}
}

func staleFilePaths(hooksDir string, names []string) []string {
	paths := make([]string, len(names))
	for i, name := range names {
		paths[i] = filepath.Join(hooksDir, name)
	}
	return paths
}

// CheckHooksPath detects a dangling core.hooksPath: a git config value that
// still points at a hooks directory which no longer exists. This happens
// when a user manually deletes .issuegraph/ (e.g. `rm -rf .issuegraph/`) without
// running `issuegraph hooks uninstall` first — git keeps looking for the missing
// directory, and issuegraph' post-checkout import can recreate a stray .issuegraph/
// workspace as a side effect (GH#4440).
func CheckHooksPath() DoctorCheck {
	const name = "Hooks Path"

	repoRoot, _ := git.GetMainRepoRoot()
	if repoRoot == "" {
		repoRoot = git.GetRepoRoot()
	}
	if repoRoot == "" {
		return DoctorCheck{
			Name:    name,
			Status:  StatusOK,
			Message: "N/A (not a git repository)",
		}
	}

	hooksPath, ok := getConfiguredHooksPath(repoRoot)
	if !ok || hooksPath == "" {
		return DoctorCheck{
			Name:    name,
			Status:  StatusOK,
			Message: "core.hooksPath is not set",
		}
	}

	resolved := expandHookPathTilde(hooksPath)
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(repoRoot, resolved)
	}

	if fileExists(resolved) {
		return DoctorCheck{
			Name:    name,
			Status:  StatusOK,
			Message: fmt.Sprintf("core.hooksPath is set to %s", hooksPath),
		}
	}

	managed := IsIssueGraphManagedHooksPath(repoRoot, hooksPath)
	var fix string
	if managed {
		fix = "Run 'issuegraph doctor --fix' to unset core.hooksPath, or: git config --unset core.hooksPath"
	} else {
		fix = "core.hooksPath is not issuegraph-managed; issuegraph will not change it. Restore the missing directory, or unset it yourself: git config --unset core.hooksPath"
	}

	return DoctorCheck{
		Name:    name,
		Status:  StatusWarning,
		Message: "core.hooksPath points at a missing directory",
		Detail:  fmt.Sprintf("core.hooksPath=%s (resolved: %s)", hooksPath, resolved),
		Fix:     fix,
	}
}

// FixHooksPath unsets core.hooksPath ONLY when it is issuegraph-managed and its
// target is missing. It re-reads the current value and re-checks both
// conditions immediately before acting, so it never mutates a third-party
// hooksPath (e.g. husky) even if the check that triggered the fix pass is
// stale.
func FixHooksPath() error {
	repoRoot, _ := git.GetMainRepoRoot()
	if repoRoot == "" {
		repoRoot = git.GetRepoRoot()
	}
	if repoRoot == "" {
		return nil // not in a git repo
	}

	hooksPath, ok := getConfiguredHooksPath(repoRoot)
	if !ok || hooksPath == "" {
		return nil // nothing set
	}

	if !IsIssueGraphManagedHooksPath(repoRoot, hooksPath) {
		return nil // never touch a third-party hooksPath
	}

	resolved := expandHookPathTilde(hooksPath)
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(repoRoot, resolved)
	}
	if fileExists(resolved) {
		return nil // target exists — nothing dangling to fix
	}

	cmd := exec.Command("git", "config", "--unset", "core.hooksPath")
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config --unset core.hooksPath failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// getConfiguredHooksPath reads the raw core.hooksPath value from repoRoot.
// The bool return is false when core.hooksPath is not set at all.
func getConfiguredHooksPath(repoRoot string) (string, bool) {
	cmd := exec.Command("git", "config", "--get", "core.hooksPath")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// IsIssueGraphManagedHooksPath reports whether hooksPath is one of the values issuegraph
// itself writes to core.hooksPath (.issuegraph/hooks or .issuegraph-hooks, relative or
// absolute under repoRoot). It is the single matcher for that question;
// resetHooksPathIfIssueGraphManaged (apps/cli/hooks.go) calls it too, so the two
// paths cannot drift apart.
//
// Absolute values are compared after symlink resolution. repoRoot comes from
// git.GetMainRepoRoot, which canonicalizes the path, while the configured
// core.hooksPath is whatever string the user or an older issuegraph wrote. A raw
// string compare therefore misses a genuinely issuegraph-managed path any time the
// repo is reached through a symlink — on macOS every repo under /var (which
// is a symlink to /private/var, including all of t.TempDir()) hits this. The
// false negative is not cosmetic: CheckHooksPath then reports the dangling
// path as third-party and FixHooksPath deliberately refuses to unset it.
func IsIssueGraphManagedHooksPath(repoRoot, hooksPath string) bool {
	if hooksPath == ".issuegraph/hooks" || hooksPath == ".issuegraph-hooks" {
		return true
	}
	if !filepath.IsAbs(hooksPath) {
		return false
	}
	root := resolveExistingPrefix(repoRoot)
	candidate := resolveExistingPrefix(hooksPath)
	return candidate == filepath.Join(root, ".issuegraph", "hooks") ||
		candidate == filepath.Join(root, ".issuegraph-hooks")
}

// resolveExistingPrefix returns path with its longest existing ancestor
// replaced by that ancestor's symlink-resolved form, leaving any missing
// trailing components untouched.
//
// filepath.EvalSymlinks alone is not enough here: the whole point of the
// dangling-hooksPath check is that the leaf does not exist, so EvalSymlinks
// on the full path fails and returns nothing usable. Walking up to the
// deepest component that does exist still canonicalizes the symlinked prefix
// (/var -> /private/var), which is where the divergence lives.
func resolveExistingPrefix(path string) string {
	if path == "" || !filepath.IsAbs(path) {
		return path
	}
	current := filepath.Clean(path)
	var trailing string
	for {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			return filepath.Join(resolved, trailing)
		}
		parent := filepath.Dir(current)
		if parent == current {
			// Reached the filesystem root without finding an existing
			// component; nothing to canonicalize against.
			return filepath.Clean(path)
		}
		trailing = filepath.Join(filepath.Base(current), trailing)
		current = parent
	}
}

// CheckGitHooksDoltCompatibility checks if installed git hooks are compatible with Dolt backend.
// Hooks installed before Dolt support was added don't have the backend check and will
// fail with confusing errors on git pull/commit.
func CheckGitHooksDoltCompatibility(path string) DoctorCheck {
	backend, issuegraphDir := getBackendAndIssueGraphDir(path)

	// Only relevant for Dolt backend
	if backend != configfile.BackendDolt {
		return DoctorCheck{
			Name:    "Git Hooks Dolt Compatibility",
			Status:  StatusOK,
			Message: "N/A (not using Dolt backend)",
		}
	}

	// Check if we're in a git repository
	hooksDir, err := git.GetGitHooksDir()
	if err != nil {
		return DoctorCheck{
			Name:    "Git Hooks Dolt Compatibility",
			Status:  StatusOK,
			Message: "N/A (not a git repository)",
		}
	}

	// Check post-merge hook (most likely to cause issues with Dolt)
	postMergePath := filepath.Join(hooksDir, "post-merge")
	content, err := os.ReadFile(postMergePath)
	if err != nil {
		// No hook installed - that's fine
		return DoctorCheck{
			Name:    "Git Hooks Dolt Compatibility",
			Status:  StatusOK,
			Message: "N/A (no post-merge hook installed)",
		}
	}

	contentStr := string(content)

	// Section-marker hooks (GH#1380) delegate to 'issuegraph hooks run' which handles Dolt correctly
	if strings.Contains(contentStr, issuegraphSectionMarkerPrefix) {
		return DoctorCheck{
			Name:    "Git Hooks Dolt Compatibility",
			Status:  StatusOK,
			Message: "Marker-managed hooks (Dolt handled by issuegraph hooks run)",
		}
	}

	// Shim hooks (bd-shim) delegate to 'issuegraph hook' which handles Dolt correctly
	if strings.Contains(contentStr, issuegraphShimMarker) {
		return DoctorCheck{
			Name:    "Git Hooks Dolt Compatibility",
			Status:  StatusOK,
			Message: "Shim hooks (Dolt handled by issuegraph hook command)",
		}
	}

	// Check if it's a issuegraph inline hook
	if !strings.Contains(contentStr, issuegraphInlineHookMarker) && !strings.Contains(contentStr, "issuegraph") {
		return DoctorCheck{
			Name:    "Git Hooks Dolt Compatibility",
			Status:  StatusOK,
			Message: "N/A (not a issuegraph hook)",
		}
	}

	// Check if inline hook has the Dolt backend skip logic
	if strings.Contains(contentStr, `"backend"`) && strings.Contains(contentStr, `"dolt"`) {
		return DoctorCheck{
			Name:    "Git Hooks Dolt Compatibility",
			Status:  StatusOK,
			Message: "Inline hooks have Dolt backend check",
		}
	}

	// Hook exists but lacks Dolt check - this will cause errors
	_ = issuegraphDir // silence unused warning
	return DoctorCheck{
		Name:    "Git Hooks Dolt Compatibility",
		Status:  StatusError,
		Message: "Git hooks incompatible with Dolt backend",
		Detail:  "Installed hooks are outdated and incompatible with the Dolt backend.",
		Fix:     "Run 'issuegraph hooks install --force' to update hooks for Dolt compatibility",
	}
}

// fixGitHooks fixes missing or broken git hooks by calling issuegraph hooks install.
func fixGitHooks(path string) error {
	return fix.GitHooks(path)
}

// FindOrphanedIssues identifies issues referenced in git commits but still open in the database.
// This is the shared core logic used by both 'issuegraph orphans' and 'issuegraph doctor' commands.
// Returns empty slice if not a git repo, no issues from provider, or no orphans found (no error).
//
// Parameters:
//   - gitPath: The directory to scan for git commits
//   - provider: The issue provider to get open issues and prefix from
func FindOrphanedIssues(gitPath string, provider types.IssueProvider) ([]OrphanIssue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()

	// Skip if not in a git repo
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	cmd.Dir = gitPath
	if err := cmd.Run(); err != nil {
		return []OrphanIssue{}, nil // Not a git repo, return empty list
	}

	// Get issue prefix from provider
	issuePrefix := provider.GetIssuePrefix()

	// Get all open/in_progress issues from provider
	issues, err := provider.GetOpenIssues(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting open issues: %w", err)
	}

	openIssues := make(map[string]*OrphanIssue)
	for _, issue := range issues {
		openIssues[issue.ID] = &OrphanIssue{
			IssueID: issue.ID,
			Title:   issue.Title,
			Status:  string(issue.Status),
		}
	}

	if len(openIssues) == 0 {
		return []OrphanIssue{}, nil
	}

	// Get git log
	cmd = exec.CommandContext(ctx, "git", "log", "--oneline", "--all")
	cmd.Dir = gitPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("reading git log: %w", err)
	}

	// Parse commits for issue references
	// Match pattern like (bd-xxx) or (bd-xxx.1) including hierarchical IDs
	pattern := fmt.Sprintf(`\(%s-[a-z0-9.]+\)`, regexp.QuoteMeta(issuePrefix))
	re := regexp.MustCompile(pattern)

	var orphanedIssues []OrphanIssue
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Extract commit hash and message
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 1 {
			continue
		}

		commitHash := parts[0]
		commitMsg := ""
		if len(parts) > 1 {
			commitMsg = parts[1]
		}

		// Find issue IDs in this commit
		matches := re.FindAllString(line, -1)
		for _, match := range matches {
			issueID := strings.Trim(match, "()")
			if orphan, exists := openIssues[issueID]; exists {
				// Only record first (most recent) commit per issue
				if orphan.LatestCommit == "" {
					orphan.LatestCommit = commitHash
					orphan.LatestCommitMessage = commitMsg
				}
			}
		}
	}

	// Collect issues with commit references
	for _, orphan := range openIssues {
		if orphan.LatestCommit != "" {
			orphanedIssues = append(orphanedIssues, *orphan)
		}
	}

	return orphanedIssues, nil
}

// CheckOrphanedIssues detects issues referenced in git commits but still open.
// This catches cases where someone implemented a fix with "(bd-xxx)" in the commit
// message but forgot to run "issuegraph close".
func CheckOrphanedIssues(path string) DoctorCheck {
	// Orphaned issue detection requires a local database provider which was removed
	// during the Dolt-only migration. This check is disabled until reimplemented
	// against the Dolt store.
	return DoctorCheck{
		Name:     "Orphaned Issues",
		Status:   StatusOK,
		Message:  "N/A (not yet implemented for Dolt backend)",
		Category: CategoryGit,
	}
}
