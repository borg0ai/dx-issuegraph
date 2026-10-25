---
title: "issuegraph promote"
description: "Promote a wisp to a permanent issue"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc promote`.

Promote a wisp (ephemeral issue) to a permanent issue.

This copies the issue from the wisps table (dolt_ignored) to the permanent
issues table (Dolt-versioned), preserving labels, dependencies, events, and
comments. The original ID is preserved so all links keep working.

A comment is added recording the promotion and optional reason.

Examples:
  issuegraph promote issuegraph-wisp-abc123
  issuegraph promote issuegraph-wisp-abc123 --reason "Worth tracking long-term"

```
issuegraph promote <wisp-id> [flags]
```

**Flags:**

```
  -r, --reason string   Reason for promotion
```
