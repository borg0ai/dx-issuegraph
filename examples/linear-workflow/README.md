# Linear Integration for issuegraph

Bidirectional synchronization between Linear and issuegraph (IssueGraph) using the built-in `issuegraph linear` commands.

## Overview

The Linear integration provides:

- **Pull**: Import issues from Linear into issuegraph
- **Push**: Export issuegraph issues to Linear
- **Bidirectional Sync**: Two-way sync with conflict resolution
- **Incremental Sync**: Only sync issues changed since last sync
- **Configurable Mappings**: Customize priority, state, label, and relation mappings

## Quick Start

### 1. Get Linear Credentials

1. **API Key**: Go to Linear → Settings → API → Personal API keys → Create key
2. **Team ID**: Go to Linear → Settings → General → find the Team ID (UUID format)

### 2. Configure issuegraph

```bash
# Set API key via environment variable (recommended — avoids git exposure)
export LINEAR_API_KEY="lin_api_YOUR_API_KEY_HERE"  # add to ~/.secrets or ~/.zshrc

# Set team ID
issuegraph config set linear.team_id "YOUR_TEAM_UUID"
```

### 3. Sync with Linear

```bash
# Check configuration status
issuegraph linear status

# Pull issues from Linear
issuegraph linear sync --pull

# Pull issues and Linear blocking relations as issuegraph dependencies
issuegraph linear sync --pull --relations

# Push local issues to Linear
issuegraph linear sync --push

# Full bidirectional sync (pull, resolve conflicts, push)
issuegraph linear sync
```

## Authentication

### API Key

Linear uses Personal API Keys for authentication. Create one at:
**Linear → Settings → API → Personal API keys**

Store securely:

```bash
# Recommended: Environment variable (avoids git exposure)
export LINEAR_API_KEY="lin_api_..."  # add to ~/.secrets or ~/.zshrc

# Alternative: issuegraph config (only if config.yaml is NOT git-tracked)
issuegraph config set linear.api_key "lin_api_..."
```

### Team ID

Find your Team ID in Linear:
- **Settings → General** → Look for Team ID
- Or extract from URLs: `https://linear.app/YOUR_TEAM/...` → Go to team settings

## Sync Modes

### Pull Only (Linear → issuegraph)

Import issues from Linear without pushing local changes:

```bash
issuegraph linear sync --pull

# Import Linear relations as issuegraph dependencies
issuegraph linear sync --pull --relations

# Filter by state
issuegraph linear sync --pull --state open    # Only open issues
issuegraph linear sync --pull --state closed  # Only closed issues
issuegraph linear sync --pull --state all     # All issues (default)

# Reconstruct Linear project milestones as local epic parents
issuegraph linear sync --pull --milestones
```

With `--milestones`, issuegraph creates or reuses one local epic per Linear
`projectMilestone`, then adds parent-child links from each pulled issue to its
milestone epic. Milestone epics are marked as Linear milestone records and are
skipped by later Linear pushes.

### Push Only (issuegraph → Linear)

Export local issues to Linear without pulling:

```bash
issuegraph linear sync --push

# Create only (don't update existing Linear issues)
issuegraph linear sync --push --create-only

# Disable automatic external_ref update
issuegraph linear sync --push --update-refs=false
```

### Bidirectional Sync

Full two-way sync with conflict detection and resolution:

```bash
# Default: newer timestamp wins conflicts
issuegraph linear sync

# Always prefer local version on conflicts
issuegraph linear sync --prefer-local

# Always prefer Linear version on conflicts
issuegraph linear sync --prefer-linear
```

### Dry Run

Preview what would happen without making changes:

```bash
issuegraph linear sync --dry-run
```

## Data Mapping

### Priority Mapping

Linear and IssueGraph use different priority semantics:

| Linear | Meaning | IssueGraph | Meaning |
|--------|---------|-------|---------|
| 0 | No priority | 4 | Backlog |
| 1 | Urgent | 0 | Critical |
| 2 | High | 1 | High |
| 3 | Medium | 2 | Medium |
| 4 | Low | 3 | Low |

**Default mapping** (Linear → IssueGraph):
- 0 (no priority) → 4 (backlog)
- 1 (urgent) → 0 (critical)
- 2 (high) → 1 (high)
- 3 (medium) → 2 (medium)
- 4 (low) → 3 (low)

**Custom mappings:**

```bash
# Override default mappings
issuegraph config set linear.priority_map.0 2    # No priority -> Medium (instead of Backlog)
issuegraph config set linear.priority_map.1 1    # Urgent -> High (instead of Critical)
```

### State Mapping

Map Linear workflow states to issuegraph statuses:

| Linear State Type | IssueGraph Status |
|-------------------|--------------|
| backlog | open |
| unstarted | open |
| started | in_progress |
| completed | closed |
| canceled | closed |

