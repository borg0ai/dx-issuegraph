---
title: "issuegraph migrate"
description: "Database migration and data transformation commands."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc migrate`.

Database migration and data transformation commands.

Without subcommand, checks and updates database metadata to current version.

Subcommands:
  hooks       Plan git hook migration to marker-managed format
  issues      Move issues between repositories
  schema      Apply pending schema migrations (idempotent)
  sync        Set up sync.branch workflow for multi-clone setups


```
issuegraph migrate [flags]
```

**Flags:**

```
      --dry-run          Show what would be done without making changes
      --inspect          Show migration plan and database state for AI agent analysis
      --json             Output migration statistics in JSON format
      --update-repo-id   Update repository ID (use after changing git remote)
      --yes              Auto-confirm prompts
```

## issuegraph migrate hooks

Analyze git hook files and sidecar artifacts for migration to marker-managed format.

Modes:
  --dry-run  Preview migration operations without changing files
  --apply    Apply migration operations

Examples:
  issuegraph migrate hooks --dry-run
  issuegraph migrate hooks --apply
  issuegraph migrate hooks --apply --yes
  issuegraph migrate hooks --dry-run --json

```
issuegraph migrate hooks [path] [flags]
```

**Flags:**

```
      --apply     Apply planned hook migration changes
      --dry-run   Show what would be done without making changes
      --json      Output in JSON format
      --yes       Skip confirmation prompt for --apply
```

## issuegraph migrate issues

Move issues from one source repository to another with filtering and dependency preservation.

This command updates the source_repo field for selected issues, allowing you to:
- Move contributor planning issues to upstream repository
- Reorganize issues across multi-phase repositories
- Consolidate issues from multiple repos

Examples:
  # Preview migration from planning repo to current repo
  issuegraph migrate-issues --from ~/.issuegraph-planning --to . --dry-run

  # Move all open P1 bugs
  issuegraph migrate-issues --from ~/repo1 --to ~/repo2 --priority 1 --type bug --status open

  # Move specific issues with their dependencies
  issuegraph migrate-issues --from . --to ~/archive --id issuegraph-abc --id issuegraph-xyz --include closure

  # Move issues with label filter
  issuegraph migrate-issues --from . --to ~/feature-work --label frontend --label urgent

```
issuegraph migrate issues [flags]
```

**Flags:**

```
      --dry-run            Show plan without making changes
      --from string        Source repository (required)
      --id strings         Specific issue IDs to migrate (can specify multiple)
      --ids-file string    File containing issue IDs (one per line)
      --include string     Include dependencies: none/upstream/downstream/closure (default "none")
      --label strings      Filter by labels (can specify multiple)
      --priority int       Filter by priority (0-4) (default -1)
      --status string      Filter by status (open/closed/all)
      --strict             Fail on orphaned dependencies or missing repos
      --to string          Destination repository (required)
      --type string        Filter by issue type (bug/feature/task/epic/chore/decision)
      --within-from-only   Only include dependencies from source repo (default true)
      --yes                Skip confirmation prompt
```

## issuegraph migrate schema

Apply pending schema migrations idempotently.

Schema migrations also run automatically on store open, so this subcommand
is typically a no-op. It exists to make migration explicit and observable
in CI, release gates, and recovery scenarios.

Example:
  issuegraph migrate schema
  issuegraph migrate schema --json

```
issuegraph migrate schema [flags]
```

**Flags:**

```
      --json   Output in JSON format
```

## issuegraph migrate sync

Configure separate branch workflow for multi-clone setups.

This sets the sync.branch config value so that issue data is committed
to a dedicated branch, keeping your main branch clean.

Example:
  issuegraph migrate sync issuegraph-sync

```
issuegraph migrate sync <branch> [flags]
```

**Flags:**

```
      --dry-run   Show what would be done without making changes
      --json      Output in JSON format
```
