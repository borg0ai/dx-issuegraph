---
title: Graph Links in IssueGraph
description: "Non-blocking links between issues: replies-to threads, relates-to, duplicates, and supersedes (replacement) links"
---

IssueGraph supports several types of links between issues to create a knowledge graph. These links enable rich querying and traversal beyond simple blocking dependencies.

## Link Types

### replies-to - Conversation Threading

Creates message threads, similar to email or chat conversations.

**Created by:**
- Orchestrator mail reply commands (orchestrator handles messaging)
- `issuegraph dep add <new-id> <original-id> --type replies-to` (manual linking)

**Use cases:**
- Agent-to-agent message threads
- Discussion chains on issues
- Follow-up communications

**Example:**

```bash
# Original message (via orchestrator mail)
# orchestrator mail send worker/ -s "Review needed" -m "Please review issue-xyz"
# Creates: msg-a1b2

# Reply (automatically sets replies-to)
# orchestrator mail reply msg-a1b2 -m "Done! Approved with minor comments."
# Creates: msg-c3d4 with replies-to: msg-a1b2
```

**Viewing threads:**

```bash
issuegraph show gt-a1b2 --thread
```

### relates-to - Loose Associations

Bidirectional "see also" links between related issues. Not blocking, not hierarchical - just related.

**Created by:**
- `issuegraph dep relate <id1> <id2>` - Links both issues to each other

**Removed by:**
- `issuegraph dep unrelate <id1> <id2>` - Removes link in both directions

**Use cases:**
- Cross-referencing related features
- Linking bugs to associated tasks
- Building knowledge graphs
- "See also" connections

**Example:**

```bash
# Link two related issues
issuegraph dep relate issuegraph-auth issuegraph-security
# Result: issuegraph-auth.relates-to includes issuegraph-security
#         issuegraph-security.relates-to includes issuegraph-auth

# View related issues
issuegraph show issuegraph-auth
# Shows: Related: issuegraph-security

# Remove the link
issuegraph dep unrelate issuegraph-auth issuegraph-security
```

**Multiple links:**
An issue can have multiple relates-to links:

```bash
issuegraph dep relate issuegraph-api issuegraph-auth
issuegraph dep relate issuegraph-api issuegraph-docs
issuegraph dep relate issuegraph-api issuegraph-tests
# issuegraph-api now relates to 3 issues
```

### duplicates - Deduplication

Marks an issue as a duplicate of a canonical issue. The duplicate is automatically closed.

**Created by:**
- `issuegraph duplicate <id> --of <canonical>`

**Use cases:**
- Consolidating duplicate bug reports
- Merging similar feature requests
- Database deduplication at scale

**Example:**

```bash
# Two similar bug reports exist
issuegraph show issuegraph-bug1  # "Login fails on Safari"
issuegraph show issuegraph-bug2  # "Safari login broken"

# Mark bug2 as duplicate of bug1
issuegraph duplicate issuegraph-bug2 --of issuegraph-bug1
# Result: issuegraph-bug2 is closed with duplicate_of: issuegraph-bug1

# View shows the relationship
issuegraph show issuegraph-bug2
# Status: closed
# Duplicate of: issuegraph-bug1
```

**Behavior:**
- Duplicate issue is automatically closed
- Original (canonical) issue remains open
- `duplicate_of` field stores the canonical ID

### supersedes - Replacement Links

Marks an issue as superseded by a different (replacement) issue. The old issue is automatically closed. This is a replacement link, not a version relation: the new issue takes the old one's place.

**Created by:**
- `issuegraph supersede <old-id> --with <new-id>`

**Use cases:**
- Design document versions
- Spec evolution
- Artifact versioning
- RFC chains

**Example:**

```bash
# Original design doc
issuegraph create --title "Design Doc v1" --type task
# Creates: issuegraph-doc1

# Later, create updated version
issuegraph create --title "Design Doc v2" --type task
# Creates: issuegraph-doc2

# Mark v1 as superseded
issuegraph supersede issuegraph-doc1 --with issuegraph-doc2
# Result: issuegraph-doc1 closed with superseded_by: issuegraph-doc2

# View shows the replacement
issuegraph show issuegraph-doc1
# Status: closed
# Superseded by: issuegraph-doc2
```

**Behavior:**
- Old issue is automatically closed
- New issue remains in its current state
- `superseded_by` field stores the replacement ID

## Schema Fields

These fields are added to issues:

| Field | Type | Description |
|-------|------|-------------|
| `replies-to` | string | ID of parent message (threading) |
| `relates-to` | []string | IDs of related issues (bidirectional) |
| `duplicate_of` | string | ID of canonical issue |
| `superseded_by` | string | ID of replacement issue |

## Querying Links

### View Issue Details

```bash
issuegraph show <id>
```

Shows all link types for an issue:

```
issuegraph-auth: Implement authentication
Status: open
Priority: P1

Related to (3):
  issuegraph-security: Security audit
  issuegraph-users: User management
  issuegraph-sessions: Session handling
```

### View Threads

```bash
issuegraph show <id> --thread
```

Follows `replies-to` chain to show conversation history.

### JSON Output

```bash
issuegraph show <id> --json
```

Returns all fields including graph links:

```json
{
  "id": "issuegraph-auth",
  "title": "Implement authentication",
  "relates-to": ["issuegraph-security", "issuegraph-users", "issuegraph-sessions"],
  "duplicate_of": "",
  "superseded_by": ""
}
```

## Comparison with Dependencies

| Link Type | Blocking? | Hierarchical? | Direction |
|-----------|-----------|---------------|-----------|
| `blocks` | Yes | No | One-way |
| `parent_id` | No | Yes | One-way |
| `relates-to` | No | No | Bidirectional |
| `replies-to` | No | No | One-way |
| `duplicate_of` | No | No | One-way |
| `superseded_by` | No | No | One-way |

## Use Cases

### Knowledge Base

Link related documentation:

```bash
issuegraph dep relate issuegraph-api-ref issuegraph-quickstart
issuegraph dep relate issuegraph-api-ref issuegraph-examples
issuegraph dep relate issuegraph-quickstart issuegraph-install
```

### Bug Triage

Consolidate duplicate reports:

```bash
# Find potential duplicates
issuegraph duplicates

# Merge duplicates
issuegraph duplicate issuegraph-bug42 --of issuegraph-bug17
issuegraph duplicate issuegraph-bug58 --of issuegraph-bug17
```

### Replacement History

Track document evolution:

```bash
issuegraph supersede issuegraph-rfc1 --with issuegraph-rfc2
issuegraph supersede issuegraph-rfc2 --with issuegraph-rfc3
# issuegraph-rfc3 is now the live issue
```

### Message Threading

Build conversation chains (via orchestrator mail):

```bash
# orchestrator mail send dev/ -s "Question" -m "How does X work?"
# orchestrator mail reply msg-q1 -m "X works by..."
# orchestrator mail reply msg-q1.reply -m "Thanks!"
```

## Best Practices

1. **Use relates-to sparingly** - Too many links become noise
2. **Prefer specific link types** - `duplicates` is clearer than generic relates-to
3. **Keep threads shallow** - Deep reply chains are hard to follow
4. **Document replacements** - Note why the issue was replaced
5. **Query before creating duplicates** - `issuegraph search` first

## See Also

- [Messaging](https://github.com/borg0ai/dx-issuegraph/blob/main/engdocs/messaging.md) - Mail commands and threading
- [Dependencies](/getting-started/quickstart#add-dependencies) - Blocking dependencies
- [CLI Reference](/cli-reference/index) - All commands
