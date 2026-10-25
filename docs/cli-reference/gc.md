---
title: "issuegraph gc"
description: "Garbage collect: decay old issues, compact Dolt commits, run Dolt GC"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc gc`.

Full lifecycle garbage collection for standalone IssueGraph databases.

Runs three phases in sequence:
  1. DECAY   — Delete closed issues older than N days (default 90)
  2. COMPACT — Squash old Dolt commits into fewer commits (issuegraph compact)
  3. GC      — Run Dolt garbage collection to reclaim disk space

Each phase can be skipped individually. Use --dry-run to preview all phases
without making changes.

Phase 3 runs Dolt's default, generational GC: it only examines data written
since the last GC. Data that survived an earlier GC lives in the old generation
and is never revisited, so on a long-lived store the space freed by decay may
not be reclaimed. Use --full to collect all generations (slower on large
stores).

Examples:
  issuegraph gc                              # Full GC with defaults (90 day decay)
  issuegraph gc --dry-run                    # Preview what would happen
  issuegraph gc --older-than 30              # Decay issues closed 30+ days ago
  issuegraph gc --skip-decay                 # Skip issue deletion, just compact+GC
  issuegraph gc --skip-dolt                  # Skip Dolt GC, just decay+compact
  issuegraph gc --full                       # Collect all storage generations
  issuegraph gc --force                      # Skip confirmation prompt

```
issuegraph gc [flags]
```

**Flags:**

```
      --dry-run          Preview without making changes
  -f, --force            Skip confirmation prompts
      --full             Run a full Dolt GC (all generations; slower, reclaims space default passes cannot)
      --older-than int   Delete closed issues older than N days (default 90)
      --skip-decay       Skip issue deletion phase
      --skip-dolt        Skip Dolt garbage collection phase
```
