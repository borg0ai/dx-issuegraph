---
title: "issuegraph migrate-personal"
description: "Move personal planning issues from the project database to your planning repo"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc migrate-personal`.

Identify issues you created in the project database and move them to your
personal planning repository (~/.issuegraph-planning by default).

This is a one-time migration for contributors who created personal planning
issues before contributor routing was configured.

The command:
  1. Finds all issues in the project database created by your git identity
  2. Shows you the list and asks for confirmation
  3. Moves them to the planning repo configured in routing.contributor

EXAMPLES:
  issuegraph migrate-personal        # Interactive: show list and prompt
  issuegraph migrate-personal -y     # Non-interactive: skip confirmation

```
issuegraph migrate-personal [flags]
```

**Flags:**

```
  -y, --yes   Skip confirmation prompt
```
