---
description: Explain local issue data snapshots and disabled remote issue sync
argument-hint: ""
---

IssueGraph stores its database locally and refreshes the tracked `.issuegraph/`
snapshot. It does not sync issue data through Dolt remotes, federation, object
storage, or Git refs.

`issuegraph sync`, `issuegraph dolt push`, `issuegraph dolt pull`, and federation
sync commands exit with an error before opening the database. Legacy remote
settings cannot re-enable them. `issuegraph init --remote` also fails.

Use the user's Git client to share the tracked snapshot with repository changes.
For a restorable local database copy, configure a filesystem destination with
`issuegraph backup init <path>` and run `issuegraph backup sync`. Remote backup
destinations are disabled.
