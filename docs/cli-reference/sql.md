---
title: "issuegraph sql"
description: "Execute a raw SQL query against the underlying database (SQLite or Dolt)."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc sql`.

Execute a raw SQL query against the underlying database (SQLite or Dolt).

Useful for debugging, maintenance, and working around bugs in higher-level commands.

Examples:
  issuegraph sql 'SELECT COUNT(*) FROM issues'
  issuegraph sql 'SELECT id, title FROM issues WHERE status = "open" LIMIT 5'
  issuegraph sql 'DELETE FROM dirty_issues WHERE issue_id = "issuegraph-abc123"'
  issuegraph sql --csv 'SELECT id, title, status FROM issues'

The query is passed directly to the database. SELECT queries return results as a
table (or JSON/CSV with --json/--csv). Non-SELECT queries (INSERT, UPDATE, DELETE)
report the number of rows affected.

WARNING: Direct database access bypasses the storage layer. Use with caution.

```
issuegraph sql <query> [flags]
```

**Flags:**

```
      --csv   Output results in CSV format
```
