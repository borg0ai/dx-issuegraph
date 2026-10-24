---
title: "issuegraph q"
description: "Quick capture creates an issue and outputs only the issue ID."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc q`.

Quick capture creates an issue and outputs only the issue ID.
Designed for scripting and AI agent integration.

Example:
  issuegraph q "Fix login bug"           # Outputs: issuegraph-a1b2
  ISSUE=$(issuegraph q "New feature")    # Capture ID in variable
  issuegraph q "Task" | xargs issuegraph show    # Pipe to other commands

```
issuegraph q [title] [flags]
```

**Flags:**

```
  -l, --labels strings    Labels
  -p, --priority string   Priority (0-4 or P0-P4) (default "2")
  -t, --type string       Issue type (default "task")
```
