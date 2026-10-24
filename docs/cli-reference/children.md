---
title: "issuegraph children"
description: "List all issuegraph that are children of the specified parent issue."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc children`.

List all issuegraph that are children of the specified parent issue.

This is a convenience alias for 'issuegraph list --parent &lt;id&gt; --status all'.
Unlike plain 'issuegraph list', children includes closed issues by default,
since the primary use case is inspecting all work under a parent.

Examples:
  issuegraph children hq-abc123        # List all children of hq-abc123
  issuegraph children hq-abc123 --json # List children in JSON format
  issuegraph children hq-abc123 --pretty # Show children in tree format

```
issuegraph children <parent-id> [flags]
```

**Flags:**

```
      --pretty   Show children in tree format
```
