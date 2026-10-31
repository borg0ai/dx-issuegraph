---
title: "issuegraph sync"
description: "Disabled: remote issue synchronization is unsupported"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc sync`.

Remote issue synchronization is disabled.

IssueGraph stores issues in the local Dolt database and tracks a complete
workspace snapshot in Git. Use Git to move repository files between clones.
This command exits with an error before opening the database or contacting a remote.

```
issuegraph sync [flags]
```

**Flags:**

```
      --attempts int    Deprecated; remote issue synchronization is disabled (default 3)
      --no-adopt        Deprecated; remote issue synchronization is disabled
      --remote string   Unsupported: remote issue synchronization is disabled
  -y, --yes             Deprecated; remote issue synchronization is disabled
```
