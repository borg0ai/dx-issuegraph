// Package main provides the issuegraph CLI commands.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/github"
	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/borg0ai/dx-issuegraph/internal/tracker"
	"github.com/borg0ai/dx-issuegraph/internal/types"
	"github.com/spf13/cobra"
)

// GitHubConfig holds GitHub connection configuration.
type GitHubConfig struct {
	Token      string // Token resolved from gh auth or GITHUB_TOKEN
	Owner      string // Repository owner (user or organization)
	Repo       string // Repository name
	Repository string // Combined "owner/repo" format
	URL        string // Custom API URL (for GitHub Enterprise)
}

// githubCmd is the root command for GitHub operations.
var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub integration commands",
	Long: `Commands for syncing issues between issuegraph and GitHub.

Repository can be set via 'issuegraph config' or environment variables:
  github.owner / GITHUB_OWNER           - Repository owner
  github.repo / GITHUB_REPO             - Repository name
  github.repository / GITHUB_REPOSITORY - Combined "owner/repo" format
  github.url / GITHUB_API_URL           - Custom API URL (GitHub Enterprise)

Authenticate with 'gh auth login' or set GITHUB_TOKEN. IssueGraph does not
store GitHub tokens. Local commands never contact GitHub.`,
}

// githubSyncCmd synchronizes issues between issuegraph and GitHub.
var githubSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync issues with GitHub",
	Long: `Synchronize issues between issuegraph and GitHub.

By default, previews bidirectional sync without writing:
- Pulls new/updated issues from GitHub to issuegraph
- Pushes local issuegraph issues to GitHub

Use --pull-only or --push-only to limit direction. Pass --apply to write changes.
Conflicts preserve both current versions and are reported.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runGitHubSync,
}

// githubStatusCmd displays GitHub configuration and sync status.
var githubStatusCmd = &cobra.Command{
	Use:           "status",
	Short:         "Show GitHub sync status",
	Long:          `Display current GitHub configuration and sync status.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runGitHubStatus,
}

