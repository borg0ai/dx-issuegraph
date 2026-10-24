package fix

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dolt"
	"github.com/borg0ai/dx-issuegraph/internal/types"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

// DatabaseVersion fixes database version mismatches by updating metadata in-process.
// For fresh clones (no database), it creates a new Dolt store.
// For existing databases, it updates version metadata directly.
//
// This runs in-process to avoid Dolt lock contention that occurs when spawning
// issuegraph subcommands while the parent process holds database connections. (GH#1805)
func DatabaseVersion(path string) error {
	return DatabaseVersionWithBdVersion(path, "")
}

// DatabaseVersionWithBdVersion is like DatabaseVersion but accepts an explicit
// issuegraph version string for setting the issuegraph_version metadata field.
func DatabaseVersionWithBdVersion(path string, issuegraphVersion string) error {
	issuegraphDir, err := resolvedWorkspaceIssueGraphDir(path)
	if err != nil {
		return err
	}

	// Load or create config
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		cfg = configfile.DefaultConfig()
	}
	if cfg == nil {
		cfg = configfile.DefaultConfig()
	}

	// Determine database path
	dbPath := cfg.DatabasePath(issuegraphDir)

	ctx := context.Background()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// No database - create a new Dolt store. Creation is this branch's
		// explicit purpose, so opt out of the dolt.New create-guard
		// (bd-kjfsq: without CreateIfMissing the guard fails the fix with
		// "database not found" on exactly the fresh clones it exists for).
		fmt.Println("  → No database found, creating Dolt store...")

		// Issue-mutating: the branch below imports every issue in the JSONL, so
		// it opens through the activating factory and those creates journal.
		store, err := openIssueMutatingStoreCreating(ctx, issuegraphDir)
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		defer func() { _ = store.Close() }()

		// Create local marker directory so FindDatabasePath can discover the
		// database. Server mode doesn't create it automatically (bd-u8rda).
		if mkErr := os.MkdirAll(dbPath, 0o750); mkErr != nil {
			fmt.Printf("  Warning: failed to create dolt marker dir: %v\n", mkErr)
		}

		// Set version metadata if provided
		if issuegraphVersion != "" {
			if err := store.SetLocalMetadata(ctx, "issuegraph_version", issuegraphVersion); err != nil {
				fmt.Printf("  Warning: failed to set issuegraph_version: %v\n", err)
			}
		}

		fmt.Println("  → Database created successfully")

		// Import from JSONL if present (fresh clone with committed issues).
		// This closes the chicken-and-egg gap where doctor --fix creates an
		// empty Dolt store and then issuegraph init refuses because the store exists.
		jsonlPath := filepath.Join(issuegraphDir, "issues.jsonl")
		if _, statErr := os.Stat(jsonlPath); statErr == nil {
			count, importErr := importJSONLIntoStore(ctx, store, jsonlPath)
			if importErr != nil {
				fmt.Printf("  Warning: failed to import from JSONL: %v\n", importErr)
			} else if count > 0 {
				fmt.Printf("  → Imported %d issues from issues.jsonl\n", count)
			}
		}

		return nil
	}

	// Database exists - update metadata in-process
	fmt.Println("  → Updating database metadata...")

	store, err := dolt.NewFromConfig(ctx, issuegraphDir)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = store.Close() }()

	// Update issuegraph_version if provided (clone-local, dolt-ignored)
	if issuegraphVersion != "" {
		if err := store.SetLocalMetadata(ctx, "issuegraph_version", issuegraphVersion); err != nil {
			return fmt.Errorf("failed to set issuegraph_version: %w", err)
		}
	}

	// Detect and set issue_prefix if missing
	prefix, err := store.GetConfig(ctx, "issue_prefix")
	if err != nil || prefix == "" {
		issues, err := store.SearchIssues(ctx, "", types.IssueFilter{})
		if err == nil && len(issues) > 0 {
			detectedPrefix := utils.ExtractIssuePrefix(issues[0].ID)
			if detectedPrefix != "" {
				if err := store.SetConfig(ctx, "issue_prefix", detectedPrefix); err != nil {
					fmt.Printf("  Warning: failed to set issue prefix: %v\n", err)
				} else {
					fmt.Printf("  → Detected and set issue prefix: %s\n", detectedPrefix)
				}
			}
		}
	}

	fmt.Println("  → Metadata updated")
	return nil
}

