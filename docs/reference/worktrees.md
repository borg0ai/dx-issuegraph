---
title: Git Worktrees Guide
description: Using issuegraph from Git worktrees, which share one .issuegraph workspace, plus external ISSUEGRAPH_DIR setups and legacy sync-branch cleanup.
---

IssueGraph works from normal Git worktrees without a separate sync branch. Current
issuegraph stores issue data in Dolt under `refs/dolt/data`, so issue sync is
separate from Git branch commits.

## Current Model

All worktrees in the same repository use the same issuegraph workspace unless you
override discovery with `ISSUEGRAPH_DIR`.

```
project/
├── .git/                 # Shared Git directory
├── .issuegraph/               # Shared issuegraph config and local Dolt data
├── main-worktree/
└── feature-worktree/
```

Key points:

- `issuegraph` discovers the repository's `.issuegraph` directory from linked worktrees.
- Issue changes are stored in Dolt, not committed to the current Git branch.
- Cross-clone sync uses `issuegraph dolt pull` and `issuegraph dolt push`.
- No `sync.branch` or issuegraph-managed Git worktree is required.

## Basic Usage

Initialize issuegraph once in the repository:

```bash
cd project
issuegraph init
```

Create linked worktrees normally:

```bash
git worktree add ../project-feature feature-branch
cd ../project-feature
issuegraph ready
issuegraph create "Implement feature X" -t feature -p 1
```

Sync issue data through the configured Dolt remote:

```bash
issuegraph dolt pull
issuegraph dolt push
```

## External IssueGraph Workspace

If you want a separate issue-tracker repository shared by many code worktrees,
point `ISSUEGRAPH_DIR` at that workspace:

```bash
export ISSUEGRAPH_DIR=~/project-issuegraph/.issuegraph

cd ~/project/main       && issuegraph list
cd ~/project/feature-1  && issuegraph list
cd ~/project/feature-2  && issuegraph list
```

With an external `ISSUEGRAPH_DIR`, `issuegraph dolt push` and `issuegraph dolt pull` target the
external issuegraph workspace, not the code repository.

## Hooks

Git hooks installed by issuegraph are worktree-aware. If hooks are stale or mention
removed legacy sync commands, refresh them:

```bash
issuegraph hooks install
```

## Legacy Cleanup

Older issuegraph versions had an experimental `sync.branch` workflow that created
hidden worktrees such as `.git/issuegraph-worktrees/<branch>/`. That workflow has
been removed.

If a legacy checkout cannot switch branches because an issuegraph-created worktree
still holds the branch, remove the stale worktree records:

```bash
rm -rf .git/issuegraph-worktrees
rm -rf .git/worktrees/issuegraph-*
git worktree prune
```

If old config still contains a sync branch, clear it:

```bash
issuegraph config set sync.branch ""
```

## Troubleshooting

### Database Not Found In A Worktree

Check that the main repository has a `.issuegraph` directory and that the worktree
belongs to that repository:

```bash
git worktree list
cd /path/to/main/repo
ls -la .issuegraph
```

If the repository has no issuegraph workspace yet, run `issuegraph init` from the main
repository.

### Multiple `.issuegraph` Directories

If a worktree has its own accidental `.issuegraph` directory, remove or archive the
extra copy after confirming it does not contain unique issue data. By default,
worktrees should share the repository workspace.

### Concurrent Writers

For ordinary single-user worktree use, run commands directly. For true
multi-writer workflows across machines or agents, sync frequently with
`issuegraph dolt pull` and `issuegraph dolt push`, and coordinate through the tracker to avoid
working the same issue concurrently.

## See Also

- [Protected Branches](/reference/protected-branches) - protected branch behavior
- [Git Integration](/reference/git-integration) - general Git integration guide
- [Multi-Repo Migration Guide](/multi-agent/multi-repo-migration) - multi-workspace patterns
