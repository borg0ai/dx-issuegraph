---
title: "issuegraph heartbeat"
description: "Refresh the lease on an issue you hold in_progress"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc heartbeat`.

Refresh the lease on an issue you currently hold in_progress.

A claim carries a lease that expires after a TTL. A worker keeps its claim alive
by heartbeating faster than the TTL; once it stops (because it died), the lease
goes stale and 'issuegraph reclaim' reverts the issue to ready so another worker can pick
it up. Heartbeat pushes lease_expires_at forward and stamps heartbeat_at = now.

Only the current owner may heartbeat. If the lease has already been reclaimed or
the issue closed, heartbeat fails so the worker learns to stop.

Leases live in an ephemeral, node-local table: heartbeats write no Dolt commit
and no history, so any cadence comfortably below the TTL is fine. Leases are
only enforceable on the node that granted them; cross-machine claim visibility
rides the issue's status and assignee, which do commit.

Examples:
  issuegraph heartbeat bd-123
  issuegraph hb bd-123

```
issuegraph heartbeat <id> [flags]
```

**Aliases:** hb
