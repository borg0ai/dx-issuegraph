package main

import (
	"fmt"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:     "upgrade",
	GroupID: "maint",
	Short:   "Check and manage issuegraph version upgrades",
	Long: `Commands for checking issuegraph version upgrades and reviewing changes.

The upgrade command helps you stay aware of issuegraph version changes:
  - issuegraph upgrade status: Check if issuegraph version changed since last use
  - issuegraph upgrade review: Show what's new since your last version
  - issuegraph upgrade ack: Acknowledge the current version

Version tracking is automatic - issuegraph updates metadata.json on every run.`,
}

var upgradeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if issuegraph version has changed",
	Long: `Check if issuegraph has been upgraded since you last used it.

This command uses the version tracking that happens automatically
at startup to detect if issuegraph was upgraded.

Examples:
  issuegraph upgrade status
  issuegraph upgrade status --json`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		evt := metrics.NewCommandEvent("upgrade-status")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		if jsonOutput {
			result := map[string]interface{}{
				"upgraded":        versionUpgradeDetected,
				"current_version": Version,
			}
			if versionUpgradeDetected {
				result["previous_version"] = previousVersion
				result["changes_available"] = len(getVersionsSince(previousVersion)) > 0
			}
			return outputJSON(result)
		}

		if versionUpgradeDetected {
			fmt.Printf("✨ issuegraph upgraded from %s to %s\n", displayVersion(previousVersion), displayVersion(Version))
			newVersions := getVersionsSince(previousVersion)
			if len(newVersions) > 0 {
				fmt.Printf("   %d version%s with changes available\n",
					len(newVersions),
					pluralize(len(newVersions)))
				fmt.Println()
				fmt.Println("Run 'issuegraph upgrade review' to see what changed")
			}
		} else if previousVersion == "" {
			fmt.Printf("issuegraph version: %s (first run or version tracking just enabled)\n", displayVersion(Version))
		} else {
			fmt.Printf("issuegraph version: %s (no upgrade detected)\n", displayVersion(Version))
		}
		return nil
	},
}

var upgradeReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review changes since last issuegraph version",
	Long: `Show what's new in issuegraph since the last version you used.

Unlike 'issuegraph info --whats-new' which shows the last 3 versions,
this command shows ALL changes since your specific last version.

If you're upgrading from an old version, you'll see the complete
changelog of everything that changed since then.

Examples:
  issuegraph upgrade review
  issuegraph upgrade review --json`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		evt := metrics.NewCommandEvent("upgrade-review")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		lastVersion := previousVersion

		if lastVersion == "" {
			fmt.Println("No previous version recorded")
			fmt.Println("Run 'issuegraph info --whats-new' to see recent changes")
			return nil
		}

		if !versionUpgradeDetected {
			fmt.Printf("You're already on %s (no upgrade detected)\n", displayVersion(Version))
			fmt.Println("Run 'issuegraph info --whats-new' to see recent changes")
			return nil
		}

		newVersions := getVersionsSince(lastVersion)

		if jsonOutput {
			return outputJSON(map[string]interface{}{
				"current_version":  Version,
				"previous_version": lastVersion,
				"new_versions":     newVersions,
			})
		}

		fmt.Printf("\n🔄 Upgraded from %s to %s\n", displayVersion(lastVersion), displayVersion(Version))
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()

		if len(newVersions) == 0 {
			fmt.Printf("%s is newer than %s but not in changelog\n", displayVersion(Version), displayVersion(lastVersion))
			fmt.Println("Run 'issuegraph info --whats-new' to see recent documented changes")
			return nil
		}

		for _, vc := range newVersions {
			versionMarker := ""
			if vc.Version == Version {
				versionMarker = " ← current"
			}

			fmt.Printf("## v%s (%s)%s\n\n", vc.Version, vc.Date, versionMarker)

			for _, change := range vc.Changes {
				fmt.Printf("  • %s\n", change)
			}
			fmt.Println()
		}

		fmt.Println("💡 Run 'issuegraph upgrade ack' to mark this version as seen")
		fmt.Println()
		return nil
	},
}

var upgradeAckCmd = &cobra.Command{
	Use:   "ack",
	Short: "Acknowledge the current issuegraph version",
	Long: `Mark the current issuegraph version as acknowledged.

This updates metadata.json to record that you've seen the current
version. Mainly useful after reviewing upgrade changes to suppress
future upgrade notifications.

Note: Version tracking happens automatically, so you don't need to
run this command unless you want to explicitly mark acknowledgement.

Examples:
  issuegraph upgrade ack
  issuegraph upgrade ack --json`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		evt := metrics.NewCommandEvent("upgrade-ack")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		issuegraphDir := issuegraph.FindIssueGraphDir()
		if issuegraphDir == "" {
			return HandleErrorWithHint(activeWorkspaceNotFoundMessage(), diagHint())
		}

		cfg, err := configfile.Load(issuegraphDir)
		if err != nil {
			return HandleError("loading metadata.json: %v", err)
		}
		if cfg == nil {
			cfg = configfile.DefaultConfig()
		}

		lastSeenVersion := cfg.LastBdVersion
		cfg.LastBdVersion = Version

		if err := cfg.Save(issuegraphDir); err != nil {
			return HandleError("saving metadata.json: %v", err)
		}

		upgradeAcknowledged = true
		versionUpgradeDetected = false

		if jsonOutput {
			return outputJSON(map[string]interface{}{
				"acknowledged":     true,
				"current_version":  Version,
				"previous_version": lastSeenVersion,
			})
		}

		if lastSeenVersion == Version {
			fmt.Printf("✓ Already on %s\n", displayVersion(Version))
		} else if lastSeenVersion == "" {
			fmt.Printf("✓ Acknowledged issuegraph %s\n", displayVersion(Version))
		} else {
			fmt.Printf("✓ Acknowledged upgrade from %s to %s\n", displayVersion(lastSeenVersion), displayVersion(Version))
		}
		return nil
	},
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func init() {
	upgradeCmd.AddCommand(upgradeStatusCmd)
	upgradeCmd.AddCommand(upgradeReviewCmd)
	upgradeCmd.AddCommand(upgradeAckCmd)
	rootCmd.AddCommand(upgradeCmd)
}
