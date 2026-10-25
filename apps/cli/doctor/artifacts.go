package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactFinding represents a single detected artifact that may need cleanup.
type ArtifactFinding struct {
	Path        string // Absolute path to the artifact
	Type        string // "jsonl", "sqlite", "cruft-issuegraph", "redirect"
	Description string // Human-readable description
	SafeDelete  bool   // Whether this is safe to delete without data loss
}

// ArtifactReport contains all findings from an artifact scan.
type ArtifactReport struct {
	SQLiteArtifacts     []ArtifactFinding
	CruftIssueGraphDirs []ArtifactFinding
	RedirectIssues      []ArtifactFinding
	TotalCount          int
	SafeDeleteCount     int
}

// CheckClassicArtifacts scans for issuegraph classic artifacts that should be cleaned up
// after the Dolt migration. This includes stale JSONL files in dolt-native directories,
// leftover SQLite database files, cruft .issuegraph directories that should be redirect-only,
// and invalid redirect files.
//
// The scan is rooted at the given path and looks for .issuegraph/ directories recursively,
// checking each for artifacts that indicate incomplete migration cleanup.
func CheckClassicArtifacts(path string) DoctorCheck {
	report, err := ScanForArtifacts(path)
	if err != nil {
		return DoctorCheck{
			Name:     "Classic Artifacts",
			Status:   StatusWarning,
			Message:  "Artifact scan failed",
			Detail:   err.Error(),
			Category: CategoryMaintenance,
		}
	}

	if report.TotalCount == 0 {
		return DoctorCheck{
			Name:     "Classic Artifacts",
			Status:   StatusOK,
			Message:  "No classic artifacts detected",
			Category: CategoryMaintenance,
		}
	}

	// Build summary message
	var parts []string
	if len(report.SQLiteArtifacts) > 0 {
		parts = append(parts, fmt.Sprintf("%d SQLite artifact(s)", len(report.SQLiteArtifacts)))
	}
	if len(report.CruftIssueGraphDirs) > 0 {
		parts = append(parts, fmt.Sprintf("%d cruft .issuegraph dir(s)", len(report.CruftIssueGraphDirs)))
	}
	if len(report.RedirectIssues) > 0 {
		parts = append(parts, fmt.Sprintf("%d redirect issue(s)", len(report.RedirectIssues)))
	}

	msg := strings.Join(parts, ", ")

	// Build detail showing examples
	var details []string
	for _, findings := range [][]ArtifactFinding{
		report.SQLiteArtifacts,
		report.CruftIssueGraphDirs, report.RedirectIssues,
	} {
		for i, f := range findings {
			if i >= 3 {
				details = append(details, fmt.Sprintf("  ... and %d more %s artifact(s)", len(findings)-3, f.Type))
				break
			}
			details = append(details, fmt.Sprintf("  %s: %s", f.Path, f.Description))
		}
	}

	return DoctorCheck{
		Name:     "Classic Artifacts",
		Status:   StatusWarning,
		Message:  msg,
		Detail:   strings.Join(details, "\n"),
		Fix:      "Run 'issuegraph doctor --fix' to clean up, or 'issuegraph doctor --check=artifacts' for details",
		Category: CategoryMaintenance,
	}
}

// ScanForArtifacts performs a recursive scan of the given path for classic issuegraph artifacts.
// Returns an error if the walk itself fails (e.g., root path doesn't exist or is inaccessible).
// Individual unreadable subdirectories are skipped without error.
func ScanForArtifacts(rootPath string) (ArtifactReport, error) {
	var report ArtifactReport

	// Walk the directory tree looking for .issuegraph/ directories
	walkErr := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}

		base := filepath.Base(path)
		if base == ".git" && info.IsDir() {
			scanGitWorktreeArtifacts(path, &report)
			return filepath.SkipDir
		}

		// Skip node_modules and similar
		if info.IsDir() && (base == "node_modules" || base == "vendor" || base == "__pycache__") {
			return filepath.SkipDir
		}

		// We only care about directories named ".issuegraph"
		if !info.IsDir() || base != ".issuegraph" {
			return nil
		}

		// Found a .issuegraph directory - scan it
		scanIssueGraphDir(path, &report)

		// Don't descend into .issuegraph/ itself (we've scanned it)
		return filepath.SkipDir
	})
	if walkErr != nil {
		return report, fmt.Errorf("scanning artifacts at %s: %w", rootPath, walkErr)
	}

	report.TotalCount = len(report.SQLiteArtifacts) +
		len(report.CruftIssueGraphDirs) + len(report.RedirectIssues)

	for _, findings := range [][]ArtifactFinding{
		report.SQLiteArtifacts,
		report.CruftIssueGraphDirs, report.RedirectIssues,
	} {
		for _, f := range findings {
			if f.SafeDelete {
				report.SafeDeleteCount++
			}
		}
	}

	return report, nil
}

