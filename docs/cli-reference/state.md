---
title: "issuegraph state"
description: "Query the current value of a state dimension from an issue's labels."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc state`.

Query the current value of a state dimension from an issue's labels.

State labels follow the convention &lt;dimension&gt;:&lt;value&gt;, for example:
  patrol:active
  mode:degraded
  health:healthy

This command extracts the value for a given dimension.

Examples:
  issuegraph state witness-abc patrol     # Output: active
  issuegraph state witness-abc mode       # Output: normal
  issuegraph state witness-abc health     # Output: healthy

```
issuegraph state <issue-id> <dimension> [flags]
```

## issuegraph state list

List all state labels (dimension:value format) on an issue.

This filters labels to only show those following the state convention.

Example:
  issuegraph state list witness-abc
  # Output:
  #   patrol: active
  #   mode: normal
  #   health: healthy

```
issuegraph state list <issue-id> [flags]
```
