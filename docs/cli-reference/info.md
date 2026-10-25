---
title: "issuegraph info"
description: "Show database information"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc info`.

Display information about the current database.

This command helps debug issues where issuegraph is using an unexpected database. It shows:
  - The absolute path to the database file
  - Database statistics (issue count)
  - Schema information (with --schema flag)
  - What's new in recent versions (with --whats-new flag)

Examples:
  issuegraph info
  issuegraph info --json
  issuegraph info --schema --json
  issuegraph info --whats-new
  issuegraph info --whats-new --json
  issuegraph info --thanks

```
issuegraph info [flags]
```

**Flags:**

```
      --schema      Include schema information in output
      --thanks      Show thank you page for contributors
      --whats-new   Show agent-relevant changes from recent versions
```