**Custom state mappings** (for custom workflow states):

```bash
# Map by state type
issuegraph config set linear.state_map.started in_progress

# Map by state name (for custom workflow states)
issuegraph config set linear.state_map.in_review in_progress
issuegraph config set linear.state_map.blocked blocked
issuegraph config set linear.state_map.on_hold blocked
issuegraph config set linear.state_map.testing in_progress
issuegraph config set linear.state_map.deployed closed
```

### Label to Issue Type

Infer issuegraph issue type from Linear labels:

| Linear Label | IssueGraph Type |
|--------------|------------|
| bug, defect | bug |
| feature, enhancement | feature |
| epic | epic |
| chore, maintenance | chore |
| task | task |

**Custom label mappings:**

```bash
issuegraph config set linear.label_type_map.incident bug
issuegraph config set linear.label_type_map.improvement feature
issuegraph config set linear.label_type_map.tech_debt chore
issuegraph config set linear.label_type_map.story feature
```

### Relation Mapping

Map Linear relations to issuegraph dependencies:

Relation import is opt-in during pull:

```bash
issuegraph linear sync --pull --relations
```

| Linear Relation | IssueGraph Dependency |
|-----------------|------------------|
| blocks | blocks |
| blockedBy | blocks (inverted) |
| duplicate | duplicates |
| related | related |
| (parent) | parent-child |

**Custom relation mappings:**

```bash
issuegraph config set linear.relation_map.causes discovered-from
issuegraph config set linear.relation_map.duplicate related
```

## Conflict Resolution

Conflicts occur when both local and Linear versions are modified since the last sync.

### Timestamp-based (Default)

The newer version wins:

```bash
issuegraph linear sync  # Newer timestamp wins
```

### Prefer Local

Local issuegraph version always wins:

```bash
issuegraph linear sync --prefer-local
```

Use when:
- Local is your source of truth
- You've made deliberate changes locally

### Prefer Linear

Linear version always wins:

```bash
issuegraph linear sync --prefer-linear
```

Use when:
- Linear is your source of truth
- You want to accept team changes

## Workflows

### Workflow 1: Initial Import from Linear

First-time import of existing Linear issues:

```bash
# Configure credentials
export LINEAR_API_KEY="lin_api_..."  # add to ~/.secrets or ~/.zshrc
issuegraph config set linear.team_id "team-uuid"

# Check status
issuegraph linear status

# Import all issues
issuegraph linear sync --pull

# See what was imported
issuegraph stats
issuegraph list --json
```

### Workflow 2: Daily Sync

Regular synchronization:

```bash
# Pull latest from Linear (incremental since last sync)
issuegraph linear sync --pull

# Do local work
issuegraph update bd-123 --claim
# ... work ...
issuegraph close bd-123 --reason "Fixed"

# Push changes to Linear
issuegraph linear sync --push

# Or do full bidirectional sync
issuegraph linear sync
```

### Workflow 3: Create Local Issues, Push to Linear

Create issues locally and sync to Linear:

```bash
# Create issue locally
issuegraph create "Fix authentication bug" -t bug -p 1

# Push to Linear (creates new Linear issue, updates external_ref)
issuegraph linear sync --push

# Verify
issuegraph show bd-abc  # Should have external_ref pointing to Linear
```

### Workflow 4: Migrate to issuegraph

Full migration from Linear to issuegraph:

```bash
# Import all issues
issuegraph linear sync --pull --state all

# Preview import
issuegraph stats

# Continue using issuegraph locally, push updates back to Linear
issuegraph linear sync  # Regular bidirectional sync
```

### Workflow 5: Read-Only Linear Mirror

Mirror Linear issues locally without pushing back:

```bash
# Only ever pull, never push
issuegraph linear sync --pull

# Set up a cron job or alias
alias bd-mirror="issuegraph linear sync --pull"
```

## Status & Debugging

### Check Sync Status

```bash
issuegraph linear status
```

Shows:
- Configuration status (API key, team ID)
- Last sync timestamp
- Issues with Linear links
- Issues pending push (local only)

### JSON Output

```bash
issuegraph linear status --json
issuegraph linear sync --json
```

### Verbose Output

The sync command shows progress:
- Number of issues pulled/pushed
- Conflicts detected and resolved
- Errors and warnings

## Configuration Reference

All configuration keys for Linear integration:

