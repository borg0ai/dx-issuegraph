---
title: Protected Branches
description: Why issuegraph needs no protected-branch workaround since Dolt stores issue data outside Git refs, plus team workflow and legacy sync-branch cleanup.
---

IssueGraph does not need a protected-branch workaround in current releases.

Issue data is stored in Dolt under `refs/dolt/data`, separate from normal Git
branches such as `main`. IssueGraph commands do not commit issue updates to your
current code branch, so GitHub, GitLab, and Bitbucket branch protection rules
continue to apply only to your code history.

## Current Workflow

Initialize issuegraph in the project:

```bash
issuegraph init
```

Commit the small tracked configuration files if your project policy requires
them:

```bash
git add .beads/.gitignore .beads/metadata.json .beads/config.yaml .gitignore
git commit -m "Initialize issuegraph issue tracker"
```

The local Dolt database directory remains gitignored. Sync issue data through a
Dolt remote:

```bash
issuegraph dolt pull
issuegraph dolt push
```

No `beads-sync` Git branch, protected-branch exception, or issuegraph-managed Git
worktree is required.

## Why Protected Branches Are Safe

Protected branches guard Git refs such as `refs/heads/main`. Dolt stores issuegraph
data in its own ref namespace. That means:

- `issuegraph create`, `issuegraph update`, and `issuegraph close` do not create commits on `main`.
- `issuegraph dolt push` pushes Dolt data, not a code branch.
- Normal code changes still go through your existing pull-request workflow.

## Team Usage

For a shared tracker:

```bash
issuegraph init --team
issuegraph dolt pull
issuegraph ready
issuegraph update <id> --claim
issuegraph dolt push
```

Pull before starting work and push before handing off so other clones see the
latest issue state.

## Legacy Sync-Branch Cleanup

Older issuegraph versions documented an experimental `sync.branch` workflow that
committed `.beads` changes to a branch such as `beads-sync` and used hidden Git
worktrees under `.git/beads-worktrees/`. That workflow has been removed.

If an old checkout still has sync-branch config, clear it:

```bash
issuegraph config set sync.branch ""
```

If stale hidden worktrees prevent branch checkout, remove them and prune Git's
worktree registry:

```bash
rm -rf .git/beads-worktrees
rm -rf .git/worktrees/beads-*
git worktree prune
```

If a remote `beads-sync` branch exists only for the removed workflow, archive or
delete it according to your repository policy after confirming all current issue
data has been synced through Dolt.

## Troubleshooting

### `issuegraph dolt push` Has No Remote

Add or inspect the Dolt remote:

```bash
issuegraph dolt remote list
issuegraph dolt remote add origin <remote-url>
issuegraph dolt push
```

### Conflicts During `issuegraph dolt pull`

Dolt reports database-level conflicts separately from Git branch conflicts. Use
the merge strategy or doctor guidance printed by the failed command:

```bash
issuegraph vc merge <branch> --strategy [ours|theirs]
issuegraph doctor --fix
```

### Stale Hooks Mention Legacy Sync Commands

Refresh generated hooks:

```bash
issuegraph hooks install
```

## See Also

- [Git Worktrees Guide](/reference/worktrees) - Git worktree behavior
- [Git Integration](/reference/git-integration) - general Git integration guide
- [Recovery Playbooks](/recovery/init-safety) - recovery playbooks
