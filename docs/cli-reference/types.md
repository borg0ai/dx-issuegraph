---
title: "issuegraph types"
description: "List valid issue types"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc types`.

List all valid issue types that can be used with issuegraph create --type.

Core work types (bug, task, feature, chore, epic, decision, spike, story, milestone) are always valid.
Additional types require configuration via types.custom in .issuegraph/config.yaml.

Examples:
  issuegraph types              # List all types with descriptions
  issuegraph types --sections   # List required sections for each type
  issuegraph types --json       # Output as JSON


```
issuegraph types [flags]
```

**Flags:**

```
      --sections   Show required sections for each issue type
```
