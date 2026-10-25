---
title: "issuegraph merge-slot"
description: "Manage merge-slot gates for serialized conflict resolution"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc merge-slot`.

Merge-slot gates serialize conflict resolution in the merge queue.

A merge slot is an exclusive access primitive: only one agent can hold it at a time.
This prevents "monkey knife fights" where multiple polecats race to resolve conflicts
and create cascading conflicts.

Each rig has one merge slot issue: &lt;prefix&gt;-merge-slot (labeled gt:slot).
The slot uses:
  - status=open: slot is available
  - status=in_progress: slot is held
  - metadata.holder: who currently holds the slot
  - metadata.waiters: priority-ordered queue of waiters

Examples:
  issuegraph merge-slot create              # Create merge slot for current rig
  issuegraph merge-slot check               # Check if slot is available
  issuegraph merge-slot acquire             # Try to acquire the slot
  issuegraph merge-slot release             # Release the slot

```
issuegraph merge-slot [command]
```

## issuegraph merge-slot acquire

Attempt to acquire the merge slot for exclusive access.

If the slot is available (status=open), it will be acquired:
  - status set to in_progress
  - holder set to the requester

If the slot is held (status=in_progress), the command fails unless
--wait is passed, which adds the requester to the waiters queue.

Use --holder to specify who is acquiring (default: ISSUEGRAPH_ACTOR env var).

```
issuegraph merge-slot acquire [flags]
```

**Flags:**

```
      --holder string   Who is acquiring the slot (default: ISSUEGRAPH_ACTOR)
      --wait            Add to waiters list if slot is held
```

## issuegraph merge-slot check

Check if the merge slot is available or held.

Returns:
  - available: slot can be acquired
  - held by &lt;holder&gt;: slot is currently held
  - not found: no merge slot exists for this rig

```
issuegraph merge-slot check [flags]
```

## issuegraph merge-slot create

Create a merge slot issue for serialized conflict resolution.

The slot ID is automatically generated based on the issuegraph prefix (e.g., gt-merge-slot).
The slot is created with status=open (available).

```
issuegraph merge-slot create [flags]
```

## issuegraph merge-slot release

Release the merge slot after conflict resolution is complete.

Sets status back to open and clears the holder field.
If there are waiters, the highest-priority waiter should then acquire.

```
issuegraph merge-slot release [flags]
```

**Flags:**

```
      --holder string   Who is releasing the slot (for verification)
```
