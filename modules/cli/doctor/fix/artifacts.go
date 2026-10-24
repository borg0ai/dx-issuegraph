package fix

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ClassicArtifacts removes issuegraph classic artifacts found by scanning the path.
// Only removes artifacts that are safe to delete:
// - JSONL export artifacts (not issues.jsonl itself)
// - SQLite WAL/SHM files and backup databases
// - Extra files in redirect-only .issuegraph directories
func ClassicArtifacts(path string) error {
	var removed, skipped, errCount int

	// Walk the directory tree looking for .issuegraph/ directories
	err := filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}

		// Skip heavy directories
		base := filepath.Base(walkPath)
		if info.IsDir() && (base == "node_modules" || base == "vendor" || base == "__pycache__") {
			return filepath.SkipDir
		}

		// We only care about directories named ".issuegraph"
		if !info.IsDir() || base != ".issuegraph" {
			return nil
		}

		r, s, e := cleanIssueGraphDirArtifacts(walkPath)
		removed += r
		skipped += s
		errCount += e

		return filepath.SkipDir
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory tree: %w", err)
	}

	// Report summary
	fmt.Printf("  Artifact cleanup: %d removed, %d skipped, %d errors\n", removed, skipped, errCount)

	if skipped > 0 {
		fmt.Println("  Skipped items may need manual review (e.g., issues.jsonl in dolt dirs, issuegraph.db files)")
	}

	if errCount > 0 {
		return fmt.Errorf("%d artifact(s) could not be removed", errCount)
	}

	return nil
}

// cleanIssueGraphDirArtifacts cleans artifacts from a single .issuegraph directory.
// Returns counts of removed, skipped, and errored items.
func cleanIssueGraphDirArtifacts(issuegraphDir string) (removed, skipped, errCount int) {
	hasDolt := hasDoltDir(issuegraphDir)
	isRedirectExpected := isRedirectExpectedLocation(issuegraphDir)

	// 1. Clean JSONL artifacts in dolt-native directories
	if hasDolt {
		r, s, e := cleanJSONLArtifacts(issuegraphDir)
		removed += r
		skipped += s
		errCount += e
	}

	// 2. Clean SQLite artifacts
	r, s, e := cleanSQLiteArtifacts(issuegraphDir)
	removed += r
	skipped += s
	errCount += e

	// 3. Clean cruft .issuegraph directories (if redirect is expected)
	// Clean even when the redirect file is missing — stale cruft files
	// (config.yaml, metadata.json, README.md, issues.jsonl, etc.) prevent
	// the redirect from being created and should be removed regardless.
	if isRedirectExpected {
		r, e := cleanCruftIssueGraphDirFiles(issuegraphDir)
		removed += r
		errCount += e
	}

	return
}

// hasDoltDir returns true if the .issuegraph directory contains a dolt/ subdirectory.
func hasDoltDir(issuegraphDir string) bool {
	info, err := os.Stat(getDatabasePath(issuegraphDir))
	return err == nil && info.IsDir()
}

// isRedirectExpectedLocation returns true if this .issuegraph directory should contain
// only a redirect file.
func isRedirectExpectedLocation(issuegraphDir string) bool {
	parent := filepath.Dir(issuegraphDir)
	grandparent := filepath.Dir(parent)
	grandparentName := filepath.Base(grandparent)
	parentName := filepath.Base(parent)

	// Pattern: */polecats/*/.issuegraph/ (orchestrator worker — backwards compat)
	if grandparentName == "polecats" {
		return true
	}
	// Pattern: */crew/*/.issuegraph/ (orchestrator assistant — backwards compat)
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
	// Rig-root .issuegraph/ with a mayor/rig/.issuegraph/ canonical location
	canonicalDir := filepath.Join(parent, "mayor", "rig", ".issuegraph")
	if _, err := os.Stat(canonicalDir); err == nil {
		// Also check it has polecats/ or mayor/ siblings
		for _, sibling := range []string{"mayor", "polecats"} {
			if info, err := os.Stat(filepath.Join(parent, sibling)); err == nil && info.IsDir() {
				return true
			}
		}
	}

	return false
}

