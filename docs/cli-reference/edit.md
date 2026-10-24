---
title: "issuegraph edit"
description: "Edit an issue field using your configured $EDITOR."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc edit`.

Edit an issue field using your configured $EDITOR.

By default, edits the description. Use flags to edit other fields.

Examples:
  issuegraph edit issuegraph-42                    # Edit description
  issuegraph edit issuegraph-42 --title            # Edit title
  issuegraph edit issuegraph-42 --design           # Edit design notes
  issuegraph edit issuegraph-42 --notes            # Edit notes
  issuegraph edit issuegraph-42 --acceptance       # Edit acceptance criteria

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
