package main

import (
	"context"

	"github.com/borg0ai/dx-issuegraph/internal/types"
	"github.com/borg0ai/dx-issuegraph/internal/workapi"
	"github.com/borg0ai/dx-issuegraph/modules/cli/doctor"
)

func runInfoProxiedServer(ctx context.Context, schemaFlag bool) error {
	uw, err := openProxiedListUOW(ctx)
	if err != nil {
		return HandleError("%v", err)
	}
	defer uw.Close(ctx)

	absDBPath := absoluteDBPath()

	info := map[string]interface{}{
		"database_path": absDBPath,
		"mode":          "proxied-server",
	}

	suppressHookWarning := false

	page, err := uw.IssueUseCase().SearchIssues(ctx, "", types.IssueFilter{})
	var issues []*types.Issue
	if err == nil {
		issues = page.Items
		info["issue_count"] = len(issues)
	}

	// THE SAME FILTER THE SETTINGS ROLE USES. `issuegraph info --json` serves
	// this map whole, and the issuegraph MCP server's get_schema_info tool
	// runs `issuegraph info --schema --json` and returns the parsed dict —
	// config included — so every memory key AND VALUE landed in the
	// transcript of any agent that asked a SCHEMA question. `issuegraph info`
	// is also the diagnostic people paste into bug reports.
	//
	// Unlike `issuegraph config show`, which an operator asks for by name to
	// see provenance, nothing here says "show me my memories".
	configMap, err := uw.ConfigUseCase().GetAllConfig(ctx)
	if err == nil {
		if filtered := workapi.FilterSettingsEnumeration(configMap); len(filtered) > 0 {
			info["config"] = filtered
		}
		// GH#6027: honor the same doctor.suppress.git-hooks issuegraph doctor does.
		suppressHookWarning = doctor.SuppressedChecksFromConfig(configMap)[doctor.CheckNameToSlug("Git Hooks")]
	}

	if schemaFlag {
		schemaVersion, err := uw.ConfigUseCase().GetLocalMetadata(ctx, "issuegraph_version")
		if err != nil {
			schemaVersion = "unknown"
		}
		prefix, _ := uw.ConfigUseCase().GetConfig(ctx, "issue_prefix")
		info["schema"] = buildInfoSchema(schemaVersion, prefix, issues)
	}

	return renderInfo(info, schemaFlag, absDBPath, suppressHookWarning)
}
