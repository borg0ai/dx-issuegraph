---
title: "issuegraph statuses"
description: "List all valid issue statuses and their categories."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc statuses`.

List all valid issue statuses and their categories.

Built-in statuses (open, in_progress, blocked, etc.) are always valid.
Additional statuses can be configured via status.custom:

  issuegraph config set status.custom "in_review:active,qa_testing:wip,on_hold:frozen"

Categories control behavior:
  active  — appears in 'issuegraph ready' and default 'issuegraph list'
  wip     — excluded from 'issuegraph ready', visible in default 'issuegraph list'
  done    — excluded from 'issuegraph ready' and default 'issuegraph list'
  frozen  — excluded from 'issuegraph ready' and default 'issuegraph list'

Statuses without a category (legacy format) are valid but excluded from 'issuegraph ready'.

Examples:
  issuegraph statuses            # List all statuses with icons and categories
  issuegraph statuses --json     # Output as JSON


```
issuegraph statuses [flags]
```
