---
title: "issuegraph recompute-blocked"
description: "Recompute the denormalized is_blocked flag for every issue and wisp."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc recompute-blocked`.

Recompute the denormalized is_blocked flag for every issue and wisp.

is_blocked is derived from the dependency graph and maintained automatically by
local writes and by a post-pull recompute scoped to what the merge changed. If
that scoped recompute is skipped — a recompute that failed after its merge
committed, or a conflicted pull resolved by hand — the flag can go stale, and a
later pull that merges nothing will not refresh it (issuegraph-6dnrw.37). 'issuegraph ready'
trusts the flag, so stale values silently hide ready work or surface blocked
work.

This command runs the full recompute unconditionally and commits the result.
It is idempotent: on a consistent database it changes nothing. Works in both
embedded and server mode (unlike 'issuegraph doctor', which is server-mode only).

Examples:
  issuegraph recompute-blocked          # Repair stale is_blocked flags
  issuegraph recompute-blocked --json   # Machine-parseable &#123;"rows_corrected": N&#125;

```
issuegraph recompute-blocked [flags]
```