// githubReposCmd lists accessible GitHub repositories.
var githubReposCmd = &cobra.Command{
	Use:           "repos",
	Short:         "List accessible GitHub repositories",
	Long:          `List GitHub repositories that the configured token has access to.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runGitHubRepos,
}

var (
	githubSyncDryRun   bool
	githubSyncApply    bool
	githubSyncPullOnly bool
	githubSyncPushOnly bool
	githubPreferLocal  bool
	githubPreferGitHub bool
)

// GitHubConflictStrategy defines how to resolve conflicts between local and GitHub versions.
type GitHubConflictStrategy string

const (
	// GitHubConflictPreferNewer uses the most recently updated version (default).
	GitHubConflictPreferNewer GitHubConflictStrategy = "prefer-newer"
	// GitHubConflictPreferLocal always keeps the local issuegraph version.
	GitHubConflictPreferLocal GitHubConflictStrategy = "prefer-local"
	// GitHubConflictPreferGitHub always uses the GitHub version.
	GitHubConflictPreferGitHub GitHubConflictStrategy = "prefer-github"
)

// getGitHubConflictStrategy determines the conflict strategy from flag values.
// Returns error if multiple conflicting flags are set.
func getGitHubConflictStrategy(preferLocal, preferGitHub, preferNewer bool) (GitHubConflictStrategy, error) {
	flagsSet := 0
	if preferLocal {
		flagsSet++
	}
	if preferGitHub {
		flagsSet++
	}
	if preferNewer {
		flagsSet++
	}
	if flagsSet > 1 {
		return "", fmt.Errorf("cannot use multiple conflict resolution flags")
	}

	if preferLocal {
		return GitHubConflictPreferLocal, nil
	}
	if preferGitHub {
		return GitHubConflictPreferGitHub, nil
	}
	return GitHubConflictPreferNewer, nil
}

// parseGitHubSourceSystem parses a source system string like "github:https://...:42"
// Returns the issue number and ok (whether it's a valid GitHub source).
func parseGitHubSourceSystem(sourceSystem string) (number int, ok bool) {
	if !strings.HasPrefix(sourceSystem, "github:") {
		return 0, false
	}

	// Find last ":" and parse the number after it
	lastColon := strings.LastIndex(sourceSystem, ":")
	if lastColon == -1 || lastColon == len(sourceSystem)-1 {
		return 0, false
	}

	var err error
	number, err = strconv.Atoi(sourceSystem[lastColon+1:])
	if err != nil {
		return 0, false
	}

	return number, true
}

func init() {
	// Add subcommands to github
	githubCmd.AddCommand(githubSyncCmd)
	githubCmd.AddCommand(githubStatusCmd)
	githubCmd.AddCommand(githubReposCmd)

	// Add flags to sync command
	githubSyncCmd.Flags().BoolVar(&githubSyncDryRun, "dry-run", false, "Preview changes without applying them")
	githubSyncCmd.Flags().BoolVar(&githubSyncApply, "apply", false, "Apply planned local and GitHub changes")
	githubSyncCmd.Flags().BoolVar(&githubSyncPullOnly, "pull-only", false, "Only pull issues from GitHub")
	githubSyncCmd.Flags().BoolVar(&githubSyncPushOnly, "push-only", false, "Only push issues to GitHub")

	// Conflict resolution flags (mutually exclusive)
	githubSyncCmd.Flags().BoolVar(&githubPreferLocal, "prefer-local", false, "On conflict, keep local issuegraph version")
	githubSyncCmd.Flags().BoolVar(&githubPreferGitHub, "prefer-github", false, "On conflict, use GitHub version")
	registerSelectiveSyncFlags(githubSyncCmd)

	// Register github command with root
	rootCmd.AddCommand(githubCmd)
}

// getGitHubConfig returns GitHub configuration from issuegraph config or environment.
func getGitHubConfig() GitHubConfig {
	ctx := context.Background()
	config := GitHubConfig{}

	config.Token = getGitHubConfigValue(ctx, "github.token")
	config.Owner = getGitHubConfigValue(ctx, "github.owner")
	config.Repo = getGitHubConfigValue(ctx, "github.repo")
	config.Repository = getGitHubConfigValue(ctx, "github.repository")
	config.URL = getGitHubConfigValue(ctx, "github.url")

	// Parse combined owner/repo format if individual fields are empty
	if (config.Owner == "" || config.Repo == "") && config.Repository != "" {
		parts := strings.SplitN(config.Repository, "/", 2)
		if len(parts) == 2 {
			if config.Owner == "" {
				config.Owner = parts[0]
			}
			if config.Repo == "" {
				config.Repo = parts[1]
			}
		}
	}

	return config
}

// getGitHubConfigValue reads a GitHub configuration value from store or environment.
func getGitHubConfigValue(ctx context.Context, key string) string {
	if key == "github.token" {
		if output, err := exec.CommandContext(ctx, "gh", "auth", "token").Output(); err == nil {
			if token := strings.TrimSpace(string(output)); token != "" {
				return token
			}
		}
		return os.Getenv("GITHUB_TOKEN")
	}
	// Secret keys (e.g. github.token) are stored in config.yaml, not the
	// Dolt database, to avoid leaking secrets when pushing to remotes.
	if config.IsYamlOnlyKey(key) {
		if value := config.GetString(key); value != "" {
			return value
		}
		// Fall back to environment variable
		envKey := githubConfigToEnvVar(key)
		if envKey != "" {
			if value := os.Getenv(envKey); value != "" {
				return value
			}
		}
		return ""
	}

	// Try to read from store (works in direct mode)
	if store != nil {
		value, _ := store.GetConfig(ctx, key)
		if value != "" {
			return value
		}
	} else if dbPath != "" {
		tempStore, err := openReadOnlyStoreForDBPath(ctx, dbPath)
		if err == nil {
			defer func() { _ = tempStore.Close() }()
			value, _ := tempStore.GetConfig(ctx, key)
			if value != "" {
				return value
			}
		}
	}

	// Fall back to environment variable
	envKey := githubConfigToEnvVar(key)
	if envKey != "" {
		if value := os.Getenv(envKey); value != "" {
			return value
		}
	}

	return ""
}

// githubConfigToEnvVar maps GitHub config keys to their environment variable names.
func githubConfigToEnvVar(key string) string {
	switch key {
	case "github.token":
		return "GITHUB_TOKEN"
	case "github.owner":
		return "GITHUB_OWNER"
	case "github.repo":
		return "GITHUB_REPO"
	case "github.repository":
		return "GITHUB_REPOSITORY"
	case "github.url":
		return "GITHUB_API_URL"
	default:
		return ""
	}
}

// validateGitHubConfig checks that required configuration is present.
func validateGitHubConfig(config GitHubConfig) error {
	if config.Token == "" {
		return fmt.Errorf("GitHub authentication unavailable; run 'gh auth login' or set GITHUB_TOKEN")
	}
	if config.Owner == "" {
		return fmt.Errorf("github.owner is not configured. Set via 'issuegraph config set github.owner <owner>' or GITHUB_OWNER environment variable")
	}
	if config.Repo == "" {
		return fmt.Errorf("github.repo is not configured. Set via 'issuegraph config set github.repo <repo>' or GITHUB_REPO environment variable")
	}
	return nil
}

// maskGitHubToken masks a token for safe display.
// Shows only the first 4 characters to aid identification without
// revealing enough to reduce brute-force entropy.
func maskGitHubToken(token string) string {
	if token == "" {
		return "(not set)"
	}
	if len(token) <= 4 {
		return "****"
	}
	return token[:4] + "****"
}

// getGitHubClient creates a GitHub client from the current configuration.
func getGitHubClient(config GitHubConfig) *github.Client {
	client := github.NewClient(config.Token, config.Owner, config.Repo)
	if config.URL != "" {
		client = client.WithBaseURL(config.URL)
	}
	return client
}

// runGitHubStatus implements the github status command.
func runGitHubStatus(cmd *cobra.Command, args []string) error {
	if usesProxiedServer() {
		return HandleErrorRespectJSON("github status is not supported in proxied-server mode")
	}
	evt := metrics.NewCommandEvent("github-status")
	defer func() {
		if c := metrics.Global(); c != nil {
			c.CloseEventAndAdd(evt)
		}
	}()

	config := getGitHubConfig()

	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(out, "GitHub Configuration")
	_, _ = fmt.Fprintln(out, "====================")
	authStatus := "not available"
	if config.Token != "" {
		authStatus = "available"
	}
	_, _ = fmt.Fprintf(out, "Authentication: %s\n", authStatus)
	_, _ = fmt.Fprintf(out, "Owner:      %s\n", config.Owner)
	_, _ = fmt.Fprintf(out, "Repository: %s\n", config.Repo)
	if config.URL != "" {
		_, _ = fmt.Fprintf(out, "API URL:    %s\n", config.URL)
	}

	// Validate configuration
	if err := validateGitHubConfig(config); err != nil {
		_, _ = fmt.Fprintf(out, "\nStatus: ❌ Not configured\n")
		_, _ = fmt.Fprintf(out, "Error: %v\n", err)
		return nil
	}

	_, _ = fmt.Fprintf(out, "\nStatus: ✓ Configured\n")
	return nil
}

// runGitHubRepos implements the github repos command.
func runGitHubRepos(cmd *cobra.Command, args []string) error {
	if usesProxiedServer() {
		return HandleErrorRespectJSON("github repos is not supported in proxied-server mode")
	}
	evt := metrics.NewCommandEvent("github-repos")
	defer func() {
		if c := metrics.Global(); c != nil {
			c.CloseEventAndAdd(evt)
		}
	}()

	config := getGitHubConfig()
	if config.Token == "" {
		return HandleError("GitHub authentication unavailable; run 'gh auth login' or set GITHUB_TOKEN")
	}

	out := cmd.OutOrStdout()
	client := getGitHubClient(config)
	ctx := context.Background()

	repos, err := client.ListRepositories(ctx)
	if err != nil {
		return HandleError("failed to fetch repositories: %v", err)
	}

	_, _ = fmt.Fprintln(out, "Accessible GitHub Repositories")
	_, _ = fmt.Fprintln(out, "==============================")
	for _, r := range repos {
		_, _ = fmt.Fprintf(out, "  %s\n", r.FullName)
		if r.Description != "" {
			_, _ = fmt.Fprintf(out, "    %s\n", r.Description)
		}
		_, _ = fmt.Fprintf(out, "    %s\n", r.HTMLURL)
		_, _ = fmt.Fprintln(out)
	}

	if len(repos) == 0 {
		_, _ = fmt.Fprintln(out, "No repositories found")
	}

	return nil
}

// runGitHubSync implements the github sync command.
// Uses the tracker.Engine for all sync operations.
func runGitHubSync(cmd *cobra.Command, args []string) error {
	if usesProxiedServer() {
		return HandleErrorRespectJSON("github sync is not supported in proxied-server mode")
	}
	evt := metrics.NewCommandEvent("github-sync")
	defer func() {
		if c := metrics.Global(); c != nil {
			c.CloseEventAndAdd(evt)
		}
	}()

	config := getGitHubConfig()
	if err := validateGitHubConfig(config); err != nil {
		return HandleError("%v", err)
	}

	if githubSyncDryRun && githubSyncApply {
		return HandleError("cannot use --dry-run and --apply together")
	}
	dryRun := !githubSyncApply || githubSyncDryRun
	if !dryRun {
		CheckReadonly("github sync")
	}

	if githubSyncPullOnly && githubSyncPushOnly {
		return HandleError("cannot use both --pull-only and --push-only")
	}

	if err := ensureStoreActive(); err != nil {
		return HandleError("database not available: %v", err)
	}

	out := cmd.OutOrStdout()
	ctx := context.Background()

	gt := &github.Tracker{}
	if err := gt.Init(ctx, store); err != nil {
		return HandleError("initializing GitHub tracker: %v", err)
	}

	// Create the sync engine
	engine := tracker.NewEngine(gt, store, actor)
	engine.OnMessage = func(msg string) { _, _ = fmt.Fprintln(out, "  "+msg) }
	engine.OnWarning = func(msg string) { _, _ = fmt.Fprintf(os.Stderr, "Warning: %s\n", msg) }

	// Set up GitHub-specific pull and push hooks
	engine.PullHooks = buildGitHubPullHooks(ctx)
	engine.PushHooks = buildGitHubPushHooks(gt)

	// Build sync options from CLI flags
	pull := !githubSyncPushOnly
	push := !githubSyncPullOnly

	opts := tracker.SyncOptions{
		Pull:   pull,
		Push:   push,
		DryRun: dryRun,
	}

	if err := applySelectiveSyncFlags(cmd, &opts, push); err != nil {
		return HandleError("%v", err)
	}

	opts.ConflictResolution = tracker.ConflictPreserve

	if dryRun {
		_, _ = fmt.Fprintln(out, "Dry run mode - no changes will be made")
		_, _ = fmt.Fprintln(out)
	}

	result, err := engine.Sync(ctx, opts)
	if err != nil {
		return HandleError("%v", err)
	}

	// Output results
	if !dryRun {
		if result.Stats.Pulled > 0 {
			_, _ = fmt.Fprintf(out, "✓ Pulled %d issues (%d created, %d updated)\n",
				result.Stats.Pulled, result.Stats.Created, result.Stats.Updated)
		}
		if result.Stats.Pushed > 0 {
			_, _ = fmt.Fprintf(out, "✓ Pushed %d issues\n", result.Stats.Pushed)
		}
		if result.Stats.Conflicts > 0 {
			_, _ = fmt.Fprintf(out, "→ Preserved %d conflicts\n", result.Stats.Conflicts)
		}
	}

	if dryRun {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "Run again with --apply to apply changes")
	}

	return nil
}

// buildGitHubPushHooks creates PushHooks for GitHub-specific push behavior.
// The ContentEqual hook lets the engine skip issues whose pushable fields
// already match GitHub, so repeated `github sync --push-only` / `github push`
// runs don't re-PATCH unchanged issues (borg0ai/dx-issuegraph#4214).
func buildGitHubPushHooks(gt *github.Tracker) *tracker.PushHooks {
	config := gt.MappingConfig()
	if config == nil {
		config = github.DefaultMappingConfig()
	}
	return &tracker.PushHooks{
		ContentEqual: func(local *types.Issue, remote *tracker.TrackerIssue) bool {
			if remote == nil {
				return false
			}
			gh, ok := remote.Raw.(*github.Issue)
			if !ok || gh == nil {
				return false
			}
			return github.PushFieldsEqual(local, gh, config)
		},
		// ContentHash lets the engine skip the per-issue GitHub fetch entirely
		// when an issue is unchanged since its last push, so a no-op
		// `github sync --push-only` makes ~zero REST calls instead of one GET
		// per linked issue (borg0ai/dx-issuegraph#4214).
		ContentHash: func(local *types.Issue) string {
			return github.PushContentHash(local, config)
		},
		// TargetScope supplies the host and repository omitted by shorthand refs
		// such as github:42, so changing GitHub target configuration invalidates
		// the local no-op cache.
		TargetScope: gt.PushTargetScope,
	}
}

// buildGitHubPullHooks creates PullHooks for GitHub-specific pull behavior.
func buildGitHubPullHooks(ctx context.Context) *tracker.PullHooks {
	prefix := "issuegraph"
	// YAML config takes precedence — in shared-server mode the DB
	// may belong to a different project (GH#2469).
	if p := config.GetString("issue-prefix"); p != "" {
		prefix = p
	} else if store != nil {
		if p, err := store.GetConfig(ctx, "issue_prefix"); err == nil && p != "" {
			prefix = p
		}
	}

	return &tracker.PullHooks{
		GenerateID: func(_ context.Context, issue *types.Issue) error {
			if issue.ID == "" {
				issue.ID = generateIssueID(prefix)
			}
			return nil
		},
	}
}
