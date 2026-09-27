# Messaging in IssueGraph

IssueGraph supports messaging as a first-class issue type, enabling inter-agent and human-agent communication within the same system used for issue tracking.

## Architecture

Mail commands (`issuegraph mail`) delegate to an external mail provider (typically `gt mail` in an orchestrator). IssueGraph stores messages as issues with `type: message`, threading via `replies_to` dependencies, and ephemeral lifecycle via the `ephemeral` flag.

This design separates concerns:
- **IssueGraph** = data plane (stores messages as issues)
- **Orchestrator** = control plane (routing, delivery, notifications)

## Setup

Configure the mail delegate (one-time):

```bash
# Environment variable (recommended for agents)
export BEADS_MAIL_DELEGATE="gt mail"

# Or per-project config
issuegraph config set mail.delegate "gt mail"
```

## Sending and Receiving

```bash
# Send mail (delegates to gt mail)
issuegraph mail send worker/ -s "Review needed" -m "Please review bd-abc"

# Check inbox
issuegraph mail inbox

# Read a message
issuegraph mail read msg-123

# Reply to a thread
issuegraph mail reply msg-123 -m "Reviewed and approved"
```

## Message Issue Type

Messages are issues with `type: message`:

| Field | Purpose |
|-------|---------|
| `type` | `message` |
| `sender` | Who sent the message |
| `assignee` | Recipient |
| `title` | Subject line |
| `description` | Message body |
| `status` | `open` (unread) / `closed` (read) |
| `ephemeral` | If true, eligible for bulk cleanup |

## Threading

Messages form threads via `replies_to` dependencies. View a full thread:

```bash
issuegraph show msg-123 --thread
```

This traces the `replies_to` chain to find the root message, then collects all replies via BFS, displaying the conversation with proper indentation.

Thread display shows:
- Sender and recipient
- Timestamp
- Subject and body
- Reply depth (indented)

## Ephemeral Messages

Messages marked `ephemeral: true` are transient - they can be bulk-deleted after a swarm completes:

```bash
# Clean up closed ephemeral messages
issuegraph purge --force

# Preview what would be deleted
issuegraph purge --dry-run

# Only delete ephemeral messages older than 7 days
issuegraph purge --older-than 7d --force
```

Ephemeral messages are:
- Excluded from `issuegraph ready` by default
- Not synced to remotes (transient)
- Eligible for bulk deletion when closed

## Identity

The actor identity (used for `sender` on messages) is resolved in order:

1. `--actor` flag on the command
2. `BEADS_ACTOR` environment variable
3. `BD_ACTOR` environment variable (deprecated alias)
4. `git config user.name`
5. `$USER` environment variable
6. `"unknown"`

## IssueGraph Event Hooks

Scripts in `.beads/hooks/` run after certain events:

| Hook | Trigger |
|------|---------|
| `on_create` | After `issuegraph create` |
| `on_update` | After `issuegraph update` |
| `on_close` | After `issuegraph close` |

Hooks receive event data as JSON on stdin. This enables orchestrator integration (e.g., notifying services of new messages) without issuegraph knowing about the orchestrator.

Creates with initial labels preserve the legacy hook sequence: `on_create` receives the issue snapshot before labels, followed by one or more `on_update` events with cumulative label snapshots. The labels are already persisted before those hooks run, so hook scripts that need the create-time sequence should rely on the JSON payload instead of re-reading the issue from the store during the hook.

Batch creates with initial dependencies emit `on_update` for each persisted
dependency after the create-time hooks. Each payload carries a cumulative
dependency snapshot in request order, matching the event shape produced by
explicit post-create dependency additions; dependencies skipped because their
target was not persisted do not produce hooks.

## See Also

- [Graph Links](../docs/core-concepts/graph-links.md) - relates_to, duplicates, supersedes, replies_to
- [CLI Reference](../docs/CLI_REFERENCE.md) - All commands
