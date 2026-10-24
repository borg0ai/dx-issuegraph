---
title: "issuegraph duplicate"
description: "Mark an issue as a duplicate of a canonical issue."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc duplicate`.

Mark an issue as a duplicate of a canonical issue.

The duplicate issue is automatically closed with a reference to the canonical.
This is essential for large issue databases with many similar reports.

Examples:
  issuegraph duplicate issuegraph-abc --of issuegraph-xyz    # Mark issuegraph-abc as duplicate of issuegraph-xyz

```
issuegraph duplicate <id> --of <canonical> [flags]
```

**Flags:**

```
      --of string   Canonical issue ID (required)
```
