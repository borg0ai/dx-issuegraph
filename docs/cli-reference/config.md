---
title: "issuegraph config"
description: "Manage configuration settings"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc config`.

Manage configuration settings for external integrations and preferences.

Configuration is stored per-project in the issuegraph database and is version-control-friendly.

Common namespaces:
  - export.*          Auto-export settings (stored in config.yaml)
  - import.*          JSONL import settings (stored in config.yaml)
  - jira.*            Jira integration settings
  - linear.*          Linear integration settings
  - github.*          GitHub integration settings
  - gitlab.*          GitLab integration settings
  - ado.*             Azure DevOps integration settings
  - notion.*          Notion integration settings
  - custom.*          Custom integration settings
  - status.*          Issue status configuration
  - claim.*           Claim arbitration settings (pool-aware claiming)
  - doctor.suppress.* Suppress specific issuegraph doctor warnings (GH#1095)
  - lint.*            Additional per-type lint sections (stored in config.yaml)

Auto-Export (config.yaml):
  Optional JSONL export to .issuegraph/issues.jsonl after write commands (throttled).
  Useful for viewers (bv), interchange, and issue-level migration; not a backup.
  It is not cross-machine sync; use issuegraph dolt push/pull with a Dolt remote.
  Disabled by default. Enable only for integrations that need fresh JSONL.
  Auto-staging is separate and disabled by default.

  Keys:
    export.auto       Enable/disable auto-export (default: false)
    export.path       Output filename relative to .issuegraph/ (default: issues.jsonl)
    export.interval   Minimum time between exports (default: 60s)
    export.git-add    Auto-stage the export file (default: false)

Auto-Import (config.yaml):
  Reads .issuegraph/issues.jsonl by default when a JSONL import path is implied.
  Use a relative filename/path so the import stays within the project .issuegraph/
  directory and remains portable across machines.

  Keys:
    import.path       Input filename relative to .issuegraph/ (default: issues.jsonl)

Custom Status States:
  You can define custom status states for multi-step pipelines using the
  status.custom config key. Statuses should be comma-separated.

  Example:
    issuegraph config set status.custom "awaiting_review,awaiting_testing,awaiting_docs"

  This enables issues to use statuses like 'awaiting_review' in addition to
  the built-in statuses (open, in_progress, blocked, deferred, closed).

Claim Pools:
  A dispatcher can pre-assign issues to a pool pseudo-assignee (e.g.
  "fable-crew") and let any actor take them with --claim. List the pool
  aliases in the claim.pools config key, comma-separated:

    issuegraph config set claim.pools "fable-crew,night-crew"

  Issues assigned to a real actor (or to an alias not in the list) keep
  their anti-steal protection. Pool takes carry the normal lease; note
  that if a taker's lease expires, issuegraph reclaim returns the issue to the
  unassigned pool, not to the pool alias it was dispatched to.

Suppressing Doctor Warnings:
  Suppress specific issuegraph doctor warnings by check name slug:
    issuegraph config set doctor.suppress.pending-migrations true
    issuegraph config set doctor.suppress.git-hooks true
  Check names are converted to slugs: "Git Hooks" → "git-hooks".
  Only warnings are suppressed (errors and passing checks always show).
  To unsuppress: issuegraph config unset doctor.suppress.&lt;slug&gt;

Examples:
  issuegraph config set export.auto true                       # Enable auto-export for viewer integrations
  issuegraph config set export.path "issuegraph.jsonl"              # Custom export filename
  issuegraph config set import.path "issuegraph.jsonl"              # Custom import filename
  issuegraph config set export.git-add true                    # Also stage the export file
  issuegraph config set jira.url "https://company.atlassian.net"
  issuegraph config set jira.project "PROJ"
  issuegraph config set status.custom "awaiting_review,awaiting_testing"
  issuegraph config set claim.pools "fable-crew,night-crew"    # Pool aliases claimable by any actor
  issuegraph config set doctor.suppress.pending-migrations true
  issuegraph config set dolt.debug true                        # Enable Dolt sql-server debug mode (loglevel=debug, --prof cpu)
  issuegraph config set dolt.local-only true                   # Skip wiring a Dolt sync remote during issuegraph init
  issuegraph config set lint.sections.epic "Standards scorecard, Cost"   # Extra sections issuegraph lint requires for epics
  issuegraph config get export.auto
  issuegraph config list
  issuegraph config unset jira.url

```
issuegraph config [command]
```

## issuegraph config apply

Reconcile actual system state to match declared configuration.

Runs drift detection and then fixes any mismatches it finds:

  - hooks     Reinstall git hooks if missing or outdated
  - remote    Add/update Dolt origin remote to match federation.remote
  - server    Start Dolt server if dolt.shared-server is enabled

This command is idempotent — safe to run multiple times. Use --dry-run
to preview what would change without making modifications.

Examples:
  issuegraph config apply
  issuegraph config apply --dry-run
  issuegraph config apply --json

```
issuegraph config apply [flags]
```

**Flags:**

```
      --dry-run   Show what would change without making modifications
