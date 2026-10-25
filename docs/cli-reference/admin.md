---
title: "issuegraph admin"
description: "Administrative commands for database maintenance"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc admin`.

Administrative commands for issuegraph database maintenance.

These commands are for advanced users and should be used carefully:
  cleanup   Delete closed issues (issue lifecycle)
  compact   Compact old closed issues to save space (storage optimization)
  reset     Remove all issuegraph data and configuration (full reset)

For routine maintenance, prefer 'issuegraph doctor --fix' which handles common repairs
automatically. Use these admin commands for targeted database operations.

```
issuegraph admin [command]
```

## issuegraph admin cleanup

Delete closed issues to reduce database size.

This command permanently removes closed issues from the database.

NOTE: This command only manages issue lifecycle (closed -&gt; deleted). For general
health checks and automatic repairs, use 'issuegraph doctor --fix' instead.

By default, deletes ALL closed issues. Use --older-than to only delete
issues closed before a certain date.

EXAMPLES:
  issuegraph admin cleanup --force                          # Delete all closed issues
  issuegraph admin cleanup --older-than 30 --force          # Only issues closed 30+ days ago
  issuegraph admin cleanup --ephemeral --force              # Only closed wisps (transient molecules)
  issuegraph admin cleanup --dry-run                        # Preview what would be deleted

SAFETY:
- Requires --force flag to actually delete (unless --dry-run)
- Supports --cascade to delete dependents
- Shows preview of what will be deleted
- Use --json for programmatic output

SEE ALSO:
  issuegraph doctor --fix    Automatic health checks and repairs (recommended for routine maintenance)
  issuegraph admin compact   Compact old closed issues to save space

```
issuegraph admin cleanup [flags]
```

**Flags:**

```
      --cascade          Recursively delete all dependent issues
      --dry-run          Preview what would be deleted without making changes
      --ephemeral        Only delete closed wisps (transient molecules)
  -f, --force            Actually delete (without this flag, shows error)
      --older-than int   Only delete issues closed more than N days ago (0 = all closed issues)
```

## issuegraph admin compact

Compact old closed issues using semantic summarization.

Compaction reduces database size by summarizing closed issues that are no longer
actively referenced. This is permanent graceful decay - original content is discarded.

Modes:
  - Analyze: Export candidates for agent review (no API key needed)
  - Apply: Accept agent-provided summary (no API key needed)
  - Auto: AI-powered compaction (requires ANTHROPIC_API_KEY, MINIMAX_API_KEY, or ai.api_key)
  - Dolt: Run Dolt garbage collection (for Dolt-backend repositories)

Tiers:
  - Tier 1: Semantic compression (30 days closed, 70% reduction)
  - Tier 2: Ultra compression (90 days closed) - planned, not yet implemented

Dolt Garbage Collection:
  With auto-commit per mutation, Dolt commit history grows over time. Use
  --dolt to run Dolt garbage collection and reclaim disk space.

  --dolt: Run Dolt GC on .issuegraph/dolt directory to free disk space.
          This removes unreachable commits and compacts storage.

Examples:
  # Dolt garbage collection
  issuegraph compact --dolt                        # Run Dolt GC
  issuegraph compact --dolt --dry-run              # Preview without running GC

  # Agent-driven workflow (recommended)
  issuegraph compact --analyze --json              # Get candidates with full content
  issuegraph compact --apply --id bd-42 --summary summary.txt
  issuegraph compact --apply --id bd-42 --summary - &lt; summary.txt

  # AI-powered workflow
  issuegraph compact --auto --dry-run              # Preview candidates
  issuegraph compact --auto --all                  # Compact all eligible issues
  issuegraph compact --auto --id bd-42             # Compact specific issue
  MINIMAX_API_KEY=... issuegraph compact --auto --all

  # Statistics
  issuegraph compact --stats                       # Show statistics


```
issuegraph admin compact [flags]
```

**Flags:**

```
      --actor string     Actor name for audit trail (default "agent")
      --all              Process all candidates
      --analyze          Analyze mode: export candidates for agent review
      --apply            Apply mode: accept agent-provided summary
      --auto             Auto mode: AI-powered compaction
      --batch-size int   Issues per batch (default 10)
      --dolt             Dolt mode: run Dolt garbage collection on .issuegraph/dolt
      --dry-run          Preview without compacting
      --force            Force compact (bypass checks, requires --id)
      --id string        Compact specific issue
      --json             Output JSON format
      --limit int        Limit number of candidates (0 = no limit)
      --stats            Show compaction statistics
      --summary string   Path to summary file (use '-' for stdin)
      --tier int         Compaction tier (only tier 1 is implemented) (default 1)
      --workers int      Parallel workers (default 5)
```

## issuegraph admin reset

Reset issuegraph to an uninitialized state, removing all local data.

This command removes:
  - The .issuegraph directory (database, JSONL, config)
  - Git hooks installed by issuegraph
  - Sync branch worktrees

By default, shows what would be deleted (dry-run mode).
Use --force to actually perform the reset.

Examples:
  issuegraph reset              # Show what would be deleted
  issuegraph reset --force      # Actually delete everything

```
issuegraph admin reset [flags]
```

**Flags:**

```
      --force   Actually perform the reset (required)
```
