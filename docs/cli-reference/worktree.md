---
title: "issuegraph worktree"
description: "Manage git worktrees with proper issuegraph configuration."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc worktree`.

Manage git worktrees with proper issuegraph configuration.

Worktrees allow multiple working directories sharing the same git repository,
enabling parallel development (e.g., multiple agents or features).

Worktrees automatically share the same issuegraph database as the main repository
via git common directory discovery — no manual redirect configuration needed.

Examples:
  issuegraph worktree create feature-auth           # Create worktree
  issuegraph worktree create bugfix --branch fix-1  # Create with specific branch name
  issuegraph worktree list                          # List all worktrees
  issuegraph worktree remove feature-auth           # Remove worktree (with safety checks)
  issuegraph worktree info                          # Show info about current worktree

```
issuegraph worktree [flags]
```

## issuegraph worktree create

Create a git worktree for parallel development.

This command:
1. Creates a git worktree at ./&lt;name&gt; (or specified path)
2. Adds the worktree path to .gitignore (if inside repo root)

The worktree automatically shares the same issuegraph database as the main
repository via git common directory discovery — no redirect file needed.

Examples:
  issuegraph worktree create feature-auth           # Create at ./feature-auth
  issuegraph worktree create bugfix --branch fix-1  # Create with branch name
  issuegraph worktree create ../agents/worker-1     # Create at relative path

```
issuegraph worktree create <name> [--branch=<branch>] [flags]
```

**Flags:**

```
      --branch string   Branch name for the worktree (default: same as name)
```

## issuegraph worktree info

Show information about the current worktree.

If the current directory is in a git worktree, shows:
- Worktree path and name
- Branch
- IssueGraph configuration (redirect or main)
- Main repository location

Examples:
  issuegraph worktree info          # Show current worktree info
  issuegraph worktree info --json   # JSON output

```
issuegraph worktree info [flags]
```

## issuegraph worktree list

List all git worktrees and their issuegraph configuration state.

Shows each worktree with:
- Name (directory name)
- Path (full path)
- Branch
- IssueGraph state: "redirect" (uses shared db), "shared" (is main), "none" (no issuegraph)

Examples:
  issuegraph worktree list          # List all worktrees
  issuegraph worktree list --json   # JSON output

```
issuegraph worktree list [flags]
```

## issuegraph worktree remove

Remove a git worktree with safety checks.

Before removing, this command checks for:
- Uncommitted changes
- Unpushed commits
- Stashes

Use --force to skip safety checks (not recommended).

Examples:
  issuegraph worktree remove feature-auth         # Remove with safety checks
  issuegraph worktree remove feature-auth --force # Skip safety checks

```
issuegraph worktree remove <name> [flags]
```

**Flags:**

```
      --force   Skip safety checks
```
