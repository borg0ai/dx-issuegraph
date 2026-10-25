package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

// openReadOnlyStoreForDBPath reopens a read-only store from an existing dbPath
// while preserving repo-local metadata such as dolt_database and the resolved
// Dolt server port. Falls back to deriving the issuegraph directory from the dbPath
// parent when no matching metadata.json can be found.
func openReadOnlyStoreForDBPath(ctx context.Context, dbPath string) (storage.DoltStorage, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("no database path available")
	}

	if issuegraphDir := resolveIssueGraphDirForDBPath(dbPath); issuegraphDir != "" {
		return newReadOnlyStoreFromConfig(ctx, issuegraphDir)
	}

	// Fallback: derive issuegraph dir from dbPath parent directory.
	return newReadOnlyStoreFromConfig(ctx, filepath.Dir(dbPath))
}

// resolveIssueGraphDirForDBPath maps a database path back to its owning .issuegraph
// directory when metadata.json is available. This is needed for repos that use
// non-default dolt_database names or custom dolt_data_dir locations.
func resolveIssueGraphDirForDBPath(dbPath string) string {
	actualDBPath := utils.CanonicalizePath(dbPath)
	if parent := filepath.Dir(dbPath); filepath.Base(parent) == ".issuegraph" {
		if _, err := os.Stat(filepath.Join(parent, "metadata.json")); err == nil {
			return parent
		}
	}
	if parent := filepath.Dir(actualDBPath); filepath.Base(parent) == ".issuegraph" {
		if _, err := os.Stat(filepath.Join(parent, "metadata.json")); err == nil {
			return parent
		}
	}
	seen := map[string]struct{}{}
	candidates := make([]string, 0, 16)

	addCandidate := func(path string) {
		if path == "" {
			return
		}
		key := utils.NormalizePathForComparison(path)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, path)
	}

	addAncestorCandidates := func(path string) {
		for dir := path; dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			addCandidate(filepath.Join(dir, ".issuegraph"))
			if filepath.Base(dir) == ".issuegraph" {
				addCandidate(dir)
			}
		}
	}

	if info, err := os.Stat(dbPath); err == nil && info.IsDir() {
		addCandidate(dbPath)
	}
	if info, err := os.Stat(actualDBPath); err == nil && info.IsDir() {
		addCandidate(actualDBPath)
	}

	addCandidate(filepath.Dir(dbPath))
	addCandidate(filepath.Dir(actualDBPath))
	addAncestorCandidates(filepath.Dir(dbPath))
	addAncestorCandidates(filepath.Dir(actualDBPath))

	if found := issuegraph.FindIssueGraphDir(); found != "" {
		addCandidate(found)
		addCandidate(utils.CanonicalizePath(found))
	}

	for _, issuegraphDir := range candidates {
		cfg, err := configfile.Load(issuegraphDir)
		if err != nil || cfg == nil {
			continue
		}
		if utils.PathsEqual(issuegraphDir, dbPath) || utils.PathsEqual(issuegraphDir, actualDBPath) {
			return issuegraphDir
		}
		// dbPath is the data directory living directly inside issuegraphDir
		// (<issuegraphDir>/embeddeddolt for embedded, <issuegraphDir>/dolt for server).
		// Match on that parent relationship so resolution works for embedded
		// stores regardless of the issuegraph dir name; Config.DatabasePath alone
		// returns the "dolt" name and misses embeddeddolt/ (GH#4574).
		if utils.PathsEqual(filepath.Dir(dbPath), issuegraphDir) || utils.PathsEqual(filepath.Dir(actualDBPath), issuegraphDir) {
			return issuegraphDir
		}
		if utils.PathsEqual(cfg.DatabasePath(issuegraphDir), dbPath) || utils.PathsEqual(cfg.DatabasePath(issuegraphDir), actualDBPath) {
			return issuegraphDir
		}
	}

	return ""
}
