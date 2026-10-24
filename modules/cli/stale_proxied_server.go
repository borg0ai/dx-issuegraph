package main

import (
	"context"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

func runStaleProxiedServer(ctx context.Context, filter types.StaleFilter) error {
	uw, err := openProxiedListUOW(ctx)
	if err != nil {
		return HandleError("%v", err)
	}
	defer uw.Close(ctx)

	issues, err := uw.IssueUseCase().GetStaleIssues(ctx, filter)
	if err != nil {
		return HandleErrorRespectJSON("%v", err)
	}

	return renderStale(issues, filter.Days)
}
