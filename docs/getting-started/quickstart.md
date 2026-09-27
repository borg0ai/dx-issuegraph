---
title: Quick Start
description: Initialize issuegraph, create issues with dependencies, find ready work, and sync with your team in a few minutes
---

Get up and running with IssueGraph in a few minutes.

## Why IssueGraph?

Flat issue trackers (GitHub Issues, Jira, etc.) show you a list of open items. You pick one. But if that item depends on something else that isn't done yet, you've wasted time. Multiply this across a team of AI agents and humans, and you get thrashing.

IssueGraph tracks **dependencies between issues** and computes a **ready queue** — only items with no active blockers appear. Here's the difference:

**Flat tracker (GitHub Issues):**
```
Open issues: Set up database, Create API, Add authentication
→ An agent picks "Add authentication" and gets stuck immediately
```

**IssueGraph:**
```bash
$ issuegraph ready
1. [P1] [task] bd-1: Set up database

$ issuegraph ready --explain --json | jq '.blocked[0]'
{
  "id": "bd-3",
  "title": "Add authentication",
  "blocked_by": [{"id": "bd-2", "title": "Create API", "status": "open"}]
}
```

The agent picks the right task every time. No wasted cycles.

## Installation

Install `issuegraph` using [the full installation guide](/getting-started/installation) (Homebrew, install script, npm, or `go install`).

**Developing in a clone of this repository:** use `make install` so the binary gets correct build metadata and a consistent install path. Avoid ad-hoc `go build` / `go install` without the Makefile unless you know what you are doing — see the repository `README` and `AGENTS.md`.

```bash
issuegraph --help
```

## Initialize

First time in a repository:

```bash
# Basic setup (prompts for contributor mode)
issuegraph init

# For AI agents (non-interactive)
issuegraph init --quiet

# OSS contributor (fork workflow with separate planning repo)
issuegraph init --contributor

# Team member (branch workflow for collaboration)
issuegraph init --team

# Protected main branch (GitHub/GitLab)
# Note: Dolt stores data under refs/dolt/data, separate from
# Git refs, so no --branch flag is needed.
```

The wizard will:
- Create `.beads/` directory and embedded Dolt database
- **Prompt for your role** (maintainer or contributor) unless a flag is provided
- Import existing issues from git (if any)
- Install git hooks (skip with `--skip-hooks`)

Notes:
- Dolt is the default (and only) storage backend. Data is stored in `.beads/embeddeddolt/`.
- By default, Dolt runs in **embedded mode** (in-process, no server needed).
- For multi-writer setups, use `issuegraph init --server` to connect to a `dolt sql-server` instead.
- To import issues from an older installation, run `issuegraph init --from-jsonl`.

### Role configuration

During `issuegraph init`, you'll be asked: "Contributing to someone else's repo? [y/N]"

- Answer **Y** if you're contributing to a fork (runs contributor wizard)
- Answer **N** if you're the maintainer or have push access

This sets `git config beads.role` which determines how issuegraph routes issues:

| Role | Use case | Issue storage |
|------|----------|---------------|
| `maintainer` | Repo owner, team with push access | In-repo `.beads/` |
| `contributor` | Fork contributor, OSS contributor | Separate planning repo |

You can also configure manually:

```bash
# Set as contributor
git config beads.role contributor

# Set as maintainer
git config beads.role maintainer

# Check current role
git config --get beads.role
```

**Note:** If `beads.role` is not configured, issuegraph falls back to URL-based detection (deprecated). Run `issuegraph doctor` to check configuration status.

## Your first issues

```bash
# Create a few issues
issuegraph create "Set up database" -p 1 -t task
issuegraph create "Create API" -p 2 -t feature
issuegraph create "Add authentication" -p 2 -t feature

# List them
issuegraph list
```

**Note:** Issue IDs are hash-based (e.g., `bd-a1b2`, `bd-f14c`) to prevent collisions when multiple agents/branches work concurrently.

## Hierarchical issues (epics)

For large features, use hierarchical IDs to organize work:

```bash
# Create epic (generates parent hash ID)
issuegraph create "Auth System" -t epic -p 1
# Returns: bd-a3f8e9

# Create child tasks (use --parent to attach to the epic)
issuegraph create "Design login UI" -p 1 --parent bd-a3f8e9       # bd-a3f8e9.1
issuegraph create "Backend validation" -p 1 --parent bd-a3f8e9    # bd-a3f8e9.2
issuegraph create "Integration tests" -p 1 --parent bd-a3f8e9     # bd-a3f8e9.3

# View hierarchy
issuegraph dep tree bd-a3f8e9
```

Output:
```
Dependency tree for bd-a3f8e9:

> bd-a3f8e9: Auth System [epic] [P1] (open)
  > bd-a3f8e9.1: Design login UI [P1] (open)
  > bd-a3f8e9.2: Backend validation [P1] (open)
  > bd-a3f8e9.3: Integration tests [P1] (open)
```

Children inherit the epic's labels by default — if the epic carries a size/effort label, see [Labels](/core-concepts/labels) for how to keep it off the children.

## Add dependencies

```bash
# API depends on database
issuegraph dep add bd-2 bd-1

# Auth depends on API
issuegraph dep add bd-3 bd-2

# View the tree
issuegraph dep tree bd-3
```

Output:
```
Dependency tree for bd-3:

> bd-3: Add authentication [P2] (open)
  > bd-2: Create API [P2] (open)
    > bd-1: Set up database [P1] (open)
```