// cleanJSONLArtifacts removes stale JSONL files from a dolt-native .issuegraph directory.
func cleanJSONLArtifacts(issuegraphDir string) (removed, skipped, errCount int) {
	// Safe to delete (not the primary data source)
	safeFiles := []string{
		"issues.jsonl.new",
	}

	for _, name := range safeFiles {
		path := filepath.Join(issuegraphDir, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := os.Remove(path); err != nil {
			fmt.Printf("  Error removing %s: %v\n", path, err)
			errCount++
			continue
		}
		fmt.Printf("  Removed: %s (JSONL artifact)\n", path)
		removed++
	}

	// interactions.jsonl - only remove if empty
	interPath := filepath.Join(issuegraphDir, "interactions.jsonl")
	if info, err := os.Stat(interPath); err == nil {
		if info.Size() == 0 {
			if err := os.Remove(interPath); err != nil {
				fmt.Printf("  Error removing %s: %v\n", interPath, err)
				errCount++
			} else {
				fmt.Printf("  Removed: %s (empty interactions log)\n", interPath)
				removed++
			}
		} else {
			fmt.Printf("  Skip (not empty): %s\n", interPath)
			skipped++
		}
	}

	// issues.jsonl in dolt-native directory - skip (needs manual review)
	issuesPath := filepath.Join(issuegraphDir, "issues.jsonl")
	if _, err := os.Stat(issuesPath); err == nil {
		fmt.Printf("  Skip (needs review): %s (issues.jsonl in dolt-native dir)\n", issuesPath)
		skipped++
	}

	return
}

// cleanSQLiteArtifacts removes leftover SQLite database files.
func cleanSQLiteArtifacts(issuegraphDir string) (removed, skipped, errCount int) {
	entries, err := os.ReadDir(issuegraphDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// WAL and SHM files are always safe to delete
		if name == "issuegraph.db-shm" || name == "issuegraph.db-wal" {
			path := filepath.Join(issuegraphDir, name)
			if err := os.Remove(path); err != nil {
				fmt.Printf("  Error removing %s: %v\n", path, err)
				errCount++
				continue
			}
			fmt.Printf("  Removed: %s (SQLite WAL/SHM)\n", path)
			removed++
			continue
		}

		// issuegraph.db - skip (needs manual review, could be active)
		if name == "issuegraph.db" {
			path := filepath.Join(issuegraphDir, name)
			fmt.Printf("  Skip (needs review): %s\n", path)
			skipped++
			continue
		}

		// Backup databases are safe to delete
		if strings.HasPrefix(name, "issuegraph.backup-") && strings.HasSuffix(name, ".db") {
			path := filepath.Join(issuegraphDir, name)
			if err := os.Remove(path); err != nil {
				fmt.Printf("  Error removing %s: %v\n", path, err)
				errCount++
				continue
			}
			fmt.Printf("  Removed: %s (pre-migration backup)\n", path)
			removed++
		}
	}

	return
}

// cleanCruftIssueGraphDirFiles removes everything from a .issuegraph directory except
// the redirect file, .gitkeep, and (when a redirect file is present)
// metadata.json.
func cleanCruftIssueGraphDirFiles(issuegraphDir string) (removed, errCount int) {
	entries, err := os.ReadDir(issuegraphDir)
	if err != nil {
		return 0, 1
	}

	// borg0ai/dx-issuegraph#4692: a co-located redirect and metadata.json is a
	// documented, supported topology (see fb51196f7 / docs/reference/
	// advanced.md "Database Redirects" -- a server-mode source rig's own
	// metadata.json supplies its dolt_database while the redirect points at
	// the shared Gas Town root). Deleting metadata.json here would corrupt
	// that source rig's identity while leaving the redirect itself intact,
	// so never delete it when a redirect is present in the same directory.
	hasRedirect := false
	for _, entry := range entries {
		if entry.Name() == "redirect" {
			hasRedirect = true
			break
		}
	}

	for _, entry := range entries {
		name := entry.Name()
		// Keep redirect and .gitkeep
		if name == "redirect" || name == ".gitkeep" {
			continue
		}
		if name == "metadata.json" && hasRedirect {
			continue
		}

		entryPath := filepath.Join(issuegraphDir, name)

		// Validate path doesn't escape
		rel, err := filepath.Rel(issuegraphDir, entryPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}

		if entry.IsDir() {
			if err := os.RemoveAll(entryPath); err != nil {
				fmt.Printf("  Error removing %s: %v\n", entryPath, err)
				errCount++
				continue
			}
		} else {
			if err := os.Remove(entryPath); err != nil {
				fmt.Printf("  Error removing %s: %v\n", entryPath, err)
				errCount++
				continue
			}
		}
		fmt.Printf("  Removed: %s (cruft in redirect-only dir)\n", entryPath)
		removed++
	}

	return
}
