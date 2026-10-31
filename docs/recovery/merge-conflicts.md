---
title: Snapshot merge conflicts
description: Resolve Git conflicts in tracked IssueGraph snapshots.
---

Separate clones share issue snapshots through Git, so concurrent changes can
conflict in `.issuegraph/issues.jsonl` or `.issuegraph/manifest.json`. Resolve
those conflicts with your Git client, then validate the snapshot before
bootstrap or restore.

IssueGraph remote Dolt pull and federation sync commands are disabled. They do
not merge remote database history.

See [Move issue data between clones](/getting-started/sync-setup) for the
snapshot workflow.
