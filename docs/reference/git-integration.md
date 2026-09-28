---
title: Git Integration
description: How issuegraph uses git for hosting and hooks, including hook installation, external hook managers, worktrees, and branch workflows.
---

How issuegraph integrates with git.

## Overview

IssueGraph uses git for:
- **Project hosting** - Your code repository also hosts issuegraph configuration
- **Hooks** - Auto-sync on git operations

Data storage and sync are handled by Dolt (a version-controlled SQL database) —
see [Sync Concepts](/core-concepts/sync-concepts) for how issue data moves
between machines.

## File Structure

```
.beads/
├── config.yaml        # Project config (git-tracked)
├── metadata.json      # Backend metadata (git-tracked)
├── .gitignore         # Written by issuegraph init (git-tracked)
├── embeddeddolt/      # Dolt database — embedded mode, the default (gitignored)
└── dolt/              # Dolt database — server mode (gitignored)
```

`issuegraph init` writes `.beads/.gitignore` to keep the database directory and
runtime files out of git — no manual gitignore rules are needed. Never track
the database directory (`.beads/embeddeddolt/` or `.beads/dolt/`) in git or
via Git LFS.

## Git Hooks

### Installation

`issuegraph init` installs hooks by default (skip with `issuegraph init --skip-hooks`). To
install or refresh them manually:

```bash
issuegraph hooks install
```

Installed hooks are thin shims that call `issuegraph hooks run <hook-name>`. Upgrading
`issuegraph` automatically updates the delegated behavior inside that command; the
shim's generated shell policy remains installed content and changes only when
the hook is installed or refreshed:

| Hook | What it does |
|------|--------------|
| `pre-commit` | Runs chained hooks; when `export.auto` is enabled, exports `.beads/issues.jsonl` so it lands in the same commit |
| `post-merge` | Runs chained hooks; imports JSONL only as a legacy fallback when no Dolt remote is configured — with `sync.remote` set, `issuegraph dolt pull` is the canonical sync |
| `pre-push` | Runs chained hooks before push |
| `post-checkout` | Runs chained hooks after branch checkout |
| `prepare-commit-msg` | Adds an `Executed-By:` agent identity trailer when an agent (`BD_ACTOR`) makes the commit |

The shims use section markers to coexist with existing hooks — content
outside the markers is preserved across installs and upgrades. Install
variants:

```bash
issuegraph hooks install --beads    # Install to .beads/hooks/ (recommended for the Dolt backend)
issuegraph hooks install --shared   # Install to .beads-hooks/ (versioned, shareable with the team)
issuegraph hooks install --chain    # Run existing hooks before issuegraph hooks
```

Hook installation is worktree-aware: `issuegraph` resolves the shared git directory,
so installing from a linked worktree works.

### Status

```bash
issuegraph hooks list
```

### Uninstall

```bash
issuegraph hooks uninstall
```

### External Hook Managers

issuegraph detects these external git hook managers and checks whether their config
calls `issuegraph hooks run`:

