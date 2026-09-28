---
title: TODO Command
description: The issuegraph todo command for managing lightweight TODO items as ordinary task-type issues, with add, list, and done shortcuts.
---

The `issuegraph todo` command provides a lightweight interface for managing TODO items as task-type issues.

## Philosophy

TODOs in issuegraph are not a separate tracking system - they are regular task-type issues with convenient shortcuts. This means:

- **No parallel systems**: TODOs use the same storage and sync as all other issues
- **Promotable**: Easy to convert a TODO to a bug/feature when needed
- **Full featured**: TODOs support all issuegraph features (dependencies, labels, routing)
- **Simple interface**: Quick commands for common TODO workflows

## Quick Start

```bash
# Add a TODO
issuegraph todo add "Fix the login bug" -p 1

# List TODOs
issuegraph todo

# Mark TODO as done
issuegraph todo done <id>
```

## Commands

### `issuegraph todo` (or `issuegraph todo list`)

List all open task-type issues.

```bash
issuegraph todo                  # List open TODOs
issuegraph todo list            # Same as above
issuegraph todo list --all      # Show completed TODOs too
issuegraph todo list --json     # JSON output
```

**Output:**
```
  ○ test-yxg  Fix the login bug                         P1  open
  ○ test-ryl  Update documentation                      P3  open

Total: 2 TODOs
```

### `issuegraph todo add <title>`

Create a new TODO item (task-type issue).

```bash
issuegraph todo add "Fix the login bug"                                # Default P2
issuegraph todo add "Update docs" -p 3 -d "Add examples"              # With priority and description
issuegraph todo add "Critical fix" --priority 0 --description "ASAP"  # P0 task
```

**Flags:**
- `-p, --priority <0-4>`: Priority (default: 2)
- `-d, --description <text>`: Description

### `issuegraph todo done <id> [<id>...]`

Mark one or more TODOs as complete.

```bash
issuegraph todo done test-abc              # Close one TODO
issuegraph todo done test-abc test-def     # Close multiple
issuegraph todo done test-abc --reason "Fixed in PR #42"  # With reason
```

**Flags:**
- `--reason <text>`: Reason for closing (default: "Completed")

## Converting TODOs

TODOs are regular task issues, so you can convert them:

```bash
# Promote TODO to bug
issuegraph update test-abc --type bug --priority 0

# Add dependencies
issuegraph dep add test-abc test-def

# Add labels
issuegraph update test-abc --set-labels "urgent,frontend"
```

## Viewing TODO Details

Use regular issuegraph commands:

```bash
issuegraph show test-abc        # View TODO details
issuegraph list --type task     # List all tasks (including TODOs)
issuegraph ready               # See ready TODOs in work queue
```

## Examples

### Daily TODO workflow

```bash
# Morning: add your tasks
issuegraph todo add "Review PRs"
issuegraph todo add "Fix CI pipeline" -p 1
issuegraph todo add "Update changelog" -p 3

# Check what's on your plate
issuegraph todo

# Complete work
issuegraph todo done <id>
issuegraph todo done <id>

# End of day: see what's left
issuegraph todo
```

### Converting TODO to full issue

```bash
# Start with a quick TODO
issuegraph todo add "Login is broken"

# Later, realize it's more serious
issuegraph update <id> --type bug --priority 0 --description "Users can't login, multiple reports"
issuegraph update <id> --acceptance "Login works for all user types"

# Now it's a full-fledged bug with proper tracking
issuegraph show <id>
```

## FAQ

**Q: Are TODOs different from tasks?**
A: No, TODOs are just task-type issues. The `issuegraph todo` command provides shortcuts for common task operations.

**Q: Can TODOs have dependencies?**
A: Yes! Use `issuegraph dep add <todo-id> <blocks-id>` like any other issue.

**Q: Do TODOs sync across machines?**
A: Yes, they're stored in the Dolt database and synced via Dolt remotes like all other issues.

**Q: Can I use TODOs with issuegraph ready?**
A: Yes! `issuegraph ready` shows all unblocked issues, including task-type TODOs.

**Q: Should I use TODOs or regular tasks?**
A: Use `issuegraph todo` for quick, informal tasks. Use `issuegraph create -t task` for tasks that need more context or are part of larger planning.

## Design Rationale

The TODO command follows issuegraph's philosophy of **minimal surface area**:

1. **No new types**: TODOs are task-type issues
2. **No special storage**: Same Dolt database as everything else
3. **Convenience layer**: Just shortcuts for common operations
4. **Fully compatible**: Works with all issuegraph features and commands

This ensures:
- No duplicate tracking systems
- No migration needed between TODOs and tasks
- Works with all existing issuegraph tooling (federation, compaction, routing)
- Simple to understand and maintain
