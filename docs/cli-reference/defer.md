---
title: "issuegraph defer"
description: "Defer issues to put them on ice for later."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc defer`.

Defer issues to put them on ice for later.

Deferred issues are deliberately set aside - not blocked by anything specific,
just postponed for future consideration. Unlike blocked issues, there's no
dependency keeping them from being worked. Unlike closed issues, they will
be revisited.

Deferred issues don't show in 'issuegraph ready' but remain visible in 'issuegraph list'.

Examples:
  issuegraph defer issuegraph-abc                  # Defer a single issue (status-based)
  issuegraph defer issuegraph-abc --until=tomorrow # Defer until specific time
  issuegraph defer issuegraph-abc --reason="waiting on API access"
  issuegraph defer issuegraph-abc issuegraph-def           # Defer multiple issues

```
issuegraph defer [id...] [flags]
```

**Flags:**

```
      --reason string   Record why this issue is being deferred (appended to notes)
      --until string    Defer until specific time (e.g., +1h, tomorrow, next monday)
```