**Dependency visibility:** `issuegraph list` shows blocking dependencies inline:
```
○ bd-a1b2 [P1] [task] - Set up database
○ bd-f14c [P2] [feature] - Create API (blocked by: bd-a1b2)
○ bd-g25d [P2] [feature] - Add authentication (blocked by: bd-f14c)
```

## Find ready work

```bash
issuegraph ready
```

Output:
```
Ready work (1 issues with no active blockers):

1. [P1] bd-1: Set up database
```

Only bd-1 is ready because bd-2 and bd-3 are blocked.

**Understanding why:** Use `--explain` to see the full graph reasoning:

```bash
issuegraph ready --explain
```

Output:
```
Ready Work Explanation

● Ready (1 issues):

  bd-1 [P1] Set up database
    Reason: no blocking dependencies
    Unblocks: 1 issue(s)

● Blocked (2 issues):

  bd-2 [P2] Create API
    ← blocked by bd-1: Set up database [open]

  bd-3 [P2] Add authentication
    ← blocked by bd-2: Create API [open]

─ Summary: 1 ready, 2 blocked
```

**Note:** `issuegraph ready` is not the same as `issuegraph list --status open`. The `list` command shows all open issues regardless of blockers. The `ready` command computes the dependency graph and only shows truly unblocked work.

## Work the queue

```bash
# Start working on bd-1
issuegraph update bd-1 --claim

# Complete it
issuegraph close bd-1 --reason "Database setup complete"

# Check ready work again
issuegraph ready
```

Now bd-2 is ready.

## Track progress

```bash
# See blocked issues
issuegraph blocked

# View statistics
issuegraph stats
```

## Team sync

Share issues with your team using Dolt remotes. Dolt stores data under `refs/dolt/data` on the same Git remote, separate from standard Git refs. In repos with `origin`, `issuegraph init` configures that Dolt remote automatically.

```bash
# Verify the remote, or add one if the repo had no origin during init
issuegraph dolt remote list
issuegraph dolt remote add origin git+ssh://git@github.com/org/repo.git  # if needed

# Push your issues
issuegraph dolt push

# Pull teammates' changes
issuegraph dolt pull
```

When a teammate clones the repo, `issuegraph bootstrap` auto-detects the existing database on `refs/dolt/data`, clones it, and wires `origin` for future `issuegraph dolt push` / `issuegraph dolt pull`.

See [`issuegraph dolt`](/cli-reference/dolt) for CLI details. For remote configuration, see [Dolt architecture](/architecture/dolt); for federation, see [federation](/multi-agent/federation).

## Optional: Notion sync

If you keep project issues in Notion, save an integration token first:

```bash
issuegraph config set notion.token <your-token>
```

Then either create a new IssueGraph database under a parent page or connect to an existing target:

```bash
issuegraph notion init --parent <page-id>
# or
issuegraph notion connect --url <notion-database-or-data-source-url>
```

The same auth value can also come from `NOTION_TOKEN`. Directly setting `notion.data_source_id` remains available as an escape hatch for advanced setups.

Check which auth source is active and whether the target schema is ready:

```bash
issuegraph notion status
issuegraph notion status --json
```

Preview or run sync:

```bash
issuegraph notion sync --dry-run
issuegraph notion sync
issuegraph notion sync --pull
issuegraph notion sync --push
```

## Database location

By default (embedded mode), data is stored in `.beads/embeddeddolt/` within your repository.
In server mode, data is managed by the external `dolt sql-server`.

## Migrating databases

After upgrading issuegraph, use `issuegraph migrate` to check for and migrate old database files:

```bash
# Inspect migration plan (AI agents)
issuegraph migrate --inspect --json

# Check schema and config
issuegraph info --schema --json

# Preview migration changes
issuegraph migrate --dry-run

# Migrate old databases to beads.db
issuegraph migrate

# Migrate and clean up old files
issuegraph migrate --yes
```

**AI agents:** Use `--inspect` to analyze migration safety before running. The system verifies required config keys and data integrity invariants.

## Database maintenance

As your project accumulates closed issues, the database grows. Manage size with these commands:

```bash
# View compaction statistics
issuegraph admin compact --stats

# Preview compaction candidates (30+ days closed)
issuegraph admin compact --analyze --json

# Apply agent-generated summary
issuegraph admin compact --apply --id bd-42 --summary summary.txt

# Immediately delete closed issues (CAUTION: permanent!)
issuegraph admin cleanup --force
```

**When to compact:**
- Database file > 10MB with many old closed issues
- After major project milestones when old issues are no longer relevant
- Before archiving a project phase

**Note:** Compaction is permanent graceful decay. Original content is discarded but recoverable via `issuegraph restore <id>` (from the pre-compaction snapshot, with Dolt history as fallback).

## Next steps

- Add labels: `issuegraph create "Task" -l "backend,urgent"`
- Filter ready work: `issuegraph ready --priority 1`
- Explain the graph: `issuegraph ready --explain`
- Check graph integrity: `issuegraph graph check`
- Search issues: `issuegraph list --status open`
- Detect cycles: `issuegraph dep cycles`
- Gates for PR/CI sync: [`issuegraph gate`](/cli-reference/gate)
- More sync scenarios: [`issuegraph dolt`](/cli-reference/dolt)
- Full command list: [CLI Reference](/cli-reference/index)

See the [repository README](https://github.com/gastownhall/beads/blob/main/README.md) for an overview and links to deeper docs.
