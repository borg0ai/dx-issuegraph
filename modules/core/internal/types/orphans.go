// Package types defines core data structures for the issuegraph issue tracker.
package types

import "context"

// IssueProvider abstracts issue storage for orphan detection.
// Implementations may be backed by Dolt or mocks.
type IssueProvider interface {
	// GetOpenIssues returns issues that are open or in_progress.
	// Should return empty slice (not error) if no issues exist.
	GetOpenIssues(ctx context.Context) ([]*Issue, error)

	// GetIssuePrefix returns the configured prefix (e.g., "issuegraph", "TEST").
	// Should return "issuegraph" as default if not configured.
	GetIssuePrefix() string
}
