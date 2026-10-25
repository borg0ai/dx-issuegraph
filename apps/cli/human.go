package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/borg0ai/dx-issuegraph/internal/types"
	"github.com/borg0ai/dx-issuegraph/internal/ui"
	"github.com/borg0ai/dx-issuegraph/internal/workapi"
	"github.com/borg0ai/dx-issuegraph/issueops"
	"github.com/spf13/cobra"
)

var humanCmd = &cobra.Command{
	Use:     "human",
	GroupID: "setup",
	Short:   "Show essential commands for human users",
	Long: `Display a focused help menu showing only the most common commands.

issuegraph has 70+ commands - many for AI agents, integrations, and advanced workflows.
This command shows the ~15 essential commands that human users need most often.

For the full command list, run: issuegraph --help

SUBCOMMANDS:
  human list              List human-needed issuegraph (issues with 'human' label; hides closed by default)
  human respond <id>      Respond to a human-needed issue (adds comment and closes)
  human dismiss <id>      Dismiss a human-needed issue permanently
  human stats             Show summary statistics for human-needed issuegraph`,
	Run: func(cmd *cobra.Command, args []string) {
		evt := metrics.NewCommandEvent("human")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		fmt.Printf("\n%s\n", ui.RenderBold("issuegraph - Essential Commands for Humans"))
		fmt.Printf("For all 70+ commands: issuegraph --help\n\n")

		// Issues - Core workflow
		fmt.Printf("%s\n", ui.RenderAccent("Working With Issues:"))
		printCmd("create", "Create a new issue")
		printCmd("list", "List issues (filter with --status, --priority, --label)")
		printCmd("show <id>", "Show issue details")
		printCmd("update <id>", "Update an issue (--status, --priority, --assignee)")
		printCmd("close <id>", "Close one or more issues")
		printCmd("reopen <id>", "Reopen a closed issue")
		printCmd("note <id> <text>", "Add a note to an issue (or: comments add <id>)")
		fmt.Println()

		// Workflow
		fmt.Printf("%s\n", ui.RenderAccent("Finding Work:"))
		printCmd("ready", "Show issues ready to work on (no blockers)")
		printCmd("search <query>", "Search issues by text")
		printCmd("status", "Show project overview and counts")
		printCmd("stats", "Show detailed statistics")
		fmt.Println()

		// Dependencies
		fmt.Printf("%s\n", ui.RenderAccent("Dependencies:"))
		printCmd("dep add <a> <b>", "Add dependency (a depends on b)")
		printCmd("dep remove <a> <b>", "Remove a dependency")
		printCmd("dep tree <id>", "Show dependency tree")
		printCmd("graph", "Display visual dependency graph")
		printCmd("blocked", "Show all blocked issues")
		fmt.Println()

		// Setup & Maintenance
		fmt.Printf("%s\n", ui.RenderAccent("Setup & Maintenance:"))
		printCmd("init", "Initialize issuegraph in current directory")
		printCmd("doctor", "Check installation health")
		fmt.Println()

		// Help
		fmt.Printf("%s\n", ui.RenderAccent("Getting Help:"))
		printCmd("quickstart", "Quick start guide with examples")
		printCmd("help <cmd>", "Help for any command")
		printCmd("--help", "Full command list (70+ commands)")
		fmt.Println()

		// Common examples
		fmt.Printf("%s\n", ui.RenderAccent("Quick Examples:"))
		fmt.Printf("  %s\n", ui.RenderMuted("# Create and track an issue"))
		fmt.Printf("  issuegraph create \"Fix login bug\" --priority 1\n")
		fmt.Printf("  issuegraph update bd-abc123 --claim\n")
		fmt.Printf("  issuegraph close bd-abc123\n\n")

		fmt.Printf("  %s\n", ui.RenderMuted("# See what needs doing"))
		fmt.Printf("  issuegraph ready                    # What can I work on?\n")
		fmt.Printf("  issuegraph list --status open       # All open issues\n")
		fmt.Printf("  issuegraph blocked                  # What's stuck?\n\n")
	},
}

