---
title: "issuegraph history"
description: "Show version history for an issue"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc history`.

Show the complete version history of an issue, including all commits
where the issue was modified.

Examples:
  issuegraph history bd-123           # Show all history for issue bd-123
  issuegraph history bd-123 --limit 5 # Show last 5 changes
  issuegraph history bd-123 --events  # Show database audit events

```
issuegraph history <id> [flags]
```

**Flags:**

```
      --events      Show database audit events instead of commit snapshots
      --limit int   Limit number of history entries (0 = all)
```
