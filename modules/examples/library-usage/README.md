# IssueGraph Library Usage Example

This example demonstrates using IssueGraph as a Go library in external projects (like VC).

## Why Use IssueGraph as a Library?

Instead of spawning `issuegraph` CLI processes:
- ✅ **Direct API access** - Call functions directly instead of parsing JSON output
- ✅ **Type safety** - Compile-time checking of types and interfaces
- ✅ **Performance** - No process spawn overhead
- ✅ **Transactions** - Access to database transactions for complex operations
- ✅ **Shared database** - Multiple components can use same database connection
- ✅ **Error handling** - Proper Go error types instead of parsing stderr

## Installation

In your Go project:

```bash
go get github.com/borg0ai/dx-issuegraph@latest
```

## Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/borg0ai/dx-issuegraph"
)

func main() {
    ctx := context.Background()

    // Find the workspace and open its configured Dolt implementation
    issuegraphDir := issuegraph.FindIssueGraphDir()
    store, err := issuegraph.OpenBestAvailable(ctx, issuegraphDir)
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    // Get ready work
    ready, err := store.GetReadyWork(ctx, issuegraph.WorkFilter{
        Status: issuegraph.StatusOpen,
        Limit: 10,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Process ready issues...
}
```

### Filtering ready work by status

`WorkFilter` offers two status fields:

- `Status` filters to exactly one status.
- `Statuses` filters to any of the given statuses (OR semantics) in a single
  query, so multi-status callers avoid one `GetReadyWork` round trip per
  status. Custom statuses participate like built-ins.

`Status` takes precedence: when it is set, `Statuses` is ignored. When both
are empty, the default of `('open', 'in_progress')` applies.

```go
    // One query for open OR blocked ready work
    ready, err := store.GetReadyWork(ctx, issuegraph.WorkFilter{
        Statuses: []issuegraph.Status{issuegraph.StatusOpen, issuegraph.StatusBlocked},
        Limit:    10,
    })
```

## Running This Example

```bash
# From this directory
cd examples/library-usage

# Make sure there's an IssueGraph database
issuegraph init --prefix demo

# Run the example
go run main.go
```

## Available Operations

The `issuegraph.Storage` interface provides:

### Issues
- `CreateIssue(ctx, issue, actor)` - Create a new issue
- `CreateIssues(ctx, issues, actor)` - Batch create issues
- `GetIssue(ctx, id)` - Get issue by ID
- `UpdateIssue(ctx, id, updates, actor)` - Update issue fields
- `UpdateIssueChecked(ctx, id, updates, actor, opts)` - Update with an optional
  optimistic-concurrency (CAS) precondition (`opts.ExpectedVersion`); see
  Concurrency below
- `CloseIssue(ctx, id, reason, actor)` - Close an issue
- `SearchIssues(ctx, query, filter)` - Search with filters

### Dependencies
- `AddDependency(ctx, dep, actor)` - Add dependency between issues
- `RemoveDependency(ctx, issueID, dependsOnID, actor)` - Remove dependency
- `GetDependencies(ctx, issueID)` - Get what this issue depends on
- `GetDependents(ctx, issueID)` - Get what depends on this issue
- `GetDependencyTree(ctx, issueID, maxDepth, showAllPaths)` - Visualize tree

### Labels
- `AddLabel(ctx, issueID, label, actor)` - Add label to issue
- `RemoveLabel(ctx, issueID, label, actor)` - Remove label
- `GetLabels(ctx, issueID)` - Get all labels for an issue
- `GetIssuesByLabel(ctx, label)` - Find issues with label

### Ready Work & Blocking
- `GetReadyWork(ctx, filter)` - Find issues with no blockers
- `GetBlockedIssues(ctx)` - Find blocked issues with blocker info
- `GetEpicsEligibleForClosure(ctx)` - Find completable epics

### Comments & Events
- `AddIssueComment(ctx, issueID, author, text)` - Add comment
- `GetIssueComments(ctx, issueID)` - Get all comments
- `GetEvents(ctx, issueID, limit)` - Get audit trail

### Statistics
- `GetStatistics(ctx)` - Get aggregate metrics

### Concurrency & Metadata
- `UpdateIssueChecked(ctx, id, updates, actor, opts)` - Like `UpdateIssue`, but
  when `opts.ExpectedVersion` is set the update proceeds only if the issue's
  current `RowVersion` still matches, else it refuses with
  `issuegraph.ErrVersionMismatch`. The version read and the write share one
  transaction (a true compare-and-swap). `RowVersion` is surfaced read-only on
  `Issue` reads.
- `MergeMetadata(ctx, issueID, key, value, actor)` - Atomically merge a single
  key into an issue's metadata JSON. Two concurrent merges of *different* keys
  both survive rather than clobbering each other.

> **Implementer note (interface change):** `UpdateIssueChecked` and
> `MergeMetadata` are now **required** methods on the `issuegraph.Storage` interface.
> Code that only *consumes* `issuegraph.Storage` needs no change, but any external
> type that *implements* `issuegraph.Storage` (a custom store, mock, or proxy) must
> add these two methods to compile. If you only need the base behavior,
> `UpdateIssueChecked` with a nil `opts.ExpectedVersion` is identical to
> `UpdateIssue`, and `MergeMetadata` can be layered over a read-modify-write of
> the issue's metadata.

## Types

All types are exported via the `issuegraph` package:

```go
// Core types
issuegraph.Issue
issuegraph.Status (Open, InProgress, Closed, Blocked)
issuegraph.IssueType (Bug, Feature, Task, Epic, Chore)
issuegraph.Priority (0-4)

// Relationships
issuegraph.Dependency
issuegraph.DependencyType (Blocks, Related, ParentChild, DiscoveredFrom)

// Metadata
issuegraph.Label
issuegraph.Comment
issuegraph.Event

// Queries
issuegraph.IssueFilter
issuegraph.WorkFilter
issuegraph.BlockedIssue
issuegraph.EpicStatus
issuegraph.Statistics
```

## VC Integration Example

For VC (VibeCoder), the integration would look like:

```go
// In VC's storage layer
type VCStorage struct {
    issuegraph issuegraph.Storage
}

func NewVCStorage(ctx context.Context, issuegraphDir string) (*VCStorage, error) {
    store, err := issuegraph.OpenBestAvailable(ctx, issuegraphDir)
    if err != nil {
        return nil, err
    }

    return &VCStorage{issuegraph: store}, nil
}

// Claim ready work for executor
func (s *VCStorage) ClaimWork(ctx context.Context, executorID string) (*issuegraph.Issue, error) {
    ready, err := s.issuegraph.GetReadyWork(ctx, issuegraph.WorkFilter{
        Status: issuegraph.StatusOpen,
        Limit: 1,
    })
    if err != nil {
        return nil, err
    }

    if len(ready) == 0 {
        return nil, nil // No work available
    }

    issue := ready[0]

    // Claim it
    updates := map[string]interface{}{
        "status": issuegraph.StatusInProgress,
        "assignee": executorID,
    }

    if err := s.issuegraph.UpdateIssue(ctx, issue.ID, updates, executorID); err != nil {
        return nil, err
    }

    return issue, nil
}
```

## Best Practices

1. **Context** - Always pass `context.Context` for cancellation support
2. **Actor** - Provide meaningful actor strings for audit trail
3. **Error handling** - Check all errors; database operations can fail
4. **Close** - Always `defer store.Close()` after opening
5. **Transactions** - For complex multi-step operations, consider using the underlying database connection directly

## See Also

- [issuegraph-example-extension-go](../issuegraph-example-extension-go/README.md) - Legacy SQLite extension example
- [issuegraph.go](../../issuegraph.go) - Public API source
- [internal/storage/storage.go](../../internal/storage/storage.go) - Storage interface