// SchemaCompatibility fixes schema compatibility issues by updating database metadata
func SchemaCompatibility(path string) error {
	return DatabaseVersion(path)
}

// FreshCloneImport handles the "Fresh Clone" fix: imports JSONL issues into an
// existing (possibly empty) Dolt store. This covers the case where the Database
// fix already created the store but a prior version didn't import.
func FreshCloneImport(path string, issuegraphVersion string) error {
	issuegraphDir, err := resolvedWorkspaceIssueGraphDir(path)
	if err != nil {
		return err
	}

	// Check for JSONL file
	jsonlPath := filepath.Join(issuegraphDir, "issues.jsonl")
	if _, err := os.Stat(jsonlPath); os.IsNotExist(err) {
		return fmt.Errorf("no issues.jsonl found")
	}

	// Check if Dolt store exists
	doltDir := getDatabasePath(issuegraphDir)
	if _, err := os.Stat(doltDir); os.IsNotExist(err) {
		// No Dolt store — delegate to Database fix which creates store + imports
		return DatabaseVersionWithBdVersion(path, issuegraphVersion)
	}

	// Dolt store exists — check if it already has issues. Issue-mutating: an
	// empty store is imported into below, so it opens through the activating
	// factory and those creates journal.
	ctx := context.Background()
	store, err := openIssueMutatingStore(ctx, issuegraphDir)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = store.Close() }()

	var issueCount int
	if err := store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM issues").Scan(&issueCount); err == nil && issueCount > 0 {
		fmt.Printf("  → Database already has %d issues, skipping import\n", issueCount)
		return nil
	}

	// Empty store — import from JSONL
	count, importErr := importJSONLIntoStore(ctx, store, jsonlPath)
	if importErr != nil {
		return fmt.Errorf("failed to import from JSONL: %w", importErr)
	}
	fmt.Printf("  → Imported %d issues from issues.jsonl\n", count)
	return nil
}

// importJSONLIntoStore reads a JSONL file and imports all issues into the Dolt store.
// Used by both the Database fix (new store creation) and Fresh Clone fix (empty store).
func importJSONLIntoStore(ctx context.Context, store storage.DoltStorage, jsonlPath string) (int, error) {
	f, err := os.Open(jsonlPath) // #nosec G304 - workspace-controlled path
	if err != nil {
		return 0, fmt.Errorf("failed to open JSONL file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 64*1024*1024) // 64MB max line
	var issues []*types.Issue

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var issue types.Issue
		if err := json.Unmarshal([]byte(line), &issue); err != nil {
			return 0, fmt.Errorf("failed to parse issue: %w", err)
		}
		issue.SetDefaults()
		issues = append(issues, &issue)
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan JSONL: %w", err)
	}

	if len(issues) == 0 {
		return 0, nil
	}

	// Auto-detect and set prefix from first issue
	configuredPrefix, _ := store.GetConfig(ctx, "issue_prefix")
	if strings.TrimSpace(configuredPrefix) == "" {
		firstPrefix := utils.ExtractIssuePrefix(issues[0].ID)
		if firstPrefix != "" {
			if err := store.SetConfig(ctx, "issue_prefix", firstPrefix); err != nil {
				fmt.Printf("  Warning: failed to set issue_prefix: %v\n", err)
			} else {
				fmt.Printf("  → Detected issue prefix: %s\n", firstPrefix)
			}
		}
	}

	// Determine actor for the import
	actor := detectActor()

	err = store.CreateIssuesWithFullOptions(ctx, issues, actor, storage.BatchCreateOptions{
		SkipPrefixValidation: true,
	})
	if err != nil {
		return 0, err
	}

	return len(issues), nil
}

// detectActor returns the best available actor name for automated operations.
func detectActor() string {
	if issuegraphActor := os.Getenv("ISSUEGRAPH_ACTOR"); issuegraphActor != "" {
		return issuegraphActor
	}
	if out, err := exec.Command("git", "config", "user.name").Output(); err == nil {
		if gitUser := strings.TrimSpace(string(out)); gitUser != "" {
			return gitUser
		}
	}
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "bd-doctor"
}
