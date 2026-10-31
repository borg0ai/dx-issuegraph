---
title: Bucket federation
description: Status of removed object-storage issue sync.
---

IssueGraph no longer sends issue data to S3, GCS, or other remote object stores.
Remote `issuegraph backup sync` and federation commands fail before database
access. Local filesystem backups remain supported.

Use tracked `.issuegraph/` snapshots and your Git client to share issues across
clones. See [Move issue data between clones](/getting-started/sync-setup).
