---
title: Remote sync is disabled
description: Explain the fail-closed behavior for removed remote issue sync.
---

IssueGraph no longer syncs Dolt history with remote databases. Commands including
`issuegraph sync`, `issuegraph dolt push`, `issuegraph dolt pull`, and federation
sync fail before opening the database or contacting a remote.

To share issue state between clones, commit and push the tracked
`.issuegraph/issues.jsonl` and `.issuegraph/manifest.json` files with your Git
client. Run `issuegraph bootstrap` in a fresh clone to import that snapshot.

For remote backup errors, choose a local filesystem path instead. Remote backup
destinations are disabled; local backups remain available through
`issuegraph backup init <path>` and `issuegraph backup sync`.
