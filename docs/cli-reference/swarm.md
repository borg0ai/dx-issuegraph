---
title: "issuegraph swarm"
description: "Swarm management for structured epics"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc swarm`.

Swarm management commands for coordinating parallel work on epics.

A swarm is a structured body of work defined by an epic and its children,
with dependencies forming a DAG (directed acyclic graph) of work.

```
issuegraph swarm [command]
```

## issuegraph swarm create

Create a swarm molecule to orchestrate parallel work on an epic.

The swarm molecule:
- Links to the epic it orchestrates
- Has mol_type=swarm for discovery
- Specifies a coordinator (optional)
- Can be picked up by any coordinator agent

If given a single issue (not an epic), it will be auto-wrapped:
- Creates an epic with that issue as its only child
- Then creates the swarm molecule for that epic

Examples:
  issuegraph swarm create bd-epic-123                          # Create swarm for epic
  issuegraph swarm create bd-epic-123 --coordinator=observer/   # With specific coordinator
  issuegraph swarm create bd-task-456                          # Auto-wrap single issue

```
issuegraph swarm create [epic-id] [flags]
```

**Flags:**

```
      --coordinator string   Coordinator address (e.g., my-project/witness)
      --force                Create new swarm even if one already exists
```

## issuegraph swarm list

List all swarm molecules with their status.

Shows each swarm molecule with:
- Progress (completed/total issues)
- Active workers
- Epic ID and title

Examples:
  issuegraph swarm list         # List all swarms
  issuegraph swarm list --json  # Machine-readable output

```
issuegraph swarm list [flags]
```

## issuegraph swarm status

Show the current status of a swarm, computed from issuegraph.

Accepts either:
- An epic ID (shows status for that epic's children)
- A swarm molecule ID (follows the link to find the epic)

Displays issues grouped by state:
- Completed: Closed issues
- Active: Issues currently in_progress (with assignee)
- Ready: Open issues with all dependencies satisfied
- Blocked: Open issues waiting on dependencies

The status is COMPUTED from issuegraph, not stored separately.
If issuegraph changes, status changes.

Examples:
  issuegraph swarm status gt-epic-123       # Show swarm status by epic
  issuegraph swarm status gt-swarm-456      # Show status via swarm molecule
  issuegraph swarm status gt-epic-123 --json  # Machine-readable output

```
issuegraph swarm status [epic-or-swarm-id] [flags]
```

## issuegraph swarm validate

Validate an epic's structure to ensure it's ready for swarm execution.

Checks for:
- Correct dependency direction (requirement-based, not temporal)
- Orphaned issues (roots with no dependents)
- Missing dependencies (leaves that should depend on something)
- Cycles (impossible to resolve)
- Disconnected subgraphs

Reports:
- Ready fronts (waves of parallel work)
- Estimated worker-sessions
- Maximum parallelism
- Warnings for potential issues

Examples:
  issuegraph swarm validate gt-epic-123           # Validate epic structure
  issuegraph swarm validate gt-epic-123 --verbose # Include detailed issue graph

```
issuegraph swarm validate [epic-id] [flags]
```

**Flags:**

```
      --verbose   Include detailed issue graph in output
```
