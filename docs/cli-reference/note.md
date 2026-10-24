---
title: "issuegraph note"
description: "Append a note to an issue's notes field."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc note`.

Append a note to an issue's notes field.

Shorthand for 'issuegraph update &lt;id&gt; --append-notes "text"'.

Examples:
  issuegraph note gt-abc "Fixed the flaky test"
  issuegraph note gt-abc Fixed the flaky test
  echo "note from pipe" | issuegraph note gt-abc --stdin
  issuegraph note gt-abc --file notes.txt

```
issuegraph note <id> [text...] [flags]
```

**Flags:**

```
      --file string   Read note text from file
      --stdin         Read note text from stdin
```
