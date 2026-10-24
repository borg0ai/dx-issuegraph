---
description: How to resolve merge conflicts in the issuegraph Dolt database
---

# Resolving IssueGraph Merge Conflicts

IssueGraph uses Dolt as its storage backend. Dolt handles merges natively using its built-in three-way merge, similar to git.

## 1. Check for Conflicts

```bash
issuegraph doctor
issuegraph dolt pull
```

If `issuegraph dolt pull` reports merge conflicts, Dolt will list the conflicting tables and rows.

## 2. Resolve Conflicts

Dolt provides SQL-based conflict resolution:

```bash
# View conflicts
issuegraph sql "SELECT * FROM dolt_conflicts"

# Resolve by accepting ours or theirs
issuegraph sql "CALL dolt_conflicts_resolve('--ours')"
# OR
issuegraph sql "CALL dolt_conflicts_resolve('--theirs')"
```

## 3. Verify and Complete

```bash
# Verify the resolution
issuegraph list --json | head

# Push the resolved state
issuegraph dolt push
```

## Legacy: JSONL Merge Conflicts

If you encounter merge conflicts in `.issuegraph/issues.jsonl` from a legacy setup, import the resolved file:

```bash
# Resolve the git conflict in the JSONL file manually, then:
issuegraph import -i .issuegraph/issues.jsonl
git add .issuegraph/issues.jsonl
git merge --continue
```
