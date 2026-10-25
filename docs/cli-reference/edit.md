---
title: "issuegraph edit"
description: "Edit an issue field in $EDITOR"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc edit`.

Edit an issue field using your configured $EDITOR.

By default, edits the description. Use flags to edit other fields.

Examples:
  issuegraph edit bd-42                    # Edit description
  issuegraph edit bd-42 --title            # Edit title
  issuegraph edit bd-42 --design           # Edit design notes
  issuegraph edit bd-42 --notes            # Edit notes
  issuegraph edit bd-42 --acceptance       # Edit acceptance criteria

```
issuegraph edit [id] [flags]
```

**Flags:**

```
      --acceptance    Edit the acceptance criteria
      --description   Edit the description (default)
      --design        Edit the design notes
      --notes         Edit the notes
      --title         Edit the title
```
