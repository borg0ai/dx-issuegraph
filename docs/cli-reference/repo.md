---
title: "issuegraph repo"
description: "Manage multiple repository configuration"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc repo`.

Configure and manage multiple repository support for multi-repo hydration.

Multi-repo support allows hydrating issues from multiple issuegraph repositories
into a single database for unified cross-repo issue tracking.

Configuration is stored in .issuegraph/config.yaml under the 'repos' section:

  repos:
    primary: "."
    additional:
      - ~/issuegraph-planning
      - ~/work-repo

Examples:
  issuegraph repo add ~/issuegraph-planning       # Add planning repo
  issuegraph repo add ../other-repo          # Add relative path repo
  issuegraph repo list                       # Show all configured repos
  issuegraph repo remove ~/issuegraph-planning    # Remove by path
  issuegraph repo sync                       # Sync from all configured repos

```
issuegraph repo [command]
```

## issuegraph repo add

Add a repository path to the repos.additional list in config.yaml.

The path should point to a directory containing a .issuegraph folder.
Paths can be absolute or relative (they are stored as-is).

This modifies .issuegraph/config.yaml, which is version-controlled and
shared across all clones of this repository.

```
issuegraph repo add <path> [flags]
```

**Flags:**

```
      --json   Output JSON
```

## issuegraph repo list

List all repositories configured in .issuegraph/config.yaml.

Shows the primary repository (always ".") and any additional
repositories configured for hydration.

```
issuegraph repo list [flags]
```

**Flags:**

```
      --json   Output JSON
```

## issuegraph repo remove

Remove a repository path from the repos.additional list in config.yaml.

The path must exactly match what was added (e.g., if you added "~/foo",
you must remove "~/foo", not "/home/user/foo").

This command also removes any previously-hydrated issues from the database
that came from the removed repository.

```
issuegraph repo remove <path> [flags]
```

**Flags:**

```
      --json   Output JSON
```

## issuegraph repo sync

Synchronize issues from all configured additional repositories.

Reads issues.jsonl from each additional repository and imports them into
the primary database with their original prefixes and source_repo set.
Uses mtime caching to skip repos whose JSONL hasn't changed.

Also triggers Dolt push/pull if a remote is configured.

```
issuegraph repo sync [flags]
```

**Flags:**

```
      --json      Output JSON
      --verbose   Show detailed sync progress
```
