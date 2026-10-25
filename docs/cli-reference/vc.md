---
title: "issuegraph vc"
description: "Version control operations"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc vc`.

Version control operations for the issuegraph database.

These commands provide git-like version control for your issue data, including branching, merging, and
viewing history.

Note: 'issuegraph history', 'issuegraph diff', and 'issuegraph branch' also work for quick access.
This subcommand provides additional operations like merge and commit.

```
issuegraph vc [command]
```

## issuegraph vc commit

Create a new Dolt commit with all current changes.

Examples:
  issuegraph vc commit -m "Added new feature issues"
  issuegraph vc commit --message "Fixed priority on several issues"
  echo "Multi-line message" | issuegraph vc commit --stdin

```
issuegraph vc commit [flags]
```

**Flags:**

```
  -m, --message string   Commit message
      --stdin            Read commit message from stdin
```

## issuegraph vc merge

Merge the specified branch into the current branch.

If there are merge conflicts, they will be reported. You can resolve
conflicts with --strategy.

Examples:
  issuegraph vc merge feature-xyz                    # Merge feature-xyz into current branch
  issuegraph vc merge feature-xyz --strategy ours    # Merge, preferring our changes on conflict
  issuegraph vc merge feature-xyz --strategy theirs  # Merge, preferring their changes on conflict

```
issuegraph vc merge <branch> [flags]
```

**Flags:**

```
      --strategy string   Conflict resolution strategy: 'ours' or 'theirs'
```

## issuegraph vc status

Show the current branch, commit hash, and any uncommitted changes.

Examples:
  issuegraph vc status

```
issuegraph vc status [flags]
```
