package main

import (
	"fmt"
	"os"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/types"
	"github.com/borg0ai/dx-issuegraph/internal/ui"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:     "ping",
	GroupID: "maint",
	Short:   "Check database connectivity",
	Long: `Lightweight health check that confirms issuegraph can reach its database.

Steps:
  1. Resolve the .issuegraph workspace
  2. Open the store (embedded or server)
  3. Run a trivial query (issue count)
  4. Report timing

Exit 0 on success, exit 1 on failure.

Examples:
  issuegraph ping              # Quick connectivity check
  issuegraph ping --json       # Structured output for automation`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if usesProxiedServer() {
			return runPingProxiedServer(rootCtx)
		}
		evt := metrics.NewCommandEvent("ping")
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		start := time.Now()

		issuegraphDir := issuegraph.FindIssueGraphDir()
		if issuegraphDir == "" {
			return pingFail(start, "no .issuegraph directory found")
		}
		resolveMs := time.Since(start).Milliseconds()

		st := getStore()
		if st == nil {
			return pingFail(start, "store not initialized")
		}
		if lm, ok := storage.UnwrapStore(st).(storage.LifecycleManager); ok && lm.IsClosed() {
			return pingFail(start, "store is closed")
		}
		storeMs := time.Since(start).Milliseconds()

		filter := types.IssueFilter{Limit: 1}
		_, err := st.SearchIssues(rootCtx, "", filter)
		if err != nil {
			return pingFail(start, fmt.Sprintf("query failed: %v", err))
		}
		totalMs := time.Since(start).Milliseconds()
		queryMs := totalMs - storeMs

		if jsonOutput {
			return outputJSON(map[string]interface{}{
				"status":     "ok",
				"resolve_ms": resolveMs,
				"store_ms":   storeMs - resolveMs,
				"query_ms":   queryMs,
				"total_ms":   totalMs,
			})
		}

		fmt.Fprintf(os.Stdout, "%s issuegraph ping: ok (%dms)\n", ui.RenderPass("✓"), totalMs)
		return nil
	},
}

func pingFail(start time.Time, reason string) error {
	totalMs := time.Since(start).Milliseconds()
	if jsonOutput {
		if jerr := outputJSON(map[string]interface{}{
			"status":   "error",
			"error":    reason,
			"total_ms": totalMs,
		}); jerr != nil {
			return jerr
		}
		return SilentExit()
	}
	return HandleError("issuegraph ping: %s (%dms)", reason, totalMs)
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
