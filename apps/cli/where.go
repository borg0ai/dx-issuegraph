package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
	"github.com/spf13/cobra"
)

// WhereResult contains information about the active issuegraph location
type WhereResult struct {
	Path           string `json:"path"`                      // Active .issuegraph directory path
	RedirectedFrom string `json:"redirected_from,omitempty"` // Original path if redirected
	Prefix         string `json:"prefix,omitempty"`          // Issue prefix (if detectable)
	DatabasePath   string `json:"database_path,omitempty"`   // Full path to database file
}

var whereCmd = &cobra.Command{
	Use:     "where",
	GroupID: "setup",
	Short:   "Show active issuegraph location",
	Long: `Show the active issuegraph database location, including redirect information.

	This command is useful for debugging when using redirects, to understand
	which issuegraph workspace is actually being used.

Examples:
  issuegraph where           # Show active issuegraph location
  issuegraph where --json    # Output in JSON format
`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		evt := metrics.NewCommandEvent("where")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		result := WhereResult{}

		if selected := selectedNoDBIssueGraphDir(cmd); selected != "" {
			prepareSelectedNoDBContext(selected)
		}

		issuegraphDir := resolveWhereIssueGraphDir(cmd)
		if issuegraphDir == "" {
			if jsonOutput {
				if jerr := outputJSON(map[string]string{
					"error":   "no_issuegraph_directory",
					"message": activeWorkspaceNotFoundMessage(),
					"hint":    whereDiagHint(),
				}); jerr != nil {
					return jerr
				}
				return SilentExit()
			}
			return HandleErrorWithHint(activeWorkspaceNotFoundMessage(), whereDiagHint())
		}

		result.Path = issuegraphDir

		// Check if we got here via redirect by looking for the original .issuegraph directory
		// Walk up from cwd to find any .issuegraph with a redirect file
		originalIssueGraphDir := findOriginalIssueGraphDir()
		if originalIssueGraphDir != "" && originalIssueGraphDir != issuegraphDir {
			result.RedirectedFrom = originalIssueGraphDir
		}

		// Find the database path
		dbPath := resolveWhereDatabasePath()
		if dbPath != "" {
			result.DatabasePath = dbPath
		}

		// Prefer the active workspace YAML when available. Avoid process-global
		// config here because `issuegraph where` may be reporting a workspace selected
		// by ISSUEGRAPH_DB/ISSUEGRAPH_DIR rather than the caller's current repository.
		if prefix := config.GetStringFromDir(issuegraphDir, "issue-prefix"); prefix != "" {
			result.Prefix = prefix
		} else if prefix := config.GetStringFromDir(issuegraphDir, "issue_prefix"); prefix != "" {
			result.Prefix = prefix
		} else if dbPath != "" && shouldReadWherePrefixFromStore(issuegraphDir) {
			_ = withStorage(getRootContext(), nil, dbPath, func(currentStore storage.DoltStorage) error {
				prefix, err := currentStore.GetConfig(getRootContext(), "issue_prefix")
				if err == nil && prefix != "" {
					result.Prefix = prefix
				}
				return nil
			})
		}

		if jsonOutput {
			return outputJSON(result)
		}
		fmt.Println(result.Path)
		if result.RedirectedFrom != "" {
			fmt.Printf("  (via redirect from %s)\n", result.RedirectedFrom)
		}
		if result.Prefix != "" {
			fmt.Printf("  prefix: %s\n", result.Prefix)
		}
		if result.DatabasePath != "" {
			fmt.Printf("  database: %s\n", result.DatabasePath)
		}
		return nil
	},
}

func resolveWhereIssueGraphDir(cmd *cobra.Command) string {
	if selected := selectedNoDBIssueGraphDir(cmd); selected != "" {
		return selected
	}

	return issuegraph.FindIssueGraphDir()
}

func resolveWhereDatabasePath() string {
	return issuegraph.FindDatabasePath()
}

func shouldReadWherePrefixFromStore(issuegraphDir string) bool {
	if issuegraphDir == "" {
		return false
	}

	cfg, err := configfile.LoadForDiscovery(issuegraphDir)
	if err != nil || cfg == nil {
		return true
	}

	// `issuegraph where` should be able to report selected metadata without requiring
	// a live Dolt server (or spawning the proxied-server daemon) just to
	// recover issue_prefix.
	return !cfg.IsDoltServerMode() && !cfg.IsDoltProxiedServerMode()
}

// findOriginalIssueGraphDir walks up from cwd looking for a .issuegraph directory with a redirect file
// Returns the original .issuegraph path if found, empty string otherwise
func findOriginalIssueGraphDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Canonicalize cwd to handle symlinks
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}

	// Check ISSUEGRAPH_DIR first: if the env points at a .issuegraph directory with a
	// redirect file, that's the original. Fall through to the fs walk if
	// ISSUEGRAPH_DIR is set but does not contain a redirect — issuegraph's startup now
	// rebinds ISSUEGRAPH_DIR to the resolved target (#3230) after following
	// redirects, so an unconditional early-return here would hide every
	// redirect from `issuegraph where` output.
	if envDir := os.Getenv("ISSUEGRAPH_DIR"); envDir != "" {
		envDir = utils.CanonicalizePath(envDir)
		redirectFile := filepath.Join(envDir, issuegraph.RedirectFileName)
		if _, err := os.Stat(redirectFile); err == nil {
			return envDir
		}
	}

	// Walk up directory tree looking for .issuegraph with redirect
	for dir := cwd; dir != "/" && dir != "."; {
		issuegraphDir := filepath.Join(dir, ".issuegraph")
		if info, err := os.Stat(issuegraphDir); err == nil && info.IsDir() {
			redirectFile := filepath.Join(issuegraphDir, issuegraph.RedirectFileName)
			if _, err := os.Stat(redirectFile); err == nil {
				return issuegraphDir
			}
			// Found .issuegraph without redirect - this is the actual location
			return ""
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

func init() {
	rootCmd.AddCommand(whereCmd)
}
