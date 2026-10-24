---
title: "issuegraph priority"
description: "Set the priority of an issue."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc priority`.

Set the priority of an issue.

Shorthand for 'issuegraph update &lt;id&gt; --priority &lt;n&gt;'.

Priority levels:
  0 - Critical (security, data loss, broken builds)
  1 - High (major features, important bugs)
  2 - Medium (default)
  3 - Low (polish, optimization)
  4 - Backlog (future ideas)

Examples:
  issuegraph priority issuegraph-123 0    # Critical
  issuegraph priority issuegraph-123 2    # Medium

```
issuegraph priority <id> <n> [flags]
```
