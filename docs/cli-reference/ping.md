---
title: "issuegraph ping"
description: "Check database connectivity"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc ping`.

Lightweight health check that confirms issuegraph can reach its database.

Steps:
  1. Resolve the .issuegraph workspace
  2. Open the store (embedded or server)
  3. Run a trivial query (issue count)
  4. Report timing

Exit 0 on success, exit 1 on failure.

Examples:
  issuegraph ping              # Quick connectivity check
  issuegraph ping --json       # Structured output for automation

```
issuegraph ping [flags]
```
