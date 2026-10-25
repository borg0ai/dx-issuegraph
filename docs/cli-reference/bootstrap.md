---
title: "issuegraph bootstrap"
description: "Non-destructive database setup for fresh clones and recovery"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc bootstrap`.

Bootstrap sets up the issuegraph database without destroying existing data.
Unlike 'issuegraph init --force', bootstrap will never delete existing issues.

Bootstrap auto-detects the right action:
  • If sync.remote is configured: clones from the remote
  • If git origin has Dolt data (refs/dolt/data): clones from git and wires origin for future push/pull
  • If .issuegraph/backup/*.jsonl exists: restores from backup
  • If .issuegraph/issues.jsonl exists: imports from git-tracked JSONL
  • If no database exists: creates a fresh one
  • If database already exists: validates and reports status

If sync.remote points at a git repository, bootstrap verifies refs/dolt/data
before cloning. Bootstrap exits non-zero when it cannot set up a database.

This is the recommended command for:
  • Setting up issuegraph on a fresh clone
  • Recovering after moving to a new machine
  • Repairing a broken database configuration

Non-interactive mode (--non-interactive, --yes/-y, or ISSUEGRAPH_NON_INTERACTIVE=1):
  Skips the confirmation prompt before executing the bootstrap plan.
  Also auto-detected when stdin is not a terminal or CI=true is set.

Examples:
  issuegraph bootstrap              # Auto-detect and set up
  issuegraph bootstrap --dry-run    # Show what would be done
  issuegraph bootstrap --json       # Output plan as JSON
  issuegraph bootstrap --yes        # Skip confirmation prompt


```
issuegraph bootstrap [flags]
```

**Flags:**

```
      --dry-run           Show what would be done without doing it
      --non-interactive   Alias for --yes
  -y, --yes               Skip confirmation prompts (for CI/automation)
```