```bash
# Required
linear.api_key          # Linear API key (or LINEAR_API_KEY env var)
linear.team_id          # Linear team UUID

# Automatic (set by issuegraph)
linear.last_sync        # ISO8601 timestamp of last sync

# ID generation (optional)
linear.id_mode          # hash (default) or db (let issuegraph generate IDs)
linear.hash_length      # Hash length 3-8 (default: 6)

# Priority mapping (Linear 0-4 to IssueGraph 0-4)
linear.priority_map.0   # No priority -> ? (default: 4/backlog)
linear.priority_map.1   # Urgent -> ? (default: 0/critical)
linear.priority_map.2   # High -> ? (default: 1/high)
linear.priority_map.3   # Medium -> ? (default: 2/medium)
linear.priority_map.4   # Low -> ? (default: 3/low)

# State mapping (Linear state type/name to IssueGraph status)
linear.state_map.backlog     # (default: open)
linear.state_map.unstarted   # (default: open)
linear.state_map.started     # (default: in_progress)
linear.state_map.completed   # (default: closed)
linear.state_map.canceled    # (default: closed)
linear.state_map.<custom>    # Map custom state names

# Label to issue type mapping
linear.label_type_map.bug         # (default: bug)
linear.label_type_map.defect      # (default: bug)
linear.label_type_map.feature     # (default: feature)
linear.label_type_map.enhancement # (default: feature)
linear.label_type_map.epic        # (default: epic)
linear.label_type_map.chore       # (default: chore)
linear.label_type_map.maintenance # (default: chore)
linear.label_type_map.task        # (default: task)
linear.label_type_map.<custom>    # Map custom labels

# Relation mapping (Linear relation type to IssueGraph dependency type)
linear.relation_map.blocks    # (default: blocks)
linear.relation_map.blockedBy # (default: blocks)
linear.relation_map.duplicate # (default: duplicates)
linear.relation_map.related   # (default: related)
```

## Troubleshooting

### "Linear API key not configured"

Set the API key:

```bash
# Recommended: environment variable
export LINEAR_API_KEY="lin_api_YOUR_KEY"  # add to ~/.secrets or ~/.zshrc
```

### "Linear team ID not configured"

Set the team ID:

```bash
issuegraph config set linear.team_id "YOUR_TEAM_UUID"
```

### "GraphQL errors: Not authorized"

- Verify your API key is correct
- Check that the API key has access to the team
- Ensure the key hasn't been revoked

### "Rate limited"

Linear has API rate limits. The client automatically retries with exponential backoff:
- 3 retries with increasing delays
- If still failing, wait and retry later

### "Conflict detection failed"

- Check network connectivity
- Verify API key permissions
- Check `issuegraph linear status` for configuration issues

### Sync seems slow

For large projects, initial sync fetches all issues. Subsequent syncs are incremental (only issues changed since `linear.last_sync`).

## Limitations

- **Single team**: Sync is configured per-team (one team_id per issuegraph project)
- **No attachments**: Attachments are not synced
- **No comments**: Comments are not synced (only description)
- **Custom fields**: Linear custom fields are not mapped
- **Projects**: Linear projects are not mapped (use labels for categorization)
- **Cycles**: Linear cycles/sprints are not mapped

## See Also

- [CONFIG.md](../../docs/reference/configuration.md) - Full configuration documentation
- [Jira Sync](../../README.md) - Similar integration for Jira (`issuegraph jira sync`)
- [Linear GraphQL API](https://developers.linear.app/docs/graphql/working-with-the-graphql-api)

---

## Example Session

```bash
# Initial setup
$ issuegraph init --quiet
$ export LINEAR_API_KEY="lin_api_abc123..."  # add to ~/.secrets or ~/.zshrc
$ issuegraph config set linear.team_id "team-uuid-456"

# Check status
$ issuegraph linear status
Linear Sync Status
==================

Team ID:      team-uuid-456
API Key:      lin_...c123
Last Sync:    Never

Total Issues: 0
With Linear:  0
Local Only:   0

# Pull from Linear
$ issuegraph linear sync --pull
→ Pulling issues from Linear...
  Full sync (no previous sync timestamp)
✓ Pulled 47 issues (47 created, 0 updated)

✓ Linear sync complete

# Check what we got
$ issuegraph stats
Issues: 47 (42 open, 5 closed)
Types:  23 task, 15 bug, 7 feature, 2 epic

# Create a local issue
$ issuegraph create "New bug from testing" -t bug -p 1
Created: bd-a1b2c3

# Push to Linear
$ issuegraph linear sync --push
→ Pushing issues to Linear...
  Created: bd-a1b2c3 -> TEAM-148
✓ Pushed 1 issues (1 created, 0 updated)

✓ Linear sync complete

# Full bidirectional sync
$ issuegraph linear sync
→ Pulling issues from Linear...
  Incremental sync since 2025-01-17 10:30:00
✓ Pulled 3 issues (0 created, 3 updated)
→ Pushing issues to Linear...
✓ Pushed 2 issues (0 created, 2 updated)

✓ Linear sync complete
```
