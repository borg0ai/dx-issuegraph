package main

import (
	"github.com/spf13/cobra"

	"github.com/borg0ai/dx-issuegraph/internal/metrics"
)

var childrenCmd = &cobra.Command{
	Use:     "children <parent-id>",
	GroupID: "issues",
	Short:   "List child issuegraph of a parent",
	Long: `List all issuegraph that are children of the specified parent issue.

This is a convenience alias for 'issuegraph list --parent <id> --status all'.
Unlike plain 'issuegraph list', children includes closed issues by default,
since the primary use case is inspecting all work under a parent.

Examples:
  issuegraph children hq-abc123        # List all children of hq-abc123
  issuegraph children hq-abc123 --json # List children in JSON format
  issuegraph children hq-abc123 --pretty # Show children in tree format`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		evt := metrics.NewCommandEvent("children")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		parentID := args[0]
		pretty, _ := cmd.Flags().GetBool("pretty")

		_ = listCmd.Flags().Set("parent", parentID)
		defer func() { _ = listCmd.Flags().Set("parent", "") }()

		_ = listCmd.Flags().Set("status", "all")
		defer func() { _ = listCmd.Flags().Set("status", "") }()

		if pretty {
			_ = listCmd.Flags().Set("pretty", "true")
			defer func() { _ = listCmd.Flags().Set("pretty", "false") }()
		}

		// Reuse the shared, non-emitting list core so a single `issuegraph children`
		// records exactly one cli_command event ("children"), not also "list".
		return runListCore(listCmd, []string{})
	},
}

func init() {
	childrenCmd.Flags().Bool("pretty", false, "Show children in tree format")
	rootCmd.AddCommand(childrenCmd)
}
