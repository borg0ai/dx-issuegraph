---
description: Reopen closed issues
argument-hint: "[issue-ids...] [--reason]"
---

Reopen one or more closed issues.

Sets status to 'open' and clears the closed_at timestamp. Emits a Reopened event.

## Usage

- **Reopen single**: `issuegraph reopen ig-42`
- **Reopen multiple**: `issuegraph reopen ig-42 ig-43 ig-44`
- **With reason**: `issuegraph reopen ig-42 --reason "Found regression"`

More explicit than `issuegraph update --status open` - specifically designed for reopening workflow.

Common reasons for reopening:
- Regression found
- Requirements changed
- Incomplete implementation
- New information discovered
