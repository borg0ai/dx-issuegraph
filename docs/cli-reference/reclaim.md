---
title: "issuegraph reclaim"
description: "Revert stale-lease in_progress issues back to ready (dead-worker recovery)"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc reclaim`.

Revert in_progress issues whose lease has gone stale back to ready.

When a worker claims an issue it takes a lease that expires after a TTL, kept
alive by 'issuegraph heartbeat'. A worker that dies stops heartbeating, so its lease
expires and its issue would otherwise stay in_progress forever. reclaim is the
reaper: it finds in_progress issues whose lease expired more than --older-than
ago, clears the assignee, and sets them back to open so another worker can
claim them. The previous owner's stale lease is recorded as a recovery event.

--older-than is a grace window past lease expiry: only leases that expired at
least this long ago are reclaimed, so a worker briefly paused (GC, clock skew)
is not robbed of live work. Run it from a supervisor on a timer with a window
of roughly 2× the claim TTL.

By default reclaim covers every stale lease THIS replica granted. The scope
filters below narrow it further, using the same label surface claiming is
scoped by (--label / --label-any / --exclude-label), plus --assignee and --id.
Filters AND-combine and never widen the set: a reclaimed lease must still be
stale.

Replicas and leases (federated deployments)
-------------------------------------------
A lease is only meaningful on the replica that granted it. Every other
replica's view of the holder's liveness is stale by up to one sync interval,
so a reaper elsewhere can revert a unit that is very much alive over there.
reclaim therefore records the granting replica on each lease and SKIPS a lease
another replica granted, summarizing what it declined on stderr (one line per
run; 'issuegraph -v' expands it to the first 20 leases individually). Reap it where it
was granted; use --any-replica only when that replica is permanently gone (or
when this node was renamed and its own old leases now look foreign — an
ordinary heartbeat keeps a lease alive but does not re-home it to the node
heartbeating it). Prefer the narrow form '--any-replica --id &lt;id&gt;': bare
--any-replica reverts EVERY foreign stale lease, live peers included.

Two invariants the guard cannot enforce for you:

  grace window &gt; sync interval, and lease TTL &gt; sync interval.

A TTL or grace shorter than the cadence at which replicas exchange state is
meaningless across the bridge — the remote view is a full interval old by
construction. Raise the TTL/grace above the sync interval, never the reverse.
The guard is opt-in: set ISSUEGRAPH_NODE_ID, or run 'issuegraph config set node_id &lt;name&gt;'
(which writes the per-machine ~/.config/issuegraph/config.yaml — never commit a node_id
to the git-tracked .issuegraph/config.yaml, or every clone reads the same name and
the guard goes armed-but-inert). One id per STORE, not per host: machines that
are clients of the same dolt sql-server are ONE replica and must share one value
or leave it unset. There is no hostname fallback — the hostname names the client
process's machine, not the store — so an unnamed deployment keeps the old,
unguarded behavior instead of stranding its own work.

Examples:
  issuegraph reclaim                       # default grace window (2× the lease TTL)
  issuegraph reclaim --older-than 10m      # reclaim leases expired &gt;10m ago
  issuegraph reclaim --older-than 0s       # reclaim every currently-expired lease
  issuegraph reclaim --label lane-a        # only this machine's claim partition
  issuegraph reclaim --label-any lane-a,lane-b --exclude-label pinned
  issuegraph reclaim --assignee zelda --assignee epona   # only these workers' leases
  issuegraph reclaim --id wy-abc --id wy-def             # exactly these issues
  issuegraph reclaim --any-replica         # also reap leases granted by a departed replica

```
issuegraph reclaim [flags]
```

**Flags:**

```
      --any-replica             Also reclaim leases granted by ANOTHER replica (unsafe unless that replica is gone; see 'Replicas and leases')
  -a, --assignee strings        Only reclaim leases held by these assignees (repeatable)
      --exclude-label strings   Never reclaim issues carrying ANY of these labels
      --id strings              Only reclaim these issue IDs (repeatable)
  -l, --label strings           Only reclaim issues with ALL these labels (AND). Can combine with --label-any
      --label-any strings       Only reclaim issues with AT LEAST ONE of these labels (OR). Can combine with --label
      --older-than duration     Only reclaim leases that expired at least this long ago (grace window) (default 10m0s)
```
