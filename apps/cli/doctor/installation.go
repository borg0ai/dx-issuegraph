package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/git"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dolt"
	"github.com/borg0ai/dx-issuegraph/modules/cli/doctor/fix"
)

// CheckInstallation verifies that .issuegraph directory exists
func CheckInstallation(path string) DoctorCheck {
	issuegraphDir := ResolveIssueGraphDirForRepo(path)
	if _, err := os.Stat(issuegraphDir); os.IsNotExist(err) {
		// Auto-detect prefix from directory name
		prefix := filepath.Base(path)
		prefix = strings.TrimRight(prefix, "-")

		return DoctorCheck{
			Name:    "Installation",
			Status:  StatusError,
			Message: "No .issuegraph/ directory found",
			Fix:     fmt.Sprintf("Run 'issuegraph init --prefix %s' to initialize issuegraph", prefix),
		}
	}

	return DoctorCheck{
		Name:    "Installation",
		Status:  StatusOK,
		Message: ".issuegraph/ directory found",
	}
}

// CheckPermissions verifies that .issuegraph directory and database are readable/writable.
// Opens its own store; prefer CheckPermissionsWithStore when a shared store is available.
func CheckPermissions(path string) DoctorCheck {
	issuegraphDir := ResolveIssueGraphDirForRepo(path)

	// Check if .issuegraph/ is writable
	testFile := filepath.Join(issuegraphDir, ".doctor-test-write")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		return DoctorCheck{
			Name:    "Permissions",
			Status:  StatusError,
			Message: ".issuegraph/ directory is not writable",
			Fix:     "Run 'issuegraph doctor --fix' to fix permissions",
		}
	}
	_ = os.Remove(testFile) // Clean up test file (intentionally ignore error)

	// Check Dolt database directory permissions
	cfg, err := configfile.Load(issuegraphDir)
	if err == nil && cfg != nil && cfg.GetBackend() == configfile.BackendDolt {
		doltPath := getDatabasePath(issuegraphDir)
		if info, err := os.Stat(doltPath); err == nil {
			if !info.IsDir() {
				return DoctorCheck{
					Name:    "Permissions",
					Status:  StatusError,
					Message: "dolt/ is not a directory",
					Fix:     "Run 'issuegraph doctor --fix' to fix permissions",
				}
			}
			// Try to open Dolt store read-only to verify accessibility
			ctx := context.Background()
			store, err := dolt.NewFromConfigWithCLIOptions(ctx, issuegraphDir, &dolt.Config{ReadOnly: true})
			if err != nil {
				return DoctorCheck{
					Name:    "Permissions",
					Status:  StatusError,
					Message: "Dolt database exists but cannot be opened",
					Detail:  err.Error(),
					Fix:     "Run 'issuegraph doctor --fix' to fix permissions",
				}
			}
			_ = store.Close()
		}
	}

	return DoctorCheck{
		Name:    "Permissions",
		Status:  StatusOK,
		Message: "All permissions OK",
	}
}

