---
title: Wisps
description: Ephemeral molecules for operational work that has no audit value once it's done.
---

Operational workflows — release checklists, health patrols, diagnostics —
create beads that are worthless the moment they close. **Wisps** are
molecules instantiated in the *vapor phase*: real beads you work through
normally, flagged `Ephemeral=true` so they stay out of sync and can be
deleted wholesale later.

## What are Wisps?

- Issues in the main database with the ephemeral flag set — worked on with
  normal `issuegraph` commands.
- Local by design: excluded from federation push by default
  (`federation.exclude_types` defaults to `[wisp]`) and not part of the
  shared audit trail.
- Deleted in bulk by `issuegraph purge` or `issuegraph mol wisp gc` once closed.

## Wisp vs Pour

| Aspect | Molecule (`issuegraph mol pour`) | Wisp (`issuegraph mol wisp`) |
|--------|--------------------------|----------------------|
| Persistence | permanent, part of history | ephemeral, purged when done |
| Sync | synced like any bead | excluded from federation push |
| Use case | feature work, anything worth referencing later | release runs, operational loops, health checks |

Formulas can declare `phase = "vapor"` to recommend wisp instantiation —
pouring a vapor-phase formula warns.

## The Wisp Lifecycle

```bash
# 1. Create — from a proto, or ad-hoc
issuegraph mol wisp <proto-id> [--var key=value]
issuegraph create "One-off check" --ephemeral

# 2. Execute — normal issuegraph operations work on wisp issues
issuegraph ready --mol <wisp-id>
issuegraph update <id> --claim
issuegraph close <id>

# 3a. Keep it after all: squash promotes to persistent (clears the flag)
issuegraph mol squash <wisp-id>

# 3b. Or burn: delete without creating a digest
issuegraph mol burn <wisp-id>
```

## Managing Wisps

```bash
issuegraph mol wisp list      # list all wisps in the current context
issuegraph mol wisp gc        # garbage collect old/abandoned wisps
issuegraph purge --force      # delete all closed ephemeral beads
```

## Forcing a Phase

`issuegraph mol bond` accepts phase overrides when combining work:

```bash
issuegraph mol bond mol-critical-bug wisp-patrol --pour   # persist a bug found during a patrol
```

## Best Practices

1. **Wisps for operational loops** — patrols, release runs, diagnostics.
2. **Molecules for tracked work** — anything with audit value gets poured,
   not wisped.
3. **Squash before you delete** — if a wisp surfaced something durable,
   `issuegraph mol squash` promotes it; burning is irreversible.
4. **Garbage collect regularly** — `issuegraph mol wisp gc` or `issuegraph purge --force`.
