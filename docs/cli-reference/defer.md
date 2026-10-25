---
title: "issuegraph defer"
description: "Defer one or more issues for later"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc defer`.

Defer issues to put them on ice for later.

Deferred issues are deliberately set aside - not blocked by anything specific,
just postponed for future consideration. Unlike blocked issues, there's no
dependency keeping them from being worked. Unlike closed issues, they will
be revisited.

Deferred issues don't show in 'issuegraph ready' but remain visible in 'issuegraph list'.

A defer WITH a date is a snooze: once --until passes, the next ready-front
read returns the issue to open automatically (same shape as 'issuegraph undefer').
A defer WITHOUT a date is the indefinite icebox: it stays deferred until
someone runs 'issuegraph undefer'.

Examples:
  issuegraph defer bd-abc                  # Icebox indefinitely (until issuegraph undefer)
  issuegraph defer bd-abc --until=tomorrow # Snooze: auto-wakes once the date passes
  issuegraph defer bd-abc --reason="waiting on API access"
  issuegraph defer bd-abc bd-def           # Defer multiple issues

```
issuegraph defer [id...] [flags]
```

**Flags:**

```
      --reason string   Record why this issue is being deferred (appended to notes)
      --until string    Defer until specific time (e.g., +1h, tomorrow, next monday)
```