// CheckPermissionsWithStore verifies permissions using a shared store (GH#2636).
// If the shared store was opened successfully, the database is accessible.
func CheckPermissionsWithStore(path string, ss *SharedStore) DoctorCheck {
	issuegraphDir := issuegraphDirFromSharedStore(path, ss)
	store := ss.Store()

	// Check if .issuegraph/ is writable
	testFile := filepath.Join(issuegraphDir, ".doctor-test-write")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		return DoctorCheck{
			Name:    "Permissions",
			Status:  StatusError,
			Message: ".issuegraph/ directory is not writable",
			Fix:     "Run 'issuegraph doctor --fix' to fix permissions",
		}
	}
	_ = os.Remove(testFile)

	// Check Dolt database directory permissions
	cfg, err := configfile.Load(issuegraphDir)
	if err == nil && cfg != nil && cfg.GetBackend() == configfile.BackendDolt {
		if cfg.IsDoltServerMode() {
			if store == nil {
				return DoctorCheck{
					Name:    "Permissions",
					Status:  StatusError,
					Message: "Unable to verify Dolt server-backed database permissions",
					Fix:     "Check 'issuegraph dolt status' for server availability, then re-run 'issuegraph doctor'",
				}
			}
			return DoctorCheck{
				Name:    "Permissions",
				Status:  StatusOK,
				Message: "All permissions OK",
			}
		}

		doltPath := getDatabasePath(issuegraphDir)
		if info, err := os.Stat(doltPath); err == nil {
			if !info.IsDir() {
				return DoctorCheck{
					Name:    "Permissions",
					Status:  StatusError,
					Message: "dolt/ is not a directory",
					Fix:     "Run 'issuegraph doctor --fix' to fix permissions",
				}
			}
			// If shared store is nil, the database could not be opened
			if store == nil {
				return DoctorCheck{
					Name:    "Permissions",
					Status:  StatusError,
					Message: "Dolt database exists but cannot be opened",
					Fix:     "Run 'issuegraph doctor --fix' to fix permissions",
				}
			}
			// Shared store was opened successfully — database is accessible
		}
	}

	return DoctorCheck{
		Name:    "Permissions",
		Status:  StatusOK,
		Message: "All permissions OK",
	}
}

// CheckUntrackedIssueGraphFiles checks for untracked .issuegraph/*.jsonl files that should be committed.
// This check only applies to legacy (non-Dolt) backends where JSONL files are the data store.
// In sync-branch mode, JSONL files are intentionally untracked in working branches
// and only committed to the dedicated sync branch (GH#858).
func CheckUntrackedIssueGraphFiles(path string) DoctorCheck {
	backend, _ := getBackendAndIssueGraphDir(path)

	// Dolt backends store data on the server, not in JSONL files
	if backend == configfile.BackendDolt {
		return DoctorCheck{
			Name:    "Untracked Files",
			Status:  StatusOK,
			Message: "N/A (Dolt backend stores data on server)",
		}
	}

	issuegraphDir := ResolveIssueGraphDirForRepo(path)
	repoRoot := resolvedIssueGraphRepoRoot(path)

	// Skip if .issuegraph doesn't exist
	if _, err := os.Stat(issuegraphDir); os.IsNotExist(err) {
		return DoctorCheck{
			Name:    "Untracked Files",
			Status:  StatusOK,
			Message: "N/A (no .issuegraph directory)",
		}
	}

	// Check if we're in a git repository using worktree-aware detection
	_, err := git.GetGitDir()
	if err != nil {
		return DoctorCheck{
			Name:    "Untracked Files",
			Status:  StatusOK,
			Message: "N/A (not a git repository)",
		}
	}

	// Run git status --porcelain to find untracked files in .issuegraph/
	cmd := exec.Command("git", "status", "--porcelain", ".issuegraph/")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return DoctorCheck{
			Name:    "Untracked Files",
			Status:  StatusWarning,
			Message: "Unable to check git status",
			Detail:  err.Error(),
		}
	}

	// Parse output for untracked JSONL files (lines starting with "??")
	var untrackedJSONL []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Untracked files start with "?? "
		if strings.HasPrefix(line, "?? ") {
			file := strings.TrimPrefix(line, "?? ")
			// Only care about .jsonl files
			if strings.HasSuffix(file, ".jsonl") {
				untrackedJSONL = append(untrackedJSONL, filepath.Base(file))
			}
		}
	}

	if len(untrackedJSONL) == 0 {
		return DoctorCheck{
			Name:    "Untracked Files",
			Status:  StatusOK,
			Message: "All .issuegraph/*.jsonl files are tracked",
		}
	}

	return DoctorCheck{
		Name:    "Untracked Files",
		Status:  StatusWarning,
		Message: fmt.Sprintf("Untracked JSONL files: %s", strings.Join(untrackedJSONL, ", ")),
		Detail:  "These files should be committed to propagate changes to other clones",
		Fix:     "Run 'issuegraph doctor --fix' to stage and commit untracked files, or manually: git add .issuegraph/*.jsonl && git commit",
	}
}

// FixPermissions fixes file permission issues in the .issuegraph directory
func FixPermissions(path string) error {
	return fix.Permissions(path)
}
