---
title: "Issues & Dependencies"
description: "The issue model: fields, types, priorities, and the dependencies that decide what work is ready"
---

Understanding the issue model in issuegraph.

## Issue Structure

Every issue has:

```bash
issuegraph show issuegraph-42 --json
```

```json
{
  "id": "issuegraph-42",
  "title": "Implement authentication",
  "description": "Add JWT-based auth",
  "type": "feature",
  "status": "open",
  "priority": 1,
  "labels": ["backend", "security"],
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

## Issue Types

| Type | Use Case |
|------|----------|
| `bug` | Something broken that needs fixing |
| `feature` | New functionality |
| `task` | Work item (tests, docs, refactoring) |
| `epic` | Large feature with subtasks |
| `chore` | Maintenance (dependencies, tooling) |

## Priorities

| Priority | Level | Examples |
|----------|-------|----------|
| 0 | Critical | Security, data loss, broken builds |
| 1 | High | Major features, important bugs |
| 2 | Medium | Nice-to-have features, minor bugs |
| 3 | Low | Polish, optimization |
| 4 | Backlog | Future ideas |

## Creating Issues

```bash
# Basic issue
issuegraph create "Fix login bug" -t bug -p 1

# With description
issuegraph create "Add password reset" \
  --description="Users need to reset forgotten passwords via email" \
  -t feature -p 2

# With labels
issuegraph create "Update dependencies" -t chore -l "maintenance,security"

# JSON output for agents
issuegraph create "Task" -t task --json
```

## Dependencies

### Blocking Dependencies

The `blocks` relationship affects the ready queue:

```bash
# Add dependency: issuegraph-2 depends on issuegraph-1
issuegraph dep add issuegraph-2 issuegraph-1

# View dependencies
issuegraph dep tree issuegraph-2

# See blocked issues
issuegraph blocked

# See ready work (not blocked)
issuegraph ready
```

### Structural Relationships

These don't affect the ready queue:

```bash
# Parent-child (epic subtasks)
issuegraph create "Epic" -t epic
issuegraph create "Subtask" --parent issuegraph-42

# Discovered-from (found during work)
issuegraph create "Found bug" --deps discovered-from:issuegraph-42

# Related (soft link)
issuegraph dep relate issuegraph-1 issuegraph-2
```

### Dependency Types

| Type | Description | Ready Queue Impact |
|------|-------------|-------------------|
| `blocks` | Hard dependency | Yes - blocked items not ready |
| `parent-child` | Epic/subtask hierarchy | No |
| `discovered-from` | Tracks origin of discovery | No |
| `related` | Soft relationship | No |

## Hierarchical Issues

For large features, use hierarchical IDs:

```bash
# Create epic
issuegraph create "Auth System" -t epic -p 1
# Returns: issuegraph-a3f8e9

# Child tasks auto-number
issuegraph create "Design login UI" --parent issuegraph-a3f8e9     # issuegraph-a3f8e9.1
issuegraph create "Backend validation" --parent issuegraph-a3f8e9  # issuegraph-a3f8e9.2

# View hierarchy
issuegraph dep tree issuegraph-a3f8e9
```

## Updating Issues

```bash
# Change status
issuegraph update issuegraph-42 --claim

# Change priority
issuegraph update issuegraph-42 --priority 0

# Add labels
issuegraph update issuegraph-42 --add-label urgent

# Multiple changes
issuegraph update issuegraph-42 --claim --priority 1 --add-label "in-review"
```

## Closing Issues

```bash
# Simple close
issuegraph close issuegraph-42

# With reason
issuegraph close issuegraph-42 --reason "Implemented in PR #123"

# JSON output
issuegraph close issuegraph-42 --json
```

## Searching and Filtering

```bash
# By status
issuegraph list --status open
issuegraph list --status in_progress

# By priority
issuegraph list --priority 1
issuegraph list --priority 0,1  # Multiple

# By type
issuegraph list --type bug
issuegraph list --type feature,task

# By label
issuegraph list --label-any urgent,critical
issuegraph list --label-all backend,security

# Combined filters
issuegraph list --status open --priority 1 --type bug --json
```
