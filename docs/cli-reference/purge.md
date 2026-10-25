---
title: "issuegraph purge"
description: "Delete closed ephemeral issuegraph to reclaim space"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc purge`.

Permanently delete closed ephemeral issuegraph and their associated data.

Closed ephemeral issuegraph (wisps, transient molecules) accumulate rapidly and
have no value once closed. This command removes them to reclaim storage.

Deletes: issues, dependencies, labels, events, and comments for matching issuegraph.
Skips: pinned issuegraph (protected).

To delete closed non-ephemeral issuegraph (regular tasks, features, bugs, etc.)
use `issuegraph prune` instead.

For full Dolt storage reclaim after deleting many rows, follow with `issuegraph flatten`
so history can be collapsed and old chunks can be garbage-collected.

EXAMPLES:
  issuegraph purge                           # Preview what would be purged
  issuegraph purge --force                   # Delete all closed ephemeral issuegraph
  issuegraph purge --older-than 7d --force   # Only purge items closed 7+ days ago
  issuegraph purge --pattern "*-wisp-*"      # Only purge matching ID pattern
  issuegraph purge --dry-run                 # Detailed preview with stats

```
issuegraph purge [flags]
```

**Flags:**

```
      --dry-run             Preview what would be purged with stats
  -f, --force               Actually purge (without this, shows preview)
      --older-than string   Only purge issuegraph closed more than N ago (e.g., 7d, 2w, 30)
      --pattern string      Only purge issuegraph matching ID glob pattern (e.g., *-wisp-*)
```