```

## issuegraph config drift

Detect drift between declared configuration and actual system state.

This is a read-only diagnostic that answers "is my environment consistent
with my config?" — no mutations are performed.

Checks:
  - hooks     Git hooks installed and up-to-date
  - remote    Dolt remote matches federation.remote config
  - server    Server state matches dolt.shared-server config

Exit codes:
  0  No drift detected (all checks ok/info/skipped)
  1  Drift detected (at least one check has status "drift")

Examples:
  issuegraph config drift
  issuegraph config drift --json

```
issuegraph config drift [flags]
```

## issuegraph config get

Get a configuration value

```
issuegraph config get <key> [flags]
```

## issuegraph config list

List all configuration

```
issuegraph config list [flags]
```

## issuegraph config set

Set a configuration value

```
issuegraph config set <key> <value> [flags]
```

**Flags:**

```
      --force-git-tracked   Allow writing secret keys to git-tracked config files (use with caution)
```

## issuegraph config set-many

Set multiple configuration values at once with a single auto-commit and auto-push.

Each argument must be in key=value format. All values are validated before
any writes occur. This is faster and less noisy than separate 'issuegraph config set'
calls, especially in CI.

Examples:
  issuegraph config set-many ado.state_map.open=New ado.state_map.closed=Closed
  issuegraph config set-many jira.url=https://example.atlassian.net jira.project=PROJ

```
issuegraph config set-many <key=value>... [flags]
```

**Flags:**

```
      --force-git-tracked   Allow writing secret keys to git-tracked config files (use with caution)
```

## issuegraph config show

Display a unified view of all effective configuration across all sources
with annotations showing where each value comes from.

Sources (by precedence for Viper-managed keys):
  - env          Environment variable (ISSUEGRAPH_* or ISSUEGRAPH_*)
  - config.yaml  Project config file (.issuegraph/config.yaml)
  - default      Built-in default value

Additional sources:
  - metadata     Connection settings from .issuegraph/metadata.json
  - database     Integration config stored in the Dolt database
  - git          Git config (e.g., issuegraph.role)

Examples:
  issuegraph config show
  issuegraph config show --json
  issuegraph config show --source config.yaml

```
issuegraph config show [flags]
```

**Flags:**

```
      --source string   Filter by source (e.g., config.yaml, env, default, metadata, database, git)
```

## issuegraph config unset

Delete a configuration value

```
issuegraph config unset <key> [flags]
```

## issuegraph config validate

Validate sync-related configuration settings.

Checks:
  - federation.sovereignty is valid (T1, T2, T3, T4, or empty)
  - federation.remote is set for Dolt sync
  - Remote URL format is valid (dolthub://, gs://, s3://, az://, file://)
  - routing.mode is valid (auto, maintainer, contributor, explicit)

	Examples:
	  issuegraph config validate
	  issuegraph config validate --json

```
issuegraph config validate [flags]
```
