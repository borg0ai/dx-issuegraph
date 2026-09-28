---
title: Dependencies and Gates
description: Ordering work with blocking and non-blocking dependencies, and gates that wait on PRs, CI, or timers
---

IssueGraph includes a full dependency system for ordering work and a gate system
for bridging external conditions (PR merges, CI runs, timers) into the
dependency graph.

## Adding Dependencies

```bash
# issue-2 depends on issue-1 (issue-1 blocks issue-2)
issuegraph dep add issue-2 issue-1

# Shorthand: issue-1 blocks issue-2
issuegraph dep issue-1 --blocks issue-2

# Alternative flags (equivalent)
issuegraph dep add issue-2 --blocked-by issue-1
issuegraph dep add issue-2 --depends-on issue-1
```

When issue-1 is open, issue-2 won't appear in `issuegraph ready`. Once issue-1
is closed, issue-2 unblocks automatically.

## Removing Dependencies

```bash
issuegraph dep remove issue-2 issue-1
issuegraph dep rm issue-2 issue-1        # alias
```

## Dependency Types

Dependencies have a type that determines whether they block work.

**Blocking types** (affect `issuegraph ready`):

| Type | Meaning | Example |
|------|---------|---------|
| `blocks` (default) | B cannot start until A closes | Task ordering |
| `parent-child` | Children blocked when parent blocked | Epic hierarchies |
| `conditional-blocks` | B runs only if A fails | Error handling paths |
| `waits-for` | B waits for all of A's children | Fanout aggregation |

**Non-blocking types** (graph annotations only):

| Type | Meaning |
|------|---------|
| `related` | Informational link |
| `tracks` | Tracks progress of another issue |
| `discovered-from` | Found during work on another issue |
| `caused-by` | Root cause link |
| `validates` | Test or verification link |
| `supersedes` | Replaced by a different issue (`issuegraph supersede old --with new`) |

Specify with `--type`:

```bash
issuegraph dep add issue-2 issue-1 --type tracks
issuegraph dep add issue-2 issue-1 --type caused-by
```

## Finding Ready Work

`issuegraph ready` shows issues with no open blocking dependencies:

```bash
issuegraph ready
```

Output:
```
📋 Ready work (1 issues with no blockers):

1. [P1] bd-a1b2: Set up database
```

An issue is ready when ALL of its blocking dependencies are closed.

```bash
# Filter ready work
issuegraph ready --priority 1              # By priority
issuegraph ready --label backend           # By label
issuegraph ready --assignee alice          # By assignee
issuegraph ready --unassigned              # Unassigned only
issuegraph ready --type task               # By issue type
issuegraph ready --sort oldest             # Oldest first
```

## Viewing Blocked Issues

```bash
issuegraph blocked
```

Shows every blocked issue and what blocks it. Use after closing an issue
to see what just unblocked.

## Visualizing Dependencies

### Dependency Tree

```bash
issuegraph dep tree issue-id                    # What does this issue depend on?
issuegraph dep tree issue-id --direction=up     # What depends on this issue?
issuegraph dep tree issue-id --direction=both   # Both directions
issuegraph dep tree issue-id --status=open      # Only open issues
issuegraph dep tree issue-id --max-depth=3      # Limit depth
issuegraph dep tree issue-id --format=mermaid   # Mermaid.js output
```

### Dependency Graph

```bash
issuegraph graph issue-id                       # Single issue DAG
issuegraph graph --all                          # All open issues

# Output formats
issuegraph graph --compact issue-id             # One line per issue
issuegraph graph --box issue-id                 # ASCII boxes with layers
issuegraph graph --dot issue-id | dot -Tsvg > graph.svg   # Graphviz
issuegraph graph --html issue-id > graph.html   # Interactive D3.js
```

The graph organizes issues into layers:
- **Layer 0**: No dependencies (can start immediately)
- **Layer 1**: Depends on layer 0
- **Higher layers**: Depend on lower layers
- **Same layer**: Can run in parallel

### Dependency List

```bash
issuegraph dep list issue-id                    # What does this depend on?
issuegraph dep list issue-id --direction=up     # What depends on this?
issuegraph dep list issue-id --type=tracks      # Filter by type
```

### Cycle Detection

```bash
issuegraph dep cycles
```

IssueGraph also rejects cycles at write time — `issuegraph dep add` checks for
cycles before committing.

## Cross-Repo Dependencies

Dependencies can reference issues in other issuegraph rigs:

```bash
issuegraph dep add local-issue external:other-project:checkout-ready
```

Configure the project path under `external_projects`, then label a target
project issue `provides:checkout-ready`. The dependency blocks until an issue
with that label is closed. `issuegraph ready`, `issuegraph blocked`, and `issuegraph dep tree` resolve
the capability at query time; an unconfigured or unavailable project remains
blocking.

## Gates

Gates are special issues that block dependent work until an external
condition is met. They bridge the gap between issuegraph (which tracks work)
and external systems (which track code, CI, or time).

### The Problem Gates Solve

