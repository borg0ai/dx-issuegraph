---
title: "issuegraph conflicts"
description: "Inspect and resolve live merge conflicts"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc conflicts`.

Inspect and resolve the merge conflicts sitting in the working set.

Conflicts appear when a pull or merge brought in changes that collide with
local ones and could not be settled automatically. These commands present them
per issue and per field, and resolve them without the raw dolt CLI.

Examples:
  issuegraph conflicts list                          # which tables and issues are conflicted
  issuegraph conflicts show                          # every conflicted row, field by field
  issuegraph conflicts show bd-1234                  # one issue
  issuegraph conflicts resolve bd-1234 --ours        # keep our side of one issue
  issuegraph conflicts resolve --all --theirs        # take their side of everything

```
issuegraph conflicts [command]
```

## issuegraph conflicts list

List tables and issues with live merge conflicts

```
issuegraph conflicts list [flags]
```

## issuegraph conflicts resolve

Resolve live merge conflicts, then conclude the merge with a commit.

Named issue IDs are resolved row by row, leaving every other conflicted row
alone. --all resolves whole tables at once (dolt's own table-level
resolution). The merge is committed only once NO conflicts remain, so a
partial resolution leaves the merge open for the next pass.

Row-by-row resolution requires the row to exist on both sides: when one side
deleted it, resolve that table wholesale or edit the row directly.

Examples:
  issuegraph conflicts resolve bd-1234 --ours              # keep our side of one issue
  issuegraph conflicts resolve bd-1234 bd-5678 --theirs    # take their side of two
  issuegraph conflicts resolve --all --ours                # every conflicted table
  issuegraph conflicts resolve --all --table config --theirs
  issuegraph conflicts resolve --conclude                  # commit an already-resolved merge

```
issuegraph conflicts resolve [<issue-id>...] [flags]
```

**Flags:**

```
      --all               Resolve whole tables instead of named issues
      --conclude          Commit a merge whose conflicts are already resolved
      --no-commit         Resolve without committing the merge
      --ours              Keep our side
      --strategy string   Resolution strategy: ours|theirs
      --table string      Table to resolve (default: issues)
      --theirs            Take their side
```

## issuegraph conflicts show

Show each conflicted row with its fields side by side.

Only fields where our side and their side disagree are shown; --all-fields
shows every column. Without an issue ID, every conflicted row of every
conflicted table is shown.

```
issuegraph conflicts show [<issue-id>] [flags]
```

**Flags:**

```
      --all-fields     Show every column, not just the fields that diverged
      --table string   Restrict to one conflicted table (default: all)
```
