---
title: Sync Setup Guide
description: "Set up Dolt sync so issue data follows you across machines: remotes, bootstrapping a clone, and day-to-day push and pull"
---

Set up beads with Dolt sync so your issues follow you across computers.

## Prerequisites

You need two tools installed on every machine:

| Tool | Version | Install |
|------|---------|---------|
| **issuegraph** (beads CLI) | 0.59.0+ | See [Installation](/getting-started/installation) |
| **Dolt** | A specific pinned version, not "latest" | See [Which Dolt version to install](/architecture/dolt#which-dolt-version-to-install) |

Verify both are installed:

```bash
issuegraph version     # must be 0.59.0+
dolt version   # must match the pinned version
```

Install a specific Dolt version rather than `releases/latest`. On Dolt 2.3.x a
few percent of freshly created databases come up with `CALL
DOLT_RESET('--hard')` broken for the life of the server process, which breaks
`issuegraph flatten`, `issuegraph admin compact` and the rollback behind `issuegraph dolt pull`;
separately, the upstream `latest` URL can resolve to a *lower* version than
one released days earlier.
[Which Dolt version to install](/architecture/dolt#which-dolt-version-to-install)
has the measurements, the install command, and how to check a database you
already have.

## Initial Setup (First Computer)

### 1. Initialize beads

```bash
cd your-project
issuegraph init
```

This creates the `.beads/` directory with a Dolt database. If the git repo has
an `origin` remote, `issuegraph init` also configures a Dolt remote named `origin`
pointing at that same git URL. Dolt stores issue data under `refs/dolt/data`,
separate from normal source branches.

### 2. Create some issues

```bash
issuegraph create "Set up CI pipeline" -p 1 -t task
issuegraph create "Add authentication" -p 2 -t feature
issuegraph list
```

### 3. Verify or add a Dolt remote

In a normal git repo with `origin`, this should already be configured:

```bash
issuegraph dolt remote list
# Expected: origin  <your git origin URL>
```

If the repo had no `origin` during init, point beads at your Git remote for
sync:

```bash
# GitHub (SSH — recommended)
issuegraph dolt remote add origin git+ssh://git@github.com/org/repo.git

# GitHub (HTTPS)
issuegraph dolt remote add origin git+https://github.com/org/repo.git

# Other options: DoltHub, S3, GCS, local path
# See DOLT.md for all remote types
```

### 4. Push your issues

```bash
issuegraph dolt push
```

Verify the push worked:

```bash
git ls-remote origin | grep dolt
# Expected: <hash>  refs/dolt/data
```

## Existing Projects Without a Dolt Remote

Projects initialized by older versions of `issuegraph init` may have a local embedded
Dolt database and a committed `.beads/issues.jsonl`, but no Dolt remote. Fix
that from the machine whose local database is authoritative:

```bash
issuegraph dolt remote list
issuegraph export -o .beads/issues.pre-remote.jsonl   # optional issue audit export
issuegraph dolt remote add origin git+ssh://git@github.com/org/repo.git
issuegraph dolt push
```

`issuegraph dolt remote add origin ...` writes `sync.remote` to `.beads/config.yaml`.
Commit and push that config file with your normal git workflow. Other clones can
then run `issuegraph bootstrap` if their database is missing/stale, or `issuegraph dolt pull`
when they already have the right database.

## Cloning to a New Computer

When you clone a repo that already has beads data on the remote, a standard `git clone` does **not** fetch `refs/dolt/data`. You need to bootstrap the Dolt database.

### Quick path: issuegraph bootstrap

On recent versions of issuegraph, `issuegraph bootstrap` handles everything automatically:

```bash
git clone git@github.com:org/repo.git
cd repo

issuegraph bootstrap
```

`issuegraph bootstrap` auto-detects `refs/dolt/data` on origin, clones the Dolt database, and configures the remote. Verify with:

```bash
issuegraph list       # should show your issues
issuegraph history    # should show recent issue history
```

If `issuegraph bootstrap` succeeds, you're done — skip to [Day-to-day Sync](#day-to-day-sync).

### Manual path (if bootstrap fails)

If `issuegraph bootstrap` doesn't work (older issuegraph versions, unusual remote configs), follow these steps:

**Step 1: Confirm the remote has beads data**

```bash
git ls-remote origin | grep dolt
# Expected: <hash>  refs/dolt/data
# If missing, the remote has no beads data — use issuegraph init normally.
```

**Step 2: Initialize beads**

```bash
issuegraph init
```

This creates `.beads/` with an empty database. Ignore any warnings about `issuegraph bootstrap` — we'll replace the empty database manually.

**Step 3: Stop the Dolt server**

```bash
issuegraph dolt stop
```

**Step 4: Find your database name and remove the empty database**

```bash
# Check your database name
cat .beads/metadata.json    # look for "dolt_database"
```

The `dolt_database` field is your `<dbname>` (typically the repo name).

```bash
# Remove the empty database
rm -rf .beads/dolt/<dbname>/
```

**Step 5: Clone the Dolt data from the remote**

```bash
cd .beads/dolt
dolt clone git@github.com:org/repo.git <dbname>
cd ../..
```

**Step 6: Start the server and migrate**

```bash
issuegraph dolt start
issuegraph migrate --yes
```

**Step 7: Ensure the remote is registered**

```bash
issuegraph dolt remote add origin git+ssh://git@github.com/org/repo.git
```

If you see "remote already exists", that's fine — `dolt clone` already set it up.

**Step 8: Verify**

```bash
issuegraph dolt remote list   # should show origin
issuegraph list               # should show your issues
```

## Day-to-day Sync

Once set up on both machines, sync is two commands:

```bash
# Push your changes to the remote
issuegraph dolt push

# Pull changes from the remote
issuegraph dolt pull
```

### Typical workflow

```
Machine A                          Machine B
─────────                          ─────────
issuegraph create "New task" -p 1
issuegraph dolt push
                                   issuegraph dolt pull
                                   issuegraph update bd-a1b2 --claim
                                   issuegraph close bd-a1b2 --reason "Done"
                                   issuegraph dolt push
issuegraph dolt pull
issuegraph list                            # sees the closed task
```

### Important rules

- **Always use `issuegraph dolt ...` commands** — never run raw `dolt` CLI commands while the Dolt server is running. It causes journal corruption.
- **Commit before pulling** — if you have uncommitted working set changes, `issuegraph dolt pull` will fail with "cannot merge with uncommitted changes". Run `issuegraph dolt commit` first.
- **Push before switching machines** — unpushed changes only exist locally.
- **Do not use JSONL as sync** — `.beads/issues.jsonl` is an export for viewers and interchange. It is not the source of truth, not a full database backup, and cannot safely reconcile deletes or pruning.

## Troubleshooting

### "no common ancestor" on push

A stale `refs/dolt/data` from a previous database is conflicting. Clear it and retry:

```bash
git update-ref -d refs/dolt/data
issuegraph dolt push
```

### "cannot merge with uncommitted changes" on pull

Commit your working set first:

```bash
issuegraph dolt commit
issuegraph dolt pull
```

### "no store available" on push or commit

This was a bug in issuegraph < 0.59.0. Upgrade issuegraph:

```bash
brew upgrade beads
# or re-run the install script
```

### issuegraph list shows nothing after clone

The Dolt database wasn't bootstrapped. Either run `issuegraph bootstrap` or follow the [manual path](#manual-path-if-bootstrap-fails) above.

### Stale lock files after crash

```bash
issuegraph doctor --fix --yes
```

**WARNING**: Do NOT manually remove files inside `.dolt/` directories (including
`noms/LOCK`). These are Dolt-internal files and removing them **will cause
unrecoverable data corruption**. Dolt manages these files itself.

### "fatal: Unable to read current working directory"

The Dolt server's working directory no longer exists (common after branch switches). Restart it:

```bash
issuegraph dolt stop
issuegraph dolt start
```

## See Also

- [Sync Concepts](/core-concepts/sync-concepts) — The conceptual model behind this setup (why Dolt is the source of truth, what JSONL is for)
- [Quick Start](/getting-started/quickstart) — Getting started with beads
- [Dolt Backend for Beads](/architecture/dolt) — Dolt backend details, server modes, federation, remote types, and sync modes
- [Installation](/getting-started/installation) — Installation for all platforms

## Attribution

This guide was inspired by [@leonletto](https://github.com/leonletto)'s community setup guide at [leonletto.github.io/thrum](https://leonletto.github.io/thrum/docs.html#guides/beads-setup.html), which documented the end-to-end setup and sync process including the manual bootstrap workflow. Thanks for contributing to the beads community!