// scanGitWorktreeArtifacts scans the git-managed worktree area only.
// This avoids traversing the entire .git directory tree, which can be large and
// is unrelated to classic issuegraph artifact cleanup.
func scanGitWorktreeArtifacts(gitDir string, report *ArtifactReport) {
	worktreesDir := filepath.Join(gitDir, "issuegraph-worktrees")
	info, err := os.Stat(worktreesDir)
	if err != nil || !info.IsDir() {
		return
	}

	_ = filepath.Walk(worktreesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() || filepath.Base(path) != ".issuegraph" {
			return nil
		}
		scanIssueGraphDir(path, report)
		return filepath.SkipDir
	})
}

// scanIssueGraphDir checks a single .issuegraph directory for artifacts.
func scanIssueGraphDir(issuegraphDir string, report *ArtifactReport) {
	// Check if this should be a redirect-only directory
	isRedirectExpected := isRedirectExpectedDir(issuegraphDir)

	// Check if it has a redirect file
	hasRedirect := hasRedirectFile(issuegraphDir)

	// 1. Check for SQLite artifacts
	scanSQLiteArtifacts(issuegraphDir, report)

	// 2. Check for cruft .issuegraph directories (should be redirect-only)
	if isRedirectExpected {
		scanCruftIssueGraphDir(issuegraphDir, report)
	}

	// 3. Validate redirect files
	if hasRedirect {
		validateRedirect(issuegraphDir, report)
	}
}

// isRedirectExpectedDir returns true if this .issuegraph directory should contain
// only a redirect file (i.e., it's in a worktree or orchestrator subdirectory).
// NOTE: The polecats/crew/refinery patterns are retained for backwards compatibility
// with existing orchestrator installations.
func isRedirectExpectedDir(issuegraphDir string) bool {
	// The parent of .issuegraph is the project dir
	// We need to determine if this is a "leaf" .issuegraph that should redirect
	// to a "canonical" .issuegraph (typically in the main rig or main worktree)

	parent := filepath.Dir(issuegraphDir)
	parentName := filepath.Base(parent)
	grandparent := filepath.Dir(parent)
	grandparentName := filepath.Base(grandparent)

	// Pattern: */polecats/*/.issuegraph/ (orchestrator worker worktree — backwards compat)
	if grandparentName == "polecats" {
		return true
	}

	// Pattern: */crew/*/.issuegraph/ (orchestrator assistant workspace — backwards compat)
	if grandparentName == "crew" {
		return true
	}

	// Pattern: */refinery/rig/.issuegraph/ (orchestrator processor — backwards compat)
	if parentName == "rig" && grandparentName == "refinery" {
		return true
	}

	// Pattern: .git/issuegraph-worktrees/*/.issuegraph/
	if grandparentName == "issuegraph-worktrees" {
		return true
	}

	// Check if this is a rig-root .issuegraph/ (e.g., my-project/.issuegraph/)
	// that should redirect to mayor/rig/.issuegraph/
	// A rig-root .issuegraph has a sibling "mayor/" or "polecats/" directory
	if hasSibling(parent, "mayor") || hasSibling(parent, "polecats") {
		// This looks like a rig root, check if there's a canonical location
		canonicalDir := filepath.Join(parent, "mayor", "rig", ".issuegraph")
		if _, err := os.Stat(canonicalDir); err == nil {
			return true
		}
	}

	return false
}

// hasSibling returns true if the directory has a sibling with the given name.
func hasSibling(dir string, siblingName string) bool {
	sibling := filepath.Join(dir, siblingName)
	info, err := os.Stat(sibling)
	return err == nil && info.IsDir()
}

// hasRedirectFile returns true if the .issuegraph directory has a redirect file.
func hasRedirectFile(issuegraphDir string) bool {
	_, err := os.Stat(filepath.Join(issuegraphDir, "redirect"))
	return err == nil
}

// scanSQLiteArtifacts checks for leftover SQLite database files.
// Only flags SQLite files as artifacts if Dolt is the active backend.
// If SQLite is still the active backend, issuegraph.db is the live database.
func scanSQLiteArtifacts(issuegraphDir string, report *ArtifactReport) {
	if !IsDoltBackend(issuegraphDir) {
		return
	}

	entries, err := os.ReadDir(issuegraphDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// issuegraph.db and its WAL/SHM files
		if name == "issuegraph.db" || name == "issuegraph.db-shm" || name == "issuegraph.db-wal" {
			report.SQLiteArtifacts = append(report.SQLiteArtifacts, ArtifactFinding{
				Path:        filepath.Join(issuegraphDir, name),
				Type:        "sqlite",
				Description: "SQLite database file (Dolt is active backend)",
				SafeDelete:  name == "issuegraph.db-shm" || name == "issuegraph.db-wal",
			})
			continue
		}

		// issuegraph.backup-*.db pre-migration backups
		if strings.HasPrefix(name, "issuegraph.backup-") && strings.HasSuffix(name, ".db") {
			report.SQLiteArtifacts = append(report.SQLiteArtifacts, ArtifactFinding{
				Path:        filepath.Join(issuegraphDir, name),
				Type:        "sqlite",
				Description: "pre-migration backup",
				SafeDelete:  true,
			})
		}
	}
}

