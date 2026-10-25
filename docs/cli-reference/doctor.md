---
title: "issuegraph doctor"
description: "Check and fix issuegraph installation health (start here)"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc doctor`.

Sanity check the issuegraph installation for the current directory or specified path.

This command checks:
  - If .issuegraph/ directory exists
  - Database version and migration status
  - Schema compatibility (all required tables and columns present)
  - Whether using hash-based vs sequential IDs
  - If CLI version is current (checks GitHub releases)
  - If Claude plugin is current (when running in Claude Code)
  - File permissions
  - Circular dependencies
  - Git hooks (pre-commit, post-merge, pre-push)
  - .issuegraph/.gitignore up to date
  - Metadata.json version tracking (LastBdVersion field)

Storage Availability:
  Full diagnostics, --perf, --deep, --server, --migration, and
  --check=validate currently require Dolt server mode. Embedded Dolt
  supports --check=artifacts, --check=conventions, and
  --check=pollution. --check-health has a limited hook-health fallback.
  Unsupported combinations return a notice without changing storage.

Performance Mode (--perf):
  Run performance diagnostics on your database:
  - Times key operations (issuegraph ready, issuegraph list, issuegraph show, etc.)
  - Collects system info (OS, arch, database stats)
  - Generates CPU profile for analysis
  - Outputs shareable report for bug reports

Export Mode (--output):
  Save diagnostics to a JSON file for historical analysis and bug reporting.
  Includes timestamp and platform info for tracking intermittent issues.

Specific Check Mode (--check):
  Run a specific check in detail. Available checks:
  - artifacts: Detect and optionally clean issuegraph classic artifacts
    (stale JSONL, SQLite files, cruft .issuegraph dirs). Use with --clean.
  - conventions: Check for convention drift (lint warnings, stale
    issues, orphaned issues). Advisory only - warns, never blocks.
  - pollution: Detect and optionally clean test issues from database
  - validate: Run focused data-integrity checks (duplicates, orphaned
    deps, test pollution, git conflicts). Use with --fix to auto-repair.

Deep Validation Mode (--deep):
  Validate full graph integrity. May be slow on large databases.
  Additional checks:
  - Parent consistency: All parent-child deps point to existing issues
  - Dependency integrity: All deps reference valid issues
  - Epic completeness: Find epics ready to close (all children closed)
  - Agent issue integrity: Agent issuegraph have valid state values
  - Mail thread integrity: Thread IDs reference existing issues
  - Molecule integrity: Molecules have valid parent-child structures

Server Mode (--server):
  Run health checks for Dolt server mode connections (bd-dolt.2.3):
  - Server reachable: Can connect to configured host:port?
  - Dolt version: Is it a Dolt server (not vanilla MySQL)?
  - Database exists: Does the 'issuegraph' database exist?
  - Schema compatible: Can query issuegraph tables?
  - Connection pool: Pool health metrics

Legacy Dolt Migration Validation Mode (--migration):
  Retained for older SQLite-to-Dolt migration workflows and available only in
  Dolt server mode. It is not the migration path for removed backends
  (PostgreSQL, MySQL, SQLite); those fail closed with export/import guidance.
  Combine
  with --json for machine-parseable diagnostic output.

Agent Mode (--agent):
  Output diagnostics designed for AI agent consumption. Instead of terse
  pass/fail messages, each issue includes:
  - Observed state: what the system actually looks like
  - Expected state: what it should look like
  - Explanation: full prose context about the issue and why it matters
  - Commands: exact remediation commands to run
  - Source files: where in the codebase to investigate further
  - Severity: blocking (prevents operation), degraded (partial function),
    or advisory (informational only)
  ZFC-compliant: Go observes and reports, the agent decides and acts.
  Combine with --json for structured agent-facing output.

Suppressing Warnings:
  Suppress specific warnings by setting doctor.suppress.&lt;check-slug&gt; config:
    issuegraph config set doctor.suppress.pending-migrations true
    issuegraph config set doctor.suppress.git-hooks true
  Check names are converted to slugs: "Git Hooks" → "git-hooks".
  Only warnings are suppressed; errors and passing checks always show.
  To unsuppress: issuegraph config unset doctor.suppress.&lt;slug&gt;

Examples:
  issuegraph doctor              # Check current directory
  issuegraph doctor /path/to/repo # Check specific repository
  issuegraph doctor --json       # Machine-readable output
  issuegraph doctor --agent      # Agent-facing diagnostic output
  issuegraph doctor --agent --json  # Structured agent diagnostics (JSON)
  issuegraph doctor --fix        # Automatically fix issues (with confirmation)
  issuegraph doctor --fix --yes  # Automatically fix issues (no confirmation)
  issuegraph doctor --fix -i     # Confirm each fix individually
  issuegraph doctor --fix --fix-child-parent  # Also fix child→parent deps (opt-in)
  issuegraph doctor --fix --force # Force repair even when database can't be opened
  issuegraph doctor --fix --source=jsonl # Rebuild database from a JSONL export
  issuegraph doctor --dry-run    # Preview what --fix would do without making changes
  issuegraph doctor --perf       # Performance diagnostics
  issuegraph doctor --output diagnostics.json  # Export diagnostics to file
  issuegraph doctor --check=artifacts           # Show classic artifacts (JSONL, SQLite, cruft dirs)
  issuegraph doctor --check=artifacts --clean  # Delete safe-to-delete artifacts (with confirmation)
  issuegraph doctor --check=conventions        # Convention drift check (lint, stale, orphans)
  issuegraph doctor --check=pollution          # Show potential test issues
  issuegraph doctor --check=pollution --clean  # Delete test issues (with confirmation)
  issuegraph doctor --check=validate         # Data-integrity checks only
  issuegraph doctor --check=validate --fix   # Auto-fix data-integrity issues
  issuegraph doctor --deep             # Full graph integrity validation
  issuegraph doctor --server           # Dolt server mode health checks
  issuegraph doctor --migration=pre    # Legacy Dolt-server migration diagnostic
  issuegraph doctor --migration=post   # Legacy Dolt-server completion diagnostic
  issuegraph doctor --migration=pre --json  # Machine-parseable legacy diagnostic

```
issuegraph doctor [path] [flags]
```

**Flags:**

```
      --agent                                   Agent-facing diagnostic mode: rich context for AI agents (ZFC-compliant)
      --check string                            Run specific check in detail (e.g., 'pollution')
      --check-health                            Quick health check for git hooks (silent on success)
      --clean                                   For pollution check: delete detected test issues
      --deep                                    Validate full graph integrity
      --dry-run                                 Preview fixes without making changes
      --fix                                     Automatically fix issues where possible
      --fix-child-parent                        Remove child→parent dependencies (opt-in)
  -i, --interactive                             Confirm each fix individually
      --migration string                        Run legacy Dolt-server migration diagnostics: 'pre' or 'post'
      --orchestrator                            Running in orchestrator multi-workspace mode (routes.jsonl is expected, higher duplicate tolerance)
      --orchestrator-duplicates-threshold int   Duplicate tolerance threshold for orchestrator mode (wisps are ephemeral) (default 1000)
  -o, --output string                           Export diagnostics to JSON file
      --perf                                    Run performance diagnostics and generate CPU profile
      --server                                  Run Dolt server mode health checks (connectivity, version, schema)
  -v, --verbose                                 Show all checks (default shows only warnings/errors)
  -y, --yes                                     Skip confirmation prompt (for non-interactive use)
```
