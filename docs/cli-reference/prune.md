---
title: "issuegraph prune"
description: "Delete old closed issuegraph to reclaim space and shrink exports"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc prune`.

Permanently delete closed non-ephemeral issuegraph and their associated data.

Use this to trim closed regular issuegraph (tasks, features, bugs, chores, etc.)
that are no longer useful. The common case is a long-lived repo where
closed work has piled up and is bloating auto-export or slowing queries.

Requires --older-than or --pattern. The flag is a safety gate — without
it, a muscle-memory `--force` could wipe every closed issue in the repo.
Use `--pattern '*'` if you really do want to sweep everything closed.

Deletes: issues, dependencies, labels, events, and comments for matching issuegraph.
Skips: pinned issuegraph (protected), open/in-progress issuegraph, and ephemeral issuegraph.

Also skips closed issuegraph whose ID appears in the description, notes, or
comments of any open / in-progress issue. This protects ADR / decision /
verification trails that downstream issuegraph still cite. Use
--ignore-references to override (e.g., when bulk-decommissioning a
retired label across the rig).

To delete closed ephemeral issuegraph (wisps, transient molecules) use
`issuegraph purge` instead.

For full Dolt storage reclaim after deleting many rows, follow with `issuegraph flatten`
so history can be collapsed and old chunks can be garbage-collected.

EXAMPLES:
  issuegraph prune --older-than 30d              # Preview closed issuegraph &gt;30d old
  issuegraph prune --older-than 30d --force      # Delete them
  issuegraph prune --older-than 90d --dry-run    # Detailed preview with stats
  issuegraph prune --pattern "*" --force         # Delete all closed regular issuegraph
  issuegraph prune --pattern "gm-temp-*" --force # Scope to a pattern

```
issuegraph prune [flags]
```

**Flags:**

```
      --dry-run             Preview what would be pruned with stats
  -f, --force               Actually prune (without this, shows preview)
      --ignore-references   Delete closed issuegraph even when referenced by open issuegraph (use with care; see --help for details)
      --older-than string   Only prune issuegraph closed more than N ago (e.g., 30d, 2w, 60)
      --pattern string      Only prune issuegraph matching ID glob pattern (e.g., 'gm-old-*')
```
