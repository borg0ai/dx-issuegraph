---
title: Git worktrees guide
description: IssueGraph database and snapshot behavior across linked Git worktrees.
---

Git worktrees in one checkout use the same local IssueGraph database. IssueGraph
refreshes the tracked `.issuegraph/` snapshot after writes. Separate clones have
separate databases and share snapshots only through ordinary Git operations.

```bash
cd project
issuegraph init
git worktree add ../project-feature feature-branch
cd ../project-feature
issuegraph ready
issuegraph create "Implement feature X" -t feature -p 1
```

Use Git to inspect and share snapshot changes:

```bash
git status --short .issuegraph/
git add .issuegraph/
git commit -m "Update issue snapshot"
```

`ISSUEGRAPH_DIR` can point several worktrees at an external local workspace. It
does not configure remote issue sync. `issuegraph dolt push`,
`issuegraph dolt pull`, `issuegraph sync`, and federation sync commands fail
before opening the database.

## Database not found in a worktree

Check that the main repository has been initialized and that the worktree belongs
to it:

```bash
git worktree list
cd /path/to/main/repo
ls -la .issuegraph
```

If it has no IssueGraph workspace, run `issuegraph init` from the main repository.

## See also

- [Protected branches](/reference/protected-branches)
- [Git integration](/reference/git-integration)
- [Sync setup](/getting-started/sync-setup)
