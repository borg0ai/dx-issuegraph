package main

import (
	"fmt"
	"io"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/borg0ai/dx-issuegraph/internal/ui"
	"github.com/spf13/cobra"
)

const copilotInstructionsContent = `# GitHub Copilot Instructions

## Issue Tracking

This project uses **issuegraph (issuegraph)** for issue tracking.
Run ` + "`issuegraph prime`" + ` for workflow context, or install hooks (` + "`issuegraph hooks install`" + `) for auto-injection.

**Quick reference:**
- ` + "`issuegraph ready`" + ` - Find unblocked work
- ` + "`issuegraph create \"Title\" --type task --priority 2`" + ` - Create issue
- ` + "`issuegraph close <id>`" + ` - Complete work
- ` + "`issuegraph dolt push`" + ` - Push issuegraph to remote

For full workflow details: ` + "`issuegraph prime`" + ``

const agentsContent = `## Issue Tracking

This project uses **issuegraph (issuegraph)** for issue tracking.
Run ` + "`issuegraph prime`" + ` for workflow context, or install hooks (` + "`issuegraph hooks install`" + `) for auto-injection.

**Quick reference:**
- ` + "`issuegraph ready`" + ` - Find unblocked work
- ` + "`issuegraph create \"Title\" --type task --priority 2`" + ` - Create issue
- ` + "`issuegraph close <id>`" + ` - Complete work
- ` + "`issuegraph dolt push`" + ` - Push issuegraph to remote

For full workflow details: ` + "`issuegraph prime`" + ``

func renderOnboardInstructions(w io.Writer) error {
	writef := func(format string, args ...interface{}) error {
		_, err := fmt.Fprintf(w, format, args...)
		return err
	}
	writeln := func(text string) error {
		_, err := fmt.Fprintln(w, text)
		return err
	}
	writeBlank := func() error {
		_, err := fmt.Fprintln(w)
		return err
	}

	if err := writef("\n%s\n\n", ui.RenderBold("issuegraph Onboarding")); err != nil {
		return err
	}
	agentsFile := config.AgentsFile()
	if err := writeln("Add this minimal snippet to " + agentsFile + " (or create it):"); err != nil {
		return err
	}
	if err := writeBlank(); err != nil {
		return err
	}

	if err := writef("%s\n", ui.RenderAccent("--- BEGIN AGENTS.MD CONTENT ---")); err != nil {
		return err
	}
	if err := writeln(agentsContent); err != nil {
		return err
	}
	if err := writef("%s\n\n", ui.RenderAccent("--- END AGENTS.MD CONTENT ---")); err != nil {
		return err
	}

	if err := writef("%s\n", ui.RenderBold("For GitHub Copilot users:")); err != nil {
		return err
	}
	if err := writeln("Add the same content to .github/copilot-instructions.md"); err != nil {
		return err
	}
	if err := writeBlank(); err != nil {
		return err
	}

	if err := writef("%s\n", ui.RenderBold("How it works:")); err != nil {
		return err
	}
	if err := writef("   • %s provides dynamic workflow context (~80 lines)\n", ui.RenderAccent("issuegraph prime")); err != nil {
		return err
	}
	if err := writef("   • %s auto-injects issuegraph prime at session start\n", ui.RenderAccent("issuegraph hooks install")); err != nil {
		return err
	}
	if err := writeln("   • " + agentsFile + " only needs this minimal pointer, not full instructions"); err != nil {
		return err
	}
	if err := writeBlank(); err != nil {
		return err
	}

	if err := writef("%s\n\n", ui.RenderPass("This keeps "+agentsFile+" lean while issuegraph prime provides up-to-date workflow details.")); err != nil {
		return err
	}

	return nil
}

var onboardCmd = &cobra.Command{
	Use:     "onboard",
	GroupID: "setup",
	Short:   "Display minimal snippet for agent instructions file",
	Long: `Display a minimal snippet to add to your agent instructions file for issuegraph integration.

By default, the agent instructions file is AGENTS.md. Use 'issuegraph init --agents-file'
to configure a different filename (e.g. ISSUEGRAPH.md).

This outputs a small (~10 line) snippet that points to 'issuegraph prime' for full
workflow context. This is the same minimal profile that 'issuegraph init' generates
by default. This approach:

  • Keeps your agent file lean (doesn't bloat with instructions)
  • issuegraph prime provides dynamic, always-current workflow details
  • Hooks auto-inject issuegraph prime at session start

For agents or environments that do not auto-inject hook output, use
'issuegraph init --agents-profile=full' to embed the complete command reference.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		evt := metrics.NewCommandEvent("onboard")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		if err := renderOnboardInstructions(cmd.OutOrStdout()); err != nil {
			return HandleError("%v", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(onboardCmd)
}