When you use Dolt (server or embedded), issue state is decoupled from
code state. Closing an issuegraph issue means "work is done" but the code
may still be on a feature branch, waiting for PR review:

```
issue-1: closed in issuegraph     (work done)
PR #42:  open on GitHub      (code not yet on main)
issue-2: blocked by issue-1  (should it start?)
```

With file-based storage (JSONL), issue updates land atomically with
code in the same commit. With Dolt, they don't. Gates solve this by
making the dependency wait for the external condition — not just the
issuegraph issue status.

### Gate Types

| Type | Condition | Auto-Resolution |
|------|-----------|-----------------|
| `gh:pr` | PR merged | `gh pr view` returns MERGED |
| `gh:run` | CI passes | `gh run view` returns completed + success |
| `timer` | Time elapsed | Current time exceeds timeout |
| `bead` | Cross-rig issue closed | Remote bead status checked |
| `human` | Manual approval | `issuegraph gate resolve <id>` |

### Creating Gates

```bash
# Wait for PR #42 to merge
issuegraph create --type=gate --title="Wait for PR #42" \
  --await-type=gh:pr --await-id=42

# Wait for CI run
issuegraph create --type=gate --title="Wait for CI" \
  --await-type=gh:run --await-id=12345

# Wait 30 minutes
issuegraph create --type=gate --title="Cooldown" \
  --await-type=timer --await-id=30m

# Wait for a cross-rig bead to close
issuegraph create --type=gate --title="Wait for upstream fix" \
  --await-type=bead --await-id=other-rig:issue-id

# Manual approval gate
issuegraph create --type=gate --title="Deploy approval"
```

### Wiring Gates into Dependencies

A gate is an issue. Wire it into the dependency graph like any other:

```bash
# issue-2 waits for the gate (which waits for PR #42)
issuegraph dep add issue-2 <gate-id>
```

### Checking Gates

`issuegraph gate check` evaluates all open gates and closes resolved ones:

```bash
issuegraph gate check                    # Check all gates
issuegraph gate check --type=gh:pr       # Only PR gates
issuegraph gate check --type=gh:run      # Only CI gates
issuegraph gate check --type=timer       # Only timers
issuegraph gate check --dry-run          # Preview without changes
issuegraph gate check --escalate         # Escalate failed gates
```

Escalation marks gates whose conditions failed (e.g., PR closed without
merge, CI run failed) so they surface for attention.

### Listing and Inspecting Gates

```bash
issuegraph gate list                     # Open gates
issuegraph gate list --all               # Including closed
issuegraph gate show <gate-id>           # Full details
```

### Manual Resolution

For `human` gates or overrides:

```bash
issuegraph gate resolve <gate-id> --reason "Approved by team lead"
```

### Discovering CI Run IDs

When you create a `gh:run` gate before the run starts, `issuegraph gate discover`
matches gates to GitHub Actions runs using heuristics (commit SHA, branch,
timing):

```bash
issuegraph gate discover                 # Auto-match gates to runs
issuegraph gate discover --dry-run       # Preview matches
issuegraph gate discover --branch main   # Filter by branch
```

### Automating Gate Checks

Run `issuegraph gate check` periodically to auto-close resolved gates:

- **CI step**: Add to your GitHub Actions workflow
- **Cron**: `*/5 * * * * cd /path/to/repo && issuegraph gate check`
- **Agent hook**: Run at session start or after PR operations

## Recipes

### PR Merge Gate (Common)

Agent A finishes work, opens PR, creates a gate so Agent B waits for merge:

```bash
# Agent A
issuegraph update issue-1 --status=in_progress
# ... write code, open PR #42 ...
issuegraph create --type=gate --title="Wait for PR #42" \
  --await-type=gh:pr --await-id=42
issuegraph dep add issue-2 <gate-id>
issuegraph close issue-1

# Agent B
issuegraph ready                         # issue-2 not shown (gate open)
# ... PR #42 merges ...
issuegraph gate check                    # gate closes
issuegraph ready                         # issue-2 appears
```

### CI Gate Before Deploy

```bash
issuegraph create --type=gate --title="CI green on main" \
  --await-type=gh:run --await-id=<run-id>
issuegraph dep add deploy-task <gate-id>
```

### Epic with Ordered Phases

```bash
issuegraph create "Auth System" -t epic
issuegraph create "Design" --parent <epic>
issuegraph create "Implement" --parent <epic>
issuegraph create "Test" --parent <epic>

issuegraph dep add <implement> <design>
issuegraph dep add <test> <implement>

issuegraph dep tree <epic>
issuegraph ready                         # Only "Design" is ready
```

## See Also

- [Quick Start](/getting-started/quickstart) — First steps with dependencies
- [Molecules](/workflows/molecules) — Molecule workflows using gates and dependencies
- [Agent Coordination](/multi-agent/coordination) — Cross-repo dependency patterns
- [Dolt Backend for IssueGraph](/architecture/dolt) — Dolt backend configuration
- [CLI Reference](/cli-reference/index) — Full command reference