// human list command
var humanListCmd = &cobra.Command{
	Use:   "list",
	Short: "List human-needed issuegraph",
	Long: `List issues labeled with 'human' tag.

These are issues that require human intervention or input. Every
human-labeled issue shows regardless of type (including gates and wisps).
By default closed, pinned, and other done/frozen issuegraph are hidden; use
--status to select specific statuses, or --status=all to include every
status.

Examples:
  issuegraph human list
  issuegraph human list --status=closed
  issuegraph human list --status=all
  issuegraph human list --json`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		evt := metrics.NewCommandEvent("human-list")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		status, _ := cmd.Flags().GetString("status")

		issues, err := humanIssues(rootCtx, status)
		if err != nil {
			return HandleErrorRespectJSON("listing human issuegraph: %v", err)
		}

		if jsonOutput {
			// SearchIssues already hydrated .Labels on both backends (the
			// filter never sets SkipLabels), so the issues marshal with their
			// labels attached.
			data, err := json.MarshalIndent(issues, "", "  ")
			if err != nil {
				return HandleErrorRespectJSON("encoding JSON: %v", err)
			}
			fmt.Println(string(data))
			return nil
		}

		printHumanList(issues)
		return nil
	},
}

// humanIssues loads the human-labeled issuegraph matching the status selector
// through whichever backend this invocation uses: the direct store, or a
// proxied-server unit of work. Both routes apply the same humanListFilter.
func humanIssues(ctx context.Context, status string) ([]*types.Issue, error) {
	if usesProxiedServer() {
		return proxiedHumanIssues(ctx, status)
	}

	if err := ensureStoreActive(); err != nil {
		return nil, err
	}
	cfg, err := workapi.LoadStoreListConfig(ctx, store)
	if err != nil {
		return nil, fmt.Errorf("loading status configuration: %w", err)
	}
	filter, err := humanListFilter(status, cfg)
	if err != nil {
		return nil, err
	}
	return store.SearchIssues(ctx, "", filter)
}

// humanListRequest builds the list request for "human list": every
// human-labeled issue regardless of type — a human label is an explicit
// request for a person's attention, and this command is the only place a
// person sees it, so gates, wisps, infra issuegraph, and templates all show.
// Status filtering matches `issuegraph list`: done/frozen statuses and pinned issuegraph
// are hidden by default, an explicit --status selects specific statuses
// (validated against the built-in and custom status names), and "all" lifts
// the status defaults entirely.
func humanListRequest(status string) issueops.ListRequest {
	unlimited := 0
	req := issueops.ListRequest{
		Labels: []string{"human"},
		Limit:  &unlimited, // human list has never paginated; 0 = unlimited
		// No issue type is hidden from a human listing.
		IncludeAllTypes: true,
	}
	req.Status = status // "" applies the defaults; "all" lifts them
	return req
}

// humanListFilter resolves humanListRequest against loaded status/type config.
func humanListFilter(status string, cfg workapi.ListConfig) (types.IssueFilter, error) {
	return workapi.BuildListFilter(humanListRequest(status), cfg)
}

// warnIfNotHumanLabeled warns on stderr when the issue lacks the 'human'
// label. The check is advisory — respond/dismiss act on whichever issue the
// user named. The issue must carry its labels; anything returned by
// GetIssue/resolveAndGetIssueForMutation does.
func warnIfNotHumanLabeled(issue *types.Issue) {
	if !slices.Contains(issue.Labels, "human") {
		fmt.Fprintf(os.Stderr, "Warning: Issue %s does not have 'human' label\n", issue.ID)
	}
}

func printHumanList(issues []*types.Issue) {
	if len(issues) == 0 {
		fmt.Println("No human-needed issuegraph found.")
		return
	}

	fmt.Printf("\n%s (%d found)\n\n", ui.RenderBold("Human-needed issuegraph"), len(issues))
	for _, issue := range issues {
		if issue.Status == types.StatusClosed {
			// Closed items fade to muted gray, matching issuegraph list/query. They
			// only appear under an explicit --status, so the fade is what
			// distinguishes them from the work still waiting on a person.
			fmt.Printf("  %s\n", ui.RenderClosedLine(fmt.Sprintf("%s %s", issue.ID, issue.Title)))
			fmt.Printf("    %s\n", ui.RenderClosedLine(fmt.Sprintf("Status: %s", issue.Status)))
			fmt.Printf("    %s\n", ui.RenderClosedLine(fmt.Sprintf("Priority: P%d", issue.Priority)))
			fmt.Println()
			continue
		}
		fmt.Printf("  %s %s\n", ui.RenderCommand(issue.ID), issue.Title)
		if issue.Status != "open" {
			fmt.Printf("    Status: %s\n", issue.Status)
		}
		fmt.Printf("    Priority: %s\n", ui.RenderPriorityForStatus(issue.Priority, string(issue.Status)))
		fmt.Println()
	}
}

