package main

import (
	"fmt"

	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/borg0ai/dx-issuegraph/internal/ui"
	"github.com/spf13/cobra"
)

var quickstartCmd = &cobra.Command{
	Use:     "quickstart",
	GroupID: "setup",
	Short:   "Quick start guide for issuegraph",
	Long:    `Display a quick start guide showing common issuegraph workflows and patterns.`,
	Run: func(cmd *cobra.Command, args []string) {
		evt := metrics.NewCommandEvent("quickstart")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		fmt.Printf("\n%s\n\n", ui.RenderBold("issuegraph - Dependency-Aware Issue Tracker"))
		fmt.Printf("Issues chained together like issuegraph.\n\n")

		fmt.Printf("%s\n", ui.RenderBold("GETTING STARTED"))
		fmt.Printf("  %s   Initialize issuegraph in your project (embedded Dolt, no server needed)\n", ui.RenderAccent("issuegraph init"))
		fmt.Printf("            Creates .issuegraph/ directory with project-specific Dolt database\n")
		fmt.Printf("            Auto-detects prefix from directory name (e.g., myapp-1, myapp-2)\n\n")

		fmt.Printf("  %s   Initialize with custom prefix\n", ui.RenderAccent("issuegraph init --prefix api"))
		fmt.Printf("            Issues will be named: api-<hash> (e.g., api-a3f2dd)\n\n")

		fmt.Printf("  %s   Initialize with external Dolt server (multi-writer)\n", ui.RenderAccent("issuegraph init --server"))
		fmt.Printf("            Connects to a running dolt sql-server for concurrent access\n\n")

		fmt.Printf("%s\n", ui.RenderBold("CREATING ISSUES"))
		fmt.Printf("  %s\n", ui.RenderAccent("issuegraph create \"Fix login bug\""))
		fmt.Printf("  %s\n", ui.RenderAccent("issuegraph create \"Add auth\" -p 0 -t feature"))
		fmt.Printf("  %s\n\n", ui.RenderAccent("issuegraph create \"Write tests\" -d \"Unit tests for auth\" --assignee alice"))

		fmt.Printf("%s\n", ui.RenderBold("VIEWING ISSUES"))
		fmt.Printf("  %s       List all issues\n", ui.RenderAccent("issuegraph list"))
		fmt.Printf("  %s  List by status\n", ui.RenderAccent("issuegraph list --status open"))
		fmt.Printf("  %s  List by priority (0-4, 0=highest)\n", ui.RenderAccent("issuegraph list --priority 0"))
		fmt.Printf("  %s       Show issue details\n\n", ui.RenderAccent("issuegraph show bd-1"))

		fmt.Printf("%s\n", ui.RenderBold("MANAGING DEPENDENCIES"))
		fmt.Printf("  %s     Add dependency (bd-2 blocks bd-1)\n", ui.RenderAccent("issuegraph dep add bd-1 bd-2"))
		fmt.Printf("  %s  Visualize dependency tree\n", ui.RenderAccent("issuegraph dep tree bd-1"))
		fmt.Printf("  %s      Detect circular dependencies\n\n", ui.RenderAccent("issuegraph dep cycles"))

		fmt.Printf("%s\n", ui.RenderBold("DEPENDENCY TYPES"))
		fmt.Printf("  %s  Task B must complete before task A\n", ui.RenderWarn("blocks"))
		fmt.Printf("  %s  Soft connection, doesn't block progress\n", ui.RenderWarn("related"))
		fmt.Printf("  %s  Epic/subtask hierarchical relationship\n", ui.RenderWarn("parent-child"))
		fmt.Printf("  %s  Auto-created when AI discovers related work\n\n", ui.RenderWarn("discovered-from"))

		fmt.Printf("%s\n", ui.RenderBold("READY WORK"))
		fmt.Printf("  %s       Show issues ready to work on\n", ui.RenderAccent("issuegraph ready"))
		fmt.Printf("            Ready = status is 'open' AND no blocking dependencies\n")
		fmt.Printf("            Perfect for agents to claim next work!\n\n")

		fmt.Printf("%s\n", ui.RenderBold("UPDATING ISSUES"))
		fmt.Printf("  %s\n", ui.RenderAccent("issuegraph update bd-1 --claim"))
		fmt.Printf("  %s\n", ui.RenderAccent("issuegraph update bd-1 --priority 0"))
		fmt.Printf("  %s\n\n", ui.RenderAccent("issuegraph update bd-1 --assignee bob"))

		fmt.Printf("%s\n", ui.RenderBold("CLOSING ISSUES"))
		fmt.Printf("  %s\n", ui.RenderAccent("issuegraph close bd-1"))
		fmt.Printf("  %s\n\n", ui.RenderAccent("issuegraph close bd-2 bd-3 --reason \"Fixed in PR #42\""))

		fmt.Printf("%s\n", ui.RenderBold("STORAGE"))
		fmt.Printf("  issuegraph uses Dolt, a version-controlled SQL database:\n")
		fmt.Printf("    %s  Embedded mode (default): in-process, zero config\n", ui.RenderPass("●"))
		fmt.Printf("              Data stored in %s\n", ui.RenderAccent(".issuegraph/embeddeddolt/"))
		fmt.Printf("    %s  Server mode (%s): multi-writer via dolt sql-server\n", ui.RenderPass("●"), ui.RenderAccent("issuegraph init --server"))
		fmt.Printf("              Data managed by external server\n\n")

		fmt.Printf("%s\n", ui.RenderBold("SYNC"))
		fmt.Printf("  Share issues with your team using Dolt remotes:\n")
		fmt.Printf("    %s  Verify remote (issuegraph init auto-wires git origin when present)\n", ui.RenderAccent("issuegraph dolt remote list"))
		fmt.Printf("    %s  Add remote if needed\n", ui.RenderAccent("issuegraph dolt remote add origin git+ssh://git@github.com/org/repo.git"))
		fmt.Printf("    %s              Push issues\n", ui.RenderAccent("issuegraph dolt push"))
		fmt.Printf("    %s              Pull from teammates\n", ui.RenderAccent("issuegraph dolt pull"))
		fmt.Printf("  Dolt handles sync natively with cell-level merge; JSONL is export, not sync\n\n")

		fmt.Printf("%s\n", ui.RenderBold("AGENT INTEGRATION"))
		fmt.Printf("  issuegraph is designed for AI-supervised workflows:\n")
		fmt.Printf("    %s Agents create issues when discovering new work\n", ui.RenderPass("●"))
		fmt.Printf("    %s %s shows unblocked work ready to claim\n", ui.RenderPass("●"), ui.RenderAccent("issuegraph ready"))
		fmt.Printf("    %s Use %s flags for programmatic parsing\n", ui.RenderPass("●"), ui.RenderAccent("--json"))
		fmt.Printf("    %s Dependencies prevent agents from duplicating effort\n\n", ui.RenderPass("●"))

		fmt.Printf("%s\n", ui.RenderPass("Ready to start!"))
		fmt.Printf("Run %s to create your first issue.\n\n", ui.RenderAccent("issuegraph create \"My first issue\""))
	},
}

func init() {
	rootCmd.AddCommand(quickstartCmd)
}
