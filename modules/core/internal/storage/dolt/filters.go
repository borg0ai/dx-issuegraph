package dolt

import (
	"github.com/borg0ai/dx-issuegraph/internal/storage/issueops"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// filterTables is a package-local alias for issueops.FilterTables.
type filterTables = issueops.FilterTables

var (
	issuesFilterTables = issueops.IssuesFilterTables
	wispsFilterTables  = issueops.WispsFilterTables
)

// buildIssueFilterClauses delegates to the shared issueops implementation.
func buildIssueFilterClauses(query string, filter types.IssueFilter, tables filterTables) ([]string, []interface{}, error) {
	return issueops.BuildIssueFilterClauses(query, filter, tables)
}

// looksLikeIssueID delegates to the shared issueops implementation.
func looksLikeIssueID(query string) bool {
	return issueops.LooksLikeIssueID(query)
}
