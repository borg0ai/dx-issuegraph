---
title: "issuegraph link"
description: "Link two issues with a dependency"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc link`.

Link two issues with a dependency.

Shorthand for 'issuegraph dep add &lt;id1&gt; &lt;id2&gt;'. By default creates a "blocks"
dependency (id2 blocks id1). Use --type to specify a different relationship.

Examples:
  issuegraph link bd-123 bd-456                    # bd-456 blocks bd-123
  issuegraph link bd-123 bd-456 --type related     # bd-123 related to bd-456
  issuegraph link bd-123 bd-456 --type parent-child

```
issuegraph link <id1> <id2> [flags]
```

**Flags:**

```
  -t, --type string   Dependency type (blocks|tracks|related|parent-child|discovered-from) (default "blocks")
```
