---
title: Federation
description: Status of the removed remote issue federation commands.
---

IssueGraph federation is disabled. `issuegraph federation` commands fail before
opening the database or contacting remote peers. Existing federation settings
cannot enable remote issue sync.

IssueGraph stores issue data in a local Dolt database and refreshes tracked
`.issuegraph/` snapshots. Use Git to share snapshots between clones. For the
current workflow, see [Move issue data between clones](/getting-started/sync-setup).
