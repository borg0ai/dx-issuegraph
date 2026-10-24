---
title: "issuegraph gc"
description: "Full lifecycle garbage collection for standalone IssueGraph databases."
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

Examples:
  issuegraph gc                              # Full GC with defaults (90 day decay)
  issuegraph gc --dry-run                    # Preview what would happen
  issuegraph gc --older-than 30              # Decay issues closed 30+ days ago
  issuegraph gc --skip-decay                 # Skip issue deletion, just compact+GC
  issuegraph gc --skip-dolt                  # Skip Dolt GC, just decay+compact
  issuegraph gc --force                      # Skip confirmation prompt

```
issuegraph gc [flags]
```

**Flags:**

```
      --dry-run          Preview without making changes
  -f, --force            Skip confirmation prompts
      --older-than int   Delete closed issues older than N days (default 90)
      --skip-decay       Skip issue deletion phase
      --skip-dolt        Skip Dolt garbage collection phase
```
