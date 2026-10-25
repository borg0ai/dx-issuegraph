---
title: "issuegraph undefer"
description: "Undefer one or more issues (restore to open)"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc undefer`.

Undefer issues to restore them to open status.

This brings issues back from the icebox so they can be worked on again.
Issues will appear in 'issuegraph ready' if they have no blockers.

Examples:
  issuegraph undefer bd-abc        # Undefer a single issue
  issuegraph undefer bd-abc bd-def # Undefer multiple issues

```
issuegraph undefer [id...] [flags]
```
