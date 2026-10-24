---
title: "issuegraph rename-prefix"
description: "Rename the issue prefix for all issues in the database."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc rename-prefix`.

Rename the issue prefix for all issues in the database.
This will update all issue IDs and all text references across all fields.

USE CASES:
- Shortening long prefixes (e.g., 'knowledge-work-' → 'kw-')
- Rebranding project naming conventions
- Consolidating multiple prefixes after database corruption
- Migrating to team naming standards

Prefix validation rules:
- Max length: 8 characters
- Allowed characters: lowercase letters, numbers, hyphens
- Must start with a letter
- Must end with a hyphen (e.g., 'kw-', 'work-')
- Cannot be empty or just a hyphen

Multiple prefix detection and repair:
If issues have multiple prefixes (corrupted database), use --repair to consolidate them.
The --repair flag will rename all issues with incorrect prefixes to the new prefix,
preserving issues that already have the correct prefix.

EXAMPLES:
  issuegraph rename-prefix kw-                # Rename from 'knowledge-work-' to 'kw-'
  issuegraph rename-prefix mtg- --repair      # Consolidate multiple prefixes into 'mtg-'
  issuegraph rename-prefix team- --dry-run    # Preview changes without applying

NOTE: This is a rare operation. Most users never need this command.

```
issuegraph rename-prefix <new-prefix> [flags]
```

**Flags:**

```
      --dry-run   Preview changes without applying them
      --repair    Repair database with multiple prefixes by consolidating them
```
