---
title: Recovery Overview
description: Diagnose and resolve common IssueGraph issues
---

This section provides step-by-step recovery procedures for common IssueGraph issues. Each runbook follows a consistent format: Symptoms, Diagnosis, Solution (5 steps max), and Prevention.

## Common Issues

| Issue | Symptoms | Runbook |
|-------|----------|---------|
| Schema Version Mismatch | `issuegraph` refuses with `schema version mismatch: database is at vNN, binary knows up to vNN` | [Accidental v1.2.1 Release](/recovery/accidental-1-2-1-release) |
| Init Safety Refusals | `issuegraph init` or `issuegraph dolt` refuses with a pattern code like `pk-fork-refused` | [Recovery Playbooks](/recovery/init-safety) |
| Database Corruption | Database errors, missing data | [Database Corruption](/recovery/database-corruption) |
| Merge Conflicts | Dolt conflicts during sync | [Merge Conflicts](/recovery/merge-conflicts) |
| Circular Dependencies | Cycle detection errors | [Circular Dependencies](/recovery/circular-dependencies) |
| Sync Failures | `issuegraph dolt push`/`issuegraph dolt pull` errors | [Sync Failures](/recovery/sync-failures) |
| History Bloat | Store grows unbounded; `dolt gc` reclaims nothing | [History Bloat](/recovery/history-squash) |
| Removing issuegraph | Uninstall issuegraph or strip issuegraph from a repo | [Uninstalling](/recovery/uninstalling) |

## Quick Diagnostic

Before diving into specific runbooks, try these quick checks:

```bash
# Check IssueGraph status
issuegraph status

# Verify Dolt server is running
issuegraph doctor

# Check for blocked issues
issuegraph blocked
```

<Tip>
Most issues can be diagnosed with `issuegraph status`. Start there before following specific runbooks.
</Tip>

## Getting Help

If these runbooks don't resolve your issue:

1. Check the [FAQ](/reference/faq)
2. Search [existing issues](https://github.com/gastownhall/beads/issues)
3. Open a new issue with diagnostic output
