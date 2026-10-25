---
title: "issuegraph orphans"
description: "Identify orphaned issues (referenced in commits but still open)"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc orphans`.

Identify orphaned issues - issues that are referenced in commit messages but remain open or in_progress in the database.

This helps identify work that has been implemented but not formally closed.

Examples:
  issuegraph orphans              # Show orphaned issues
  issuegraph orphans --json       # Machine-readable output
  issuegraph orphans --details    # Show full commit information
  issuegraph orphans --fix        # Close orphaned issues with confirmation
  issuegraph orphans --label theme:personal             # Only orphans with this label
  issuegraph orphans --label-any theme:personal,theme:ventures  # Orphans with either label

```
issuegraph orphans [flags]
```

**Flags:**

```
      --details             Show full commit information
  -f, --fix                 Close orphaned issues with confirmation
  -l, --label strings       Filter by labels (AND: must have ALL). Can combine with --label-any
      --label-any strings   Filter by labels (OR: must have AT LEAST ONE). Can combine with --label
```
