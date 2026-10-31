---
title: Dolt history maintenance
description: Current scope for local Dolt history and remote sharing.
---

IssueGraph keeps Dolt history local. Remote history rewrites, remote backup
repointing, and cross-clone Dolt history recovery procedures from older releases
no longer apply. `issuegraph dolt push` and `issuegraph dolt pull` fail closed.

Use `issuegraph admin compact` for supported local maintenance. Keep a local
filesystem backup before database maintenance. For sharing issue state between
clones, use tracked `.issuegraph/` snapshots and Git.
