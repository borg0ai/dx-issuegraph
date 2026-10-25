---
title: "issuegraph reopen"
description: "Reopen one or more closed issues"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc reopen`.

Reopen closed issues by setting status to 'open' and clearing the closed_at timestamp.
This is more explicit than 'issuegraph update --status open' and emits a Reopened event.

```
issuegraph reopen [id...] [flags]
```

**Flags:**

```
  -r, --reason string   Reason for reopening
```
