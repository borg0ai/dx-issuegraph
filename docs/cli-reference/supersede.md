---
title: "issuegraph supersede"
description: "Mark an issue as superseded by a replacement issue"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc supersede`.

Mark an issue as superseded by a different (replacement) issue.

The superseded issue is automatically closed with a reference to the replacement.
Useful for design docs, specs, and evolving artifacts.

Examples:
  issuegraph supersede bd-old --with bd-new    # Mark bd-old as superseded by bd-new

```
issuegraph supersede <id> --with <new> [flags]
```

**Flags:**

```
      --with string   Replacement issue ID (required)
```
