---
title: "issuegraph compact"
description: "Squash Dolt commits older than N days into a single commit."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc compact`.

Squash Dolt commits older than N days into a single commit.

Recent commits (within the retention window) are preserved via cherry-pick.
This reduces Dolt storage overhead from auto-commit history while keeping
recent change tracking intact.

For semantic issue compaction (summarizing closed issues), use 'issuegraph admin compact'.
For full history squash, use 'issuegraph flatten'.

How it works:
  1. Identifies commits older than --days threshold
  2. Creates a squashed base commit from all old history
  3. Cherry-picks recent commits on top
  4. Swaps main branch to the compacted version
  5. Runs Dolt GC to reclaim space

Examples:
  issuegraph compact --dry-run               # Preview: show commit breakdown
  issuegraph compact --force                 # Squash commits older than 30 days
  issuegraph compact --days 7 --force        # Keep only last 7 days of history
  issuegraph compact --days 90 --force       # Conservative: squash 90+ day old commits

```
issuegraph compact [flags]
```

**Flags:**

```
      --days int   Keep commits newer than N days (default 30)
      --dry-run    Preview without making changes
  -f, --force      Confirm commit squash
```