// scanCruftIssueGraphDir checks if a .issuegraph directory that should be redirect-only
// contains extra files beyond the redirect file.
func scanCruftIssueGraphDir(issuegraphDir string, report *ArtifactReport) {
	entries, err := os.ReadDir(issuegraphDir)
	if err != nil {
		return
	}

	// Count non-redirect entries
	var extraFiles []string
	for _, entry := range entries {
		name := entry.Name()
		// redirect file is expected
		if name == "redirect" {
			continue
		}
		// .gitkeep is harmless
		if name == ".gitkeep" {
			continue
		}
		extraFiles = append(extraFiles, name)
	}

	if len(extraFiles) == 0 {
		return
	}

	desc := fmt.Sprintf("should be redirect-only but contains: %s", strings.Join(extraFiles, ", "))
	if len(extraFiles) > 5 {
		desc = fmt.Sprintf("should be redirect-only but contains %d extra files", len(extraFiles))
	}

	report.CruftIssueGraphDirs = append(report.CruftIssueGraphDirs, ArtifactFinding{
		Path:        issuegraphDir,
		Type:        "cruft-issuegraph",
		Description: desc,
		SafeDelete:  true, // Safe: location is redirect-expected, extra files are cruft
	})
}

// validateRedirect checks that a redirect file points to a valid target.
func validateRedirect(issuegraphDir string, report *ArtifactReport) {
	redirectPath := filepath.Join(issuegraphDir, "redirect")
	data, err := os.ReadFile(redirectPath) // #nosec G304 - path constructed from walked dir
	if err != nil {
		report.RedirectIssues = append(report.RedirectIssues, ArtifactFinding{
			Path:        redirectPath,
			Type:        "redirect",
			Description: "redirect file unreadable",
			SafeDelete:  false,
		})
		return
	}

	target := strings.TrimSpace(string(data))

	// Skip comments, find first path line
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
		report.RedirectIssues = append(report.RedirectIssues, ArtifactFinding{
			Path:        redirectPath,
			Type:        "redirect",
			Description: "redirect file is empty",
			SafeDelete:  false,
		})
		return
	}

	// Resolve relative paths
	resolvedTarget := target
	if !filepath.IsAbs(target) {
		projectRoot := filepath.Dir(issuegraphDir)
		resolvedTarget = filepath.Join(projectRoot, target)
	}

	// Check target exists
	info, err := os.Stat(resolvedTarget)
	if err != nil {
		report.RedirectIssues = append(report.RedirectIssues, ArtifactFinding{
			Path:        redirectPath,
			Type:        "redirect",
			Description: fmt.Sprintf("redirect target does not exist: %s", target),
			SafeDelete:  false,
		})
		return
	}

	if !info.IsDir() {
		report.RedirectIssues = append(report.RedirectIssues, ArtifactFinding{
			Path:        redirectPath,
			Type:        "redirect",
			Description: fmt.Sprintf("redirect target is not a directory: %s", target),
			SafeDelete:  false,
		})
		return
	}

	// borg0ai/dx-issuegraph#4692: FollowRedirect ignores a redirect whose target
	// has no metadata.json and no recognizable database (a stray
	// worktree-depth redirect, e.g. a relic of the "issuegraph worktree create"
	// write-site removed in #3051, can point past the real .issuegraph dir into
	// an empty location). Flag that here too as an actionable warning, not
	// SafeDelete cruft -- the fix is to correct or remove the redirect file,
	// not to delete anything automatically.
	if !hasDatabaseOrMetadata(resolvedTarget) {
		report.RedirectIssues = append(report.RedirectIssues, ArtifactFinding{
			Path:        redirectPath,
			Type:        "redirect",
			Description: fmt.Sprintf("redirect target has no database or metadata.json (ignored by issuegraph): %s", target),
			SafeDelete:  false,
		})
	}
}

// hasDatabaseOrMetadata reports whether dir contains a metadata.json or a
// recognizable Dolt/SQLite database. Mirrors the target-validity check
// FollowRedirect (internal/issuegraph) applies before honoring a redirect;
// duplicated locally (rather than importing internal/issuegraph) to keep this
// package's existing pattern of small, self-contained filesystem checks
// (see isRedirectExpectedDir/hasRedirectFile above).
func hasDatabaseOrMetadata(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "metadata.json")); err == nil {
		return true
	}
	if info, err := os.Stat(filepath.Join(dir, "dolt")); err == nil && info.IsDir() {
		return true
	}
	if info, err := os.Stat(filepath.Join(dir, "embeddeddolt")); err == nil && info.IsDir() {
		return true
	}
	dbMatches, _ := filepath.Glob(filepath.Join(dir, "*.db"))
	for _, match := range dbMatches {
		baseName := filepath.Base(match)
		if !strings.Contains(baseName, ".backup") && baseName != "vc.db" {
			return true
		}
	}
	return false
}
