---
title: Move issue data between clones
description: Share IssueGraph's tracked workspace snapshot with Git
---

IssueGraph stores its working database locally. It writes the complete workspace
snapshot to `.issuegraph/issues.jsonl` and `.issuegraph/manifest.json` for Git to
track.

## Share a snapshot

Commit and push both snapshot files with your normal Git workflow. IssueGraph
does not run Git network commands or send issue history to a Dolt remote.

On a fresh clone, run `issuegraph bootstrap`. If no local database exists,
bootstrap imports the tracked snapshot.

To replace an existing local database with the snapshot, review the files and run
`issuegraph snapshot restore`. Restore deletes local issues that the snapshot no
longer lists.

## Disabled commands

`issuegraph dolt push`, `issuegraph dolt pull`, `issuegraph sync`, and federation
commands exit with an error before opening a database or contacting a remote.
Legacy `sync.remote`, `sync.git-remote`, `federation.remote`, and
`dolt.auto-push` settings cannot enable remote issue sync.

See [Git integration](/reference/git-integration) for the snapshot contract.
