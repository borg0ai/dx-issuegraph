---
title: Uninstalling
description: Remove issuegraph from a repository with issuegraph admin reset, uninstall git hooks, and delete the issuegraph binary after backing up issue data
---

This guide explains how to remove issuegraph from a repository or remove the `issuegraph`
binary from a machine.

## Before You Remove Data

Removing `.beads/` permanently deletes the local Dolt database. If the issue
history matters, make a Dolt-native backup first:

```bash
issuegraph backup init /path/to/beads-backup
issuegraph backup sync
```

For review, migration, or interoperability, you can also write an issue-table
export:

```bash
issuegraph export -o ~/beads-issues-$(date +%Y%m%d).jsonl
```

`issuegraph export` is not a complete restorable database backup. It does not preserve
Dolt branches, commit history, working-set state, or non-issue tables.

## Repository Reset

Use `issuegraph admin reset` from the repository root. It previews what will be
removed by default:

```bash
issuegraph admin reset
```

If the preview is correct, run:

```bash
issuegraph admin reset --force
```

This removes issuegraph-managed repository data such as:

- the `.beads/` directory
- git hooks that issuegraph installed in full
- legacy issuegraph sync worktrees under `.git/beads-worktrees/`

Reset works on whole hook files, not on sections. A hook of your own that
issuegraph injected a section into is left in place and reported, because deleting
the file would take your content with it. Remove the section from those with
`issuegraph hooks uninstall`.

## Remove Hooks Only

To keep issue data but remove git hooks:

```bash
issuegraph hooks uninstall
```

This is preferable to manually deleting hook files because issuegraph preserves
unrelated user hook content outside its managed hook markers.

## Manual Cleanup

Use manual cleanup only if `issuegraph admin reset` is unavailable or cannot run in
the repository.

Start by stopping a local Dolt server, if one is running:

```bash
issuegraph dolt stop 2>/dev/null || true
```

### Hooks: look before you delete

There is no batch command for this step, deliberately. `pre-commit`,
`prepare-commit-msg`, `post-merge`, `pre-push` and `post-checkout` are the
standard git hook names, not names issuegraph reserves, so any of them may be a hook
you wrote — and if you are reading this section, `issuegraph hooks uninstall` was not
available to tell the difference for you.

List which of them exist and what issuegraph left in them:

```bash
grep -l -e 'bd-hooks-version:' -e 'bd-shim' -e 'bd (beads)' -e 'BEGIN BEADS INTEGRATION' \
  .git/hooks/pre-commit .git/hooks/prepare-commit-msg .git/hooks/post-merge \
  .git/hooks/pre-push .git/hooks/post-checkout 2>/dev/null
```

Open each file that matched and decide from what is in it:

- A file whose **whole content** issuegraph generated carries a
  `# bd-hooks-version:`, `# bd-shim`, or `# bd (beads)` line and has nothing
  else in it but the shebang. Delete that one: `rm -f .git/hooks/<name>`.
- A file of yours with a `# --- BEGIN BEADS INTEGRATION ... ---` block in it is
  **your** file. Edit it: remove the lines from the `BEGIN` marker to the `END`
  marker, keep the rest, and leave the file in place. Comments of yours around
  the block count — a hook that is a header comment plus issuegraph's block is still
  yours to edit rather than delete.

Hooks the command did not list are yours regardless of what they mention.
Naming issuegraph in a comment, or calling `issuegraph` from a hook you composed, does not
make the file issuegraph's.

### The rest

```bash
# Remove the local issuegraph database and config.
rm -rf .beads

# Remove legacy sync-branch worktrees from older issuegraph versions.
rm -rf .git/beads-worktrees
git worktree prune
```

If `.gitattributes` contains only issuegraph merge-driver configuration, remove it.
If it contains other project entries, edit out only the issuegraph line.

If issuegraph-specific git config remains, remove it:

```bash
git config --unset beads.role 2>/dev/null || true
git config --unset core.hooksPath 2>/dev/null || true
git config --unset merge.beads.driver 2>/dev/null || true
git config --unset merge.beads.name 2>/dev/null || true
```

Do not skip `core.hooksPath`: if it is left set, git keeps looking for a
hooks directory that no longer exists, and issuegraph's post-checkout import can
recreate a `.beads/` workspace under the old prefix.

Check the value first, though — `core.hooksPath` is not issuegraph-only. If
`git config --get core.hooksPath` reports a directory that belongs to another
hook manager (husky's `.husky/_`, for example) rather than `.beads/hooks` or
`.beads-hooks`, leave it alone; unsetting it would disable that tool's hooks
too. `issuegraph doctor` applies the same rule and will not touch a hooks path it did
not set.

## Remove the `issuegraph` Binary

The CLI is a standalone binary. Remove it according to how it was installed:

```bash
# Homebrew
brew uninstall beads

# Go install
rm -f "$(which issuegraph)"

# Manual install location
rm -f /usr/local/bin/issuegraph
```

If you installed the MCP package separately, remove that package with the tool
you used to install it.

## Verify Removal

```bash
which issuegraph
test ! -e .beads
issuegraph hooks list 2>/dev/null || true
git config --get merge.beads.driver
```

## Reinstall Later

To initialize issuegraph again:

```bash
issuegraph init
```