// human respond command
var humanRespondCmd = &cobra.Command{
	Use:   "respond <issue-id> [response...]",
	Short: "Respond to a human-needed issue",
	Long: `Respond to a human-needed issue by adding a comment and closing it.

The response is added as a comment and the issue is closed with reason "Responded".
The response text can be given as positional arguments, --response, --file, or --stdin.

Examples:
  issuegraph human respond bd-123 "Use OAuth2 for authentication"
  issuegraph human respond bd-123 -r "Approved, proceed with implementation"
  issuegraph human respond bd-123 --file response.md
  echo "Approved" | issuegraph human respond bd-123 --stdin`,
	Args:          cobra.MinimumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		CheckReadonly("human respond")

		evt := metrics.NewCommandEvent("human-respond")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		src := cmdTextSources(cmd, args[1:])
		src.flagText, _ = cmd.Flags().GetString("response")
		src.flagName = "--response"
		src.flagSet = cmd.Flags().Changed("response")
		response, err := requireTextFromSources("response text", "use positional args, --response, --file, or --stdin", src)
		if err != nil {
			return HandleErrorRespectJSON("%v", err)
		}

		ctx := rootCtx
		issueID := args[0]
		// Formatted once here so the direct and proxied backends store
		// identically-shaped comments.
		commentText := fmt.Sprintf("Response: %s", response)

		if usesProxiedServer() {
			return runHumanRespondProxiedServer(ctx, issueID, commentText)
		}

		// Direct mode
		if err := ensureStoreActive(); err != nil {
			return HandleErrorRespectJSON("responding to human issue: %v", err)
		}

		// Resolve partial ID and get issue
		result, err := resolveAndGetIssueForMutation(ctx, store, issueID)
		if err != nil {
			return HandleErrorRespectJSON("resolving issue ID %s: %v", issueID, err)
		}
		if result == nil || result.Issue == nil {
			if result != nil {
				result.Close()
			}
			return HandleErrorRespectJSON("issue not found: %s", issueID)
		}
		defer result.Close()

		resolvedID := result.ResolvedID
		issue := result.Issue
		targetStore := result.Store

		if issue.Status == "closed" {
			return HandleErrorRespectJSON("issue %s is already closed", resolvedID)
		}

		warnIfNotHumanLabeled(issue)

		_, err = targetStore.AddIssueComment(ctx, resolvedID, actor, commentText)
		if err != nil {
			return HandleErrorRespectJSON("adding comment: %v", err)
		}

		if err := targetStore.CloseIssue(ctx, resolvedID, "Responded", actor, ""); err != nil {
			return HandleErrorRespectJSON("closing issue: %v", err)
		}

		fmt.Printf("%s Issue %s closed with response.\n", ui.RenderPass("✔"), resolvedID)
		return nil
	},
}

