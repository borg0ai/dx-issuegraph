---
title: "issuegraph comment"
description: "Add a comment to an issue"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc comment`.

Add a comment to an issue.

Shorthand for 'issuegraph comments add &lt;id&gt; "text"'.

Examples:
  issuegraph comment bd-123 "Working on this now"
  issuegraph comment bd-123 Working on this now
  echo "comment from pipe" | issuegraph comment bd-123 --stdin
  issuegraph comment bd-123 --file notes.txt

Note: "comment" (singular) only adds a comment — it has no "list" subcommand.
To list comments on an issue, use the plural form: issuegraph comments &lt;id&gt;

```
issuegraph comment <id> [text...] [flags]
```

**Flags:**

```
      --file string   Read comment text from file
      --stdin         Read comment text from stdin
```
