# Git Worktree Support

> Adapted from ACF beads skill

**v0.40+**: First-class worktree management via `issuegraph worktree` command.

## When to Use Worktrees

| Scenario | Worktree? | Why |
|----------|-----------|-----|
| Parallel agent work | Yes | Each agent gets isolated working directory |
| Long-running feature | Yes | Avoids stash/switch dance for interruptions |
| Quick branch switch | No | `git switch` is simpler |
| PR review isolation | Yes | Review without disturbing main work |

## Creating Worktrees

Normal Git worktrees work with issuegraph. Use `issuegraph worktree` when its convenience
features are useful:

```bash
# IssueGraph convenience command: creates the Git worktree and adds an in-repo path
# to .gitignore.
issuegraph worktree create .worktrees/{name} --branch feature/{name}
issuegraph worktree remove .worktrees/{name}

# Standard Git commands are also supported.
git worktree add -b feature/{name} .worktrees/{name}
git worktree remove .worktrees/{name}
```

`issuegraph worktree remove` adds safety checks for uncommitted changes and unpushed
commits. By default, both creation paths use the same shared issuegraph workspace.

## Architecture

By default, linked worktrees share the repository's `.beads/` workspace through
Git common directory discovery. They do not need per-worktree redirect files:

```
main-repo/
├── .git/                ← Shared Git directory
├── .beads/              ← Shared issuegraph config and local Dolt data
└── .worktrees/
    ├── feature-a/
    └── feature-b/
```

`issuegraph` uses the workspace's configured storage mode from every linked worktree;
worktree use does not force embedded mode.

Set `BEADS_DIR` to use an external issuegraph workspace instead. A worktree can also
use its own `.beads/` database explicitly; otherwise discovery falls back to
the shared workspace.

## Commands

```bash
# Create worktree with issuegraph support
issuegraph worktree create .worktrees/my-feature --branch feature/my-feature

# List worktrees
issuegraph worktree list

# Show info for the current worktree
cd .worktrees/my-feature
issuegraph worktree info

# Remove worktree cleanly
issuegraph worktree remove .worktrees/my-feature
```

## Debugging

When issuegraph commands behave unexpectedly in a worktree:

```bash
issuegraph where              # Shows the effective .beads workspace location
issuegraph doctor --deep      # Validates full graph integrity
```

## Protected Branch Workflows

Protected Git branches need no special issuegraph branch because issue data is
stored in Dolt under `refs/dolt/data`, separate from code branches:

```bash
# Choose one initialization path:
issuegraph init                            # Standard repository setup
# OR
issuegraph init --contributor              # OSS fork setup with contributor routing

issuegraph dolt pull                       # Pull shared issue data
issuegraph dolt push                       # Push shared issue data
```

No `--branch` flag or `.git/beads-worktrees/` directory is used. Keep using
your normal Git feature branches and worktrees for code changes.

## Multi-Clone Support

Multi-clone, multi-branch workflows:

- Hash-based IDs (`bd-abc`) eliminate collision across clones
- Each clone syncs through the configured Dolt remote with `issuegraph dolt pull` and
  `issuegraph dolt push`
- See [WORKTREES.md](https://github.com/gastownhall/beads/blob/main/docs/reference/worktrees.md) for comprehensive guide

## External References

- **Official Docs**: [github.com/gastownhall/beads/docs](https://github.com/gastownhall/beads/tree/main/docs)
- **Protected Branches**: [PROTECTED_BRANCHES.md](https://github.com/gastownhall/beads/blob/main/docs/reference/protected-branches.md)
- **Worktrees**: [WORKTREES.md](https://github.com/gastownhall/beads/blob/main/docs/reference/worktrees.md)
