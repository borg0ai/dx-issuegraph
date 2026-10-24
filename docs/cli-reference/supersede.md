---
title: "issuegraph supersede"
description: "Mark an issue as superseded by a newer version."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc supersede`.

Mark an issue as superseded by a newer version.

The superseded issue is automatically closed with a reference to the replacement.
Useful for design docs, specs, and evolving artifacts.

Examples:
  issuegraph supersede issuegraph-old --with issuegraph-new    # Mark issuegraph-old as superseded by issuegraph-new

```
issuegraph supersede <id> --with <new> [flags]
```

**Flags:**

```
      --with string   Replacement issue ID (required)
```
