---
title: Merge Conflicts
description: Resolve Dolt merge conflicts
---

This runbook helps you resolve merge conflicts that occur during Dolt sync operations.

## Symptoms

- `issuegraph dolt pull` fails with conflict errors
- Different issue states between clones

## Diagnosis

```bash
# Check database health
issuegraph doctor

# Preview what fixes would be applied
issuegraph doctor --dry-run
```

## Solution

**Step 1:** Back up current state
```bash
cp -r .beads .beads.backup
```

**Step 2:** Check for conflicts
```bash
issuegraph doctor
```

**Step 3:** Fix to reconcile
```bash
issuegraph doctor --fix
```

**Step 4:** Verify state
```bash
issuegraph list
issuegraph stats
```

**Step 5:** Push resolved state
```bash
issuegraph dolt push
```

## Prevention

- Sync before and after work sessions using `issuegraph dolt pull` / `issuegraph dolt push`
- Avoid concurrent modifications from multiple clones without the Dolt server running