- [lefthook](https://lefthook.dev/) — YAML/TOML/JSON config
- [husky](https://typicode.github.io/husky/) — `.husky/` directory scripts
- [pre-commit](https://pre-commit.com/) — `.pre-commit-config.yaml`
- [prek](https://prek.j178.dev/) — Rust-based pre-commit alternative (same config)
- [hk](https://hk.jdx.dev/) — fast hook manager using Pkl config
- [overcommit](https://github.com/sds/overcommit) — Ruby-based (detection only)
- yorkie — detection only
- [simple-git-hooks](https://github.com/toplenboren/simple-git-hooks) — lightweight JS (detection only)

`issuegraph doctor` reports whether a detected manager is integrated with issuegraph, and
`issuegraph doctor --fix` reinstalls the hooks with `--chain` so the manager's
existing hooks keep running.

For config-driven managers, add issuegraph steps directly. Example `hk.pkl`:

```pkl
hooks {
    ["pre-commit"] {
        steps {
            ["bd-pre-commit"] {
                check = "issuegraph hooks run pre-commit"
            }
        }
    }
    ["post-merge"] {
        steps {
            ["bd-post-merge"] {
                check = "issuegraph hooks run post-merge"
            }
        }
    }
    ["pre-push"] {
        steps {
            ["bd-pre-push"] {
                check = "issuegraph hooks run pre-push \"$@\""
            }
        }
    }
}
```

### Hook Timeout

The hook shim applies a soft deadline to `issuegraph hooks run` when a compatible
helper is available. It
uses `timeout` or `gtimeout` only after a successful GNU coreutils identity
probe, avoiding the incompatible `timeout.exe` that native Windows can place
on `PATH`. GNU timeout sends `TERM` at the configured deadline. On POSIX hosts,
the Perl fallback uses `SIGALRM` on the direct `issuegraph` process at the deadline.
Git for Windows Perl does not guarantee that alarm across `exec`, so GNU
coreutils is the preferred deadline backend there.

The default deadline is **300 seconds** (5 minutes), which accommodates chained
pre-commit pipelines (eslint, prettier, TypeScript compilation). Override it
with the `BEADS_HOOK_TIMEOUT` environment variable:

```bash
# Set a longer timeout (in seconds)
export BEADS_HOOK_TIMEOUT=600  # 10 minutes

# Or set it per-invocation
BEADS_HOOK_TIMEOUT=600 git commit -m "..."
```

The value must be a positive whole number of seconds. Invalid values and zero
produce a warning and use the 300-second default. These are soft process
deadlines, not process-tree containment: TERM-resistant work or descendants
can outlive them. If neither GNU timeout nor Perl is available, the hook warns
and runs directly without a deadline; that last-resort path can hang until the
hook itself returns.

After upgrading from a release whose generated hooks used a name-only timeout
check, run `issuegraph hooks install` once to refresh already-installed canonical hook
sections. Automatic generated-policy adoption is tracked separately.

When the timeout is reached, issuegraph prints a warning and lets the git
operation proceed — the commit or push is not blocked.

## Conflict Resolution

Dolt handles merge conflicts at the database level using its built-in
merge capabilities. When conflicts arise during sync, Dolt identifies
conflicting rows and allows resolution through SQL.

```bash
# Check for and fix conflicts
issuegraph doctor --fix
```

## Protected Branches

Dolt stores data under `refs/dolt/data`, separate from Git refs. This means
issuegraph data does not conflict with protected Git branches, and no separate
`beads-sync` branch or protected-branch exception is needed. On new projects
with a Git `origin`, `issuegraph init` configures that origin as the Dolt remote
automatically.

See [Protected Branches](/reference/protected-branches) for the full
workflow, including legacy `beads-sync` cleanup.

## Git Worktrees

IssueGraph works in Git worktrees without extra setup. Linked worktrees discover the
repository's `.beads` workspace and sync issue data through Dolt:

```bash
# In a linked worktree
issuegraph create "Task"
issuegraph list
issuegraph dolt pull
issuegraph dolt push
```

All worktrees share the repository's `.beads` workspace: discovery follows
`BEADS_DIR` if set, then the main repository's `.beads`, preventing database
duplication across worktrees. Use `issuegraph where` as the authoritative check for
which workspace is active — a local `./.beads` may legitimately be absent in
a worktree. Embedded mode (the default) serves one writer at a time; for
concurrent writers across worktrees, use server mode. See
[Git Worktrees](/reference/worktrees) for the full guide.

Older issuegraph versions documented a `sync.branch` workflow that created hidden
Git worktrees. That workflow has been removed; current sync uses Dolt remotes.

## Branch Workflows

### Feature Branch

```bash
git checkout -b feature-x
issuegraph create "Feature X" -t feature
# Work...
issuegraph dolt push
git push
```

### Fork Workflow

```bash
# In fork
issuegraph init --contributor   # Interactive wizard
# Work in separate planning repo...
issuegraph dolt push
```

The contributor wizard keeps issue data in a separate planning repository,
leaving the upstream repo without any `.beads/`. Best for open source
contributors, solo developers, and private task tracking on public repos.

`issuegraph init` auto-detects forks and offers to configure `.git/info/exclude`
(`--setup-exclude`) so issuegraph files stay local. Set the role without prompting
via `--role contributor` or `--role maintainer` (the default in
non-interactive mode).

### Team Workflow

```bash
issuegraph init --team
# All team members share the Dolt database
issuegraph dolt pull   # Pull latest changes from Dolt remote
issuegraph dolt push   # Push your changes to Dolt remote
```

Best for teams on protected branches and review-before-merge policies. See
[Multi-Repo Migration](/multi-agent/multi-repo-migration) for multi-repo
patterns.

### Duplicate Detection

After merging branches:

```bash
issuegraph duplicates --auto-merge
```

## Branchless Workflows (Jujutsu / jj)

IssueGraph works with branchless VCS tools like
[Jujutsu (jj)](https://martinvonz.github.io/jj/). Since issuegraph data is stored
in Dolt (not git branches), there is no dependency on the "current branch"
concept.

### What Works Without Hooks

All core issuegraph functionality works without git hooks:

| Feature | Hooks Required? | Notes |
|---------|----------------|-------|
| `issuegraph create`, `issuegraph update`, `issuegraph close` | No | Core CRUD uses Dolt directly |
| `issuegraph ready`, `issuegraph list`, `issuegraph show` | No | Read-only queries |
| `issuegraph dolt push` / `issuegraph dolt pull` | No | Dolt-native sync, independent of git |
| `issuegraph onboard`, `issuegraph doctor` | No | Diagnostics and onboarding |
| Agent identity trailers | Yes | `prepare-commit-msg` hook adds `Executed-By:` to commits |
| Hook chaining | Yes | Preserves existing pre-commit, post-merge hooks |

To skip hooks entirely during init:

```bash
issuegraph init --skip-hooks
```

### What Works Without AGENTS.md

The AGENTS.md file generated by `issuegraph init` provides AI agent instructions. If
you manage your own agent instructions or don't want issuegraph to modify tracked
files:

```bash
issuegraph init --skip-agents    # Skip AGENTS.md and Claude/Codex setup generation
issuegraph init --stealth        # Full invisible mode (also skips hooks + agents)
```

### Jujutsu Setup

**Colocated repos** (`jj git init --colocate`): Git hooks work normally.
IssueGraph installs simplified hooks (`pre-commit` and `post-merge` only, no
staging logic).

**Pure jj repos** (no git): Since jj doesn't have native hooks yet, set up
push aliases:

```toml
# ~/.config/jj/config.toml
[aliases]
push = ["util", "exec", "--", "sh", "-c", "issuegraph dolt commit && issuegraph dolt push && jj git push \"$@\"", ""]
```

Then use `jj push` instead of `jj git push`.

## Best Practices

1. **Install hooks** - `issuegraph hooks install`
2. **Push regularly** - `issuegraph dolt push` at session end
3. **Pull before work** - `issuegraph dolt pull` to get latest issues
4. **Use normal Git worktrees** - no sync branch is required
