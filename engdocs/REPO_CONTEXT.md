# Repository Context

This document explains how issuegraph resolves repository context when commands run from
different directories than where `.issuegraph/` lives.

## Problem

Git commands must run in the correct repository, but users may invoke `issuegraph` from:

- A different repository (using `ISSUEGRAPH_DIR` environment variable)
- A git worktree (separate working directory, shared `.issuegraph/`)
- A subdirectory within the repository

Without centralized handling, each command must implement its own path resolution,
leading to bugs when assumptions don't match reality.

## Solution: RepoContext API

The `RepoContext` API provides a single source of truth for repository resolution:

```go
import "github.com/borg0ai/dx-issuegraph/internal/issuegraph"

rc, err := issuegraph.GetRepoContext()
if err != nil {
    return err
}

// Run git in issuegraph repository (not CWD)
cmd := rc.GitCmd(ctx, "status")
output, err := cmd.Output()
```

## When to Use Each Method

| Method | Use Case | Example |
|--------|----------|---------|
| `GitCmd()` | Git commands for issuegraph operations | `git add .issuegraph/`, `git push` |
| `GitCmdCWD()` | Git commands for user's working repo | `git status` (show user's changes) |
| `RelPath()` | Convert absolute path to repo-relative | Display paths in output |

### GitCmd() vs GitCmdCWD()

The distinction matters when `ISSUEGRAPH_DIR` points to a different repository:

```go
rc, _ := issuegraph.GetRepoContext()

// GitCmd: runs in the issuegraph repository
// Use for: committing .issuegraph/, pushing/pulling issuegraph data
cmd := rc.GitCmd(ctx, "add", ".issuegraph/issues.jsonl")

// GitCmdCWD: runs in user's current repository
// Use for: checking user's uncommitted changes, status display
cmd := rc.GitCmdCWD(ctx, "status", "--porcelain")
```

## Scenarios

### Normal Repository

CWD is inside the repository containing `.issuegraph/`:

```
/project/
├── .issuegraph/
├── src/
└── README.md

$ cd /project/src
$ issuegraph dolt push
# GitCmd() runs in /project (correct)
```

### ISSUEGRAPH_DIR Redirect

User is in one repository but managing issuegraph in another:

```
$ cd /repo-a          # Has uncommitted changes
$ export ISSUEGRAPH_DIR=/repo-b/.issuegraph
$ issuegraph dolt push
# GitCmd() runs in /repo-b (correct, not /repo-a)
```

This pattern is common for:
- Fork contribution tracking (your tracker in separate repo)
- Shared team databases
- Monorepo setups

### Git Worktree

User is in a worktree but `.issuegraph/` lives in main repository:

```
/project/                    # Main repo
├── .issuegraph/
├── .worktrees/
│   └── feature-branch/      # Worktree (CWD)
└── src/

$ cd /project/.worktrees/feature-branch
$ issuegraph dolt push
# GitCmd() runs in /project (main repo, where .issuegraph lives)
```

### Combined: Worktree + Redirect

Both worktree and ISSUEGRAPH_DIR can be active simultaneously:

```
$ cd /repo-a/.worktrees/branch-x
$ export ISSUEGRAPH_DIR=/repo-b/.issuegraph
$ issuegraph dolt push
# GitCmd() runs in /repo-b (ISSUEGRAPH_DIR takes precedence)
```

## RepoContext Fields

| Field | Description |
|-------|-------------|
| `IssueGraphDir` | Actual `.issuegraph/` directory (after following redirects) |
| `RepoRoot` | Repository root containing `IssueGraphDir` |
| `CWDRepoRoot` | Repository root containing user's CWD (may differ) |
| `IsRedirected` | True if ISSUEGRAPH_DIR points to different repo than CWD |
| `IsWorktree` | True if CWD is in a git worktree |

## Security

### Git Hooks Disabled

`GitCmd()` disables git hooks and templates to prevent code execution in
potentially malicious repositories:

```go
cmd.Env = append(os.Environ(),
    "GIT_HOOKS_PATH=",   // Disable hooks
    "GIT_TEMPLATE_DIR=", // Disable templates
)
```

This protects against scenarios where `ISSUEGRAPH_DIR` points to an untrusted
repository that contains malicious `.git/hooks/` scripts.

### Path Boundary Validation

`GetRepoContext()` validates that `ISSUEGRAPH_DIR` does not point to sensitive
system directories:

- `/etc`, `/usr`, `/var`, `/root` (Unix system directories)
- `/System`, `/Library` (macOS system directories)
- Other users' home directories

Temporary directories (e.g., `/var/folders` on macOS) are explicitly allowed
for test environments.

## Server Mode Handling

### CLI vs Server Context

For CLI commands, `GetRepoContext()` caches the result via `sync.Once` because:
- CWD doesn't change during command execution
- ISSUEGRAPH_DIR doesn't change during command execution
- Repeated filesystem access would be wasteful

For the Dolt server (long-running process), this caching is inappropriate:
- User may create new worktrees
- ISSUEGRAPH_DIR may change via direnv
- Multiple workspaces may be active simultaneously

### Workspace-Specific API

The server uses `GetRepoContextForWorkspace()` for fresh resolution:

```go
// For server mode: fresh resolution per-operation (no caching)
rc, err := issuegraph.GetRepoContextForWorkspace(workspacePath)

// Validation hook for detecting stale contexts
if err := rc.Validate(); err != nil {
    // Context is stale, need fresh resolution
}
```

This function:
- Does NOT cache results
- Does NOT respect ISSUEGRAPH_DIR (workspace path is explicit)
- Resolves worktree relationships correctly
- Validates that paths still exist

## Migration Guide

### Before (scattered resolution)

```go
func doGitOperation(ctx context.Context) error {
    // Each function resolved paths differently
    issuegraphDir := issuegraph.FindIssueGraphDir()
    redirectInfo := issuegraph.GetRedirectInfo()
    var repoRoot string
    if redirectInfo.IsRedirected {
        repoRoot = filepath.Dir(issuegraphDir)
    } else {
        repoRoot = getRepoRootForWorktree(ctx)
    }
    cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "status")
    // ...
}
```

### After (centralized)

```go
func doGitOperation(ctx context.Context) error {
    rc, err := issuegraph.GetRepoContext()
    if err != nil {
        return err
    }
    cmd := rc.GitCmd(ctx, "status")
    // ...
}
```

### Key Changes for Contributors

1. **Replace direct exec.Command**: Use `rc.GitCmd()` or `rc.GitCmdCWD()`
2. **Remove manual path resolution**: RepoContext handles all scenarios
3. **Clear caches in tests**: Call `issuegraph.ResetCaches()` in test cleanup

## Testing

Tests use `issuegraph.ResetCaches()` to clear cached context between test cases:

```go
func TestSomething(t *testing.T) {
    t.Cleanup(func() {
        issuegraph.ResetCaches()
        git.ResetCaches()
    })
    // Test code...
}
```

## Related Documentation

- [WORKTREES.md](../docs/reference/worktrees.md) - Git worktree integration
- [ROUTING.md](../docs/multi-agent/routing.md) - Multi-repository routing
- [CONFIG.md](../docs/reference/configuration.md) - ISSUEGRAPH_DIR and environment variables

## Implementation Notes

- Result is cached via `sync.Once` for CLI efficiency
- CWD and ISSUEGRAPH_DIR don't change during command execution
- Uses `cmd.Dir` pattern (not `-C` flag) for Go-idiomatic execution
- Security mitigations implemented for git hooks and path traversal
