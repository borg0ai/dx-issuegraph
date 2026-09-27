---
title: Database Corruption
description: Recover from Dolt database corruption
---

This runbook helps you recover from database corruption in IssueGraph.

## Symptoms

- Error messages during `issuegraph` commands
- "database is locked" errors that persist
- Missing issues that should exist
- Inconsistent database state

## Diagnosis

```bash
# Check database integrity
issuegraph doctor

# Check Dolt server health
issuegraph dolt show
```

## Solution

**Step 1:** Stop the Dolt server
```bash
issuegraph dolt stop
```

**Step 2:** Back up current state
```bash
cp -r .beads .beads.backup
```

**Step 3:** Preview what doctor would fix
```bash
issuegraph doctor --dry-run
```

**Step 4:** Rebuild database
```bash
issuegraph doctor --fix
```

**Step 5:** Verify recovery
```bash
issuegraph doctor
issuegraph list
```

**Step 6:** Restart the Dolt server
```bash
dolt sql-server
```

## Prevention

- Let the Dolt server handle synchronization
- Use `issuegraph dolt stop` before system shutdown
- Run `issuegraph doctor` periodically to catch issues early
