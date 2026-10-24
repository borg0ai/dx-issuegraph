---
title: "issuegraph link"
description: "Link two issues with a dependency."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc link`.

Link two issues with a dependency.

Shorthand for 'issuegraph dep add &lt;id1&gt; &lt;id2&gt;'. By default creates a "blocks"
dependency (id2 blocks id1). Use --type to specify a different relationship.

Examples:
  issuegraph link issuegraph-123 issuegraph-456                    # issuegraph-456 blocks issuegraph-123
  issuegraph link issuegraph-123 issuegraph-456 --type related     # issuegraph-123 related to issuegraph-456
  issuegraph link issuegraph-123 issuegraph-456 --type parent-child

```
issuegraph link <id1> <id2> [flags]
```

**Flags:**

```
  -t, --type string   Dependency type (blocks|tracks|related|parent-child|discovered-from) (default "blocks")
```