// human dismiss command
var humanDismissCmd = &cobra.Command{
	Use:   "dismiss <issue-id> [reason...]",
	Short: "Dismiss a human-needed issue",
	Long: `Dismiss a human-needed issue permanently without responding.

The issue is closed with a "Dismissed" reason and optional note.
The reason can be given as positional arguments or --reason.

Examples:
  issuegraph human dismiss bd-123
  issuegraph human dismiss bd-123 "No longer applicable"
  issuegraph human dismiss bd-123 --reason "No longer applicable"`,
	Args:          cobra.MinimumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		CheckReadonly("human dismiss")

		evt := metrics.NewCommandEvent("human-dismiss")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		reasonFlag, _ := cmd.Flags().GetString("reason")

		// A dismissal reason is optional, so this is textFromSources rather
		// than requireTextFromSources: no source at all is fine, but naming
		// two is still the same hard error every other text input gives.
		reason, _, err := textFromSources(textSources{
			flagText:   reasonFlag,
			flagName:   "--reason",
			flagSet:    cmd.Flags().Changed("reason"),
			positional: args[1:],
		})
		if err != nil {
			return HandleErrorRespectJSON("%v", err)
		}
		reason = strings.TrimSpace(reason)

		closeReason := dismissedCloseReason
		if reason != "" {
			closeReason = fmt.Sprintf("%s: %s", dismissedCloseReason, reason)
		}

		ctx := rootCtx
		issueID := args[0]

		if usesProxiedServer() {
			return runHumanDismissProxiedServer(ctx, issueID, closeReason)
		}

		// Direct mode
		if err := ensureStoreActive(); err != nil {
			return HandleErrorRespectJSON("dismissing human issue: %v", err)
		}

		// Resolve partial ID and get issue
		result, err := resolveAndGetIssueForMutation(ctx, store, issueID)
		if err != nil {
			return HandleErrorRespectJSON("resolving issue ID %s: %v", issueID, err)
		}
		if result == nil || result.Issue == nil {
			if result != nil {
				result.Close()
			}
			return HandleErrorRespectJSON("issue not found: %s", issueID)
		}
		defer result.Close()

		resolvedID := result.ResolvedID
		issue := result.Issue
		targetStore := result.Store

		if issue.Status == "closed" {
			return HandleErrorRespectJSON("issue %s is already closed", resolvedID)
		}

		warnIfNotHumanLabeled(issue)

		if err := targetStore.CloseIssue(ctx, resolvedID, closeReason, actor, ""); err != nil {
			return HandleErrorRespectJSON("closing issue: %v", err)
		}

		fmt.Printf("%s Issue %s dismissed.\n", ui.RenderPass("✔"), resolvedID)
		return nil
	},
}

// human stats command
var humanStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show summary statistics for human-needed issuegraph",
	Long: `Display summary statistics for human-needed issuegraph.

Shows counts for total, pending (open), responded (closed without dismiss),
and dismissed issuegraph.

Example:
  issuegraph human stats`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		evt := metrics.NewCommandEvent("human-stats")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		// The stats universe is `human list --status=all`: every human-labeled
		// issue, every status, every type.
		issues, err := humanIssues(rootCtx, "all")
		if err != nil {
			return HandleErrorRespectJSON("getting human issue stats: %v", err)
		}

		printHumanStats(issues)
		return nil
	},
}

// dismissedCloseReason prefixes the close reason `human dismiss` writes;
// printHumanStats classifies dismissed issuegraph by this same prefix so the
// writer and the classifier cannot drift apart.
const dismissedCloseReason = "Dismissed"

func printHumanStats(issues []*types.Issue) {
	total := len(issues)
	pending := 0
	closed := 0
	dismissed := 0

	for _, issue := range issues {
		switch issue.Status {
		case "closed":
			closed++
			if strings.HasPrefix(issue.CloseReason, dismissedCloseReason) {
				dismissed++
			}
		default:
			// All non-closed statuses (open, in_progress, blocked, hooked, etc.) are pending
			pending++
		}
	}

	responded := closed - dismissed

	fmt.Printf("\n%s\n", ui.RenderBold("Human IssueGraph Stats"))
	fmt.Println()
	fmt.Printf("  Total:      %d\n", total)
	fmt.Printf("  Pending:    %d\n", pending)
	fmt.Printf("  Responded:  %d\n", responded)
	fmt.Printf("  Dismissed:  %d\n", dismissed)
	fmt.Println()
}

// printCmd prints a command with consistent formatting
func printCmd(cmd, description string) {
	fmt.Printf("  %-20s %s\n", ui.RenderCommand(cmd), description)
}

func init() {
	// Add subcommands to humanCmd
	humanCmd.AddCommand(humanListCmd)
	humanCmd.AddCommand(humanRespondCmd)
	humanCmd.AddCommand(humanDismissCmd)
	humanCmd.AddCommand(humanStatsCmd)

	// Add flags for subcommands
	humanListCmd.Flags().StringP("status", "s", "", "Filter by status (open, closed, etc.; comma-separated for multiple, 'all' for every status)")
	humanRespondCmd.Flags().StringP("response", "r", "", "Response text")
	registerTextSourceFlags(humanRespondCmd, "response text", "response")
	humanDismissCmd.Flags().StringP("reason", "", "", "Reason for dismissal (optional)")

	// Register with root command
	rootCmd.AddCommand(humanCmd)
}
