---
title: "issuegraph undefer"
description: "Undefer issues to restore them to open status."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc undefer`.

Undefer issues to restore them to open status.

This brings issues back from the icebox so they can be worked on again.
Issues will appear in 'issuegraph ready' if they have no blockers.

Examples:
  issuegraph undefer issuegraph-abc        # Undefer a single issue
  issuegraph undefer issuegraph-abc issuegraph-def # Undefer multiple issues

```
issuegraph undefer [id...] [flags]
```
