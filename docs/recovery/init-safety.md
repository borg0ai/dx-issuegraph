---
title: Init safety
description: Current local-only initialization and bootstrap behavior.
---

`issuegraph init` creates or updates a local database and does not inspect Git
remotes for Dolt history. `issuegraph init --remote <url>` exits with an error.
`issuegraph bootstrap` imports the tracked workspace snapshot and ignores
`refs/dolt/data`.

Older versions documented remote-history fork refusals and recovery commands.
Those conditions cannot occur through current IssueGraph commands. Remote Dolt
push, pull, and remote management commands fail before opening the database.

For snapshot recovery, see [Move issue data between clones](/getting-started/sync-setup).
For local database recovery, see [Database Corruption](/recovery/database-corruption).
