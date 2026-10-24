---
title: "issuegraph linear"
description: "Synchronize issues between issuegraph and Linear."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc linear`.

Synchronize issues between issuegraph and Linear.

Configuration:
  issuegraph config set linear.api_key "YOUR_API_KEY"
  issuegraph config set linear.team_id "TEAM_ID"
  issuegraph config set linear.team_ids "TEAM_ID1,TEAM_ID2"  # Multiple teams (comma-separated)
  issuegraph config set linear.project_id "PROJECT_ID"  # Optional: sync only this project

Environment variables (alternative to config):
  LINEAR_API_KEY  - Linear API key (for individual developers)
  LINEAR_TEAM_ID  - Linear team ID (UUID, singular)
  LINEAR_TEAM_IDS - Linear team IDs (comma-separated UUIDs)

OAuth (for CI workers / automated sync):
  LINEAR_OAUTH_CLIENT_ID     - OAuth app client ID
  LINEAR_OAUTH_CLIENT_SECRET - OAuth app client secret

  When both OAuth env vars are set, OAuth client_credentials flow is used
  instead of the API key. This allows CI workers to authenticate as an
  application (actor=application) rather than impersonating a user.
  Precedence: OAuth &gt; LINEAR_API_KEY &gt; config file.

Data Mapping (optional, sensible defaults provided):
  Priority mapping (Linear 0-4 to IssueGraph 0-4):
    issuegraph config set linear.priority_map.0 4    # No priority -&gt; Backlog
    issuegraph config set linear.priority_map.1 0    # Urgent -&gt; Critical
    issuegraph config set linear.priority_map.2 1    # High -&gt; High
    issuegraph config set linear.priority_map.3 2    # Medium -&gt; Medium
    issuegraph config set linear.priority_map.4 3    # Low -&gt; Low

  State mapping (Linear state type to IssueGraph status):
    issuegraph config set linear.state_map.backlog open
    issuegraph config set linear.state_map.unstarted open
    issuegraph config set linear.state_map.started in_progress
    issuegraph config set linear.state_map.completed closed
    issuegraph config set linear.state_map.canceled closed
    issuegraph config set linear.state_map.my_custom_state in_progress  # Custom state names

  Label to issue type mapping:
    issuegraph config set linear.label_type_map.bug bug
    issuegraph config set linear.label_type_map.feature feature
    issuegraph config set linear.label_type_map.epic epic

  Relation type mapping (Linear relations to IssueGraph dependencies):
    issuegraph config set linear.relation_map.blocks blocks
    issuegraph config set linear.relation_map.blockedBy blocks
    issuegraph config set linear.relation_map.duplicate duplicates
    issuegraph config set linear.relation_map.related related

  ID generation (optional, hash IDs to match issuegraph/Jira hash mode):
    issuegraph config set linear.id_mode "hash"      # hash (default)
    issuegraph config set linear.hash_length "6"     # hash length 3-8 (default: 6)

Examples:
  issuegraph linear sync --pull         # Import issues from Linear
  issuegraph linear sync --push         # Export issues to Linear
  issuegraph linear sync                # Bidirectional sync (pull then push)
  issuegraph linear sync --dry-run      # Preview sync without changes
  issuegraph create "Fix login" --external-ref https://linear.app/team/issue/TEAM-123
                              # Link a local issue to an existing Linear issue
  issuegraph linear status              # Show sync status

```
issuegraph linear [flags]
```

## issuegraph linear pull

Pull one or more items from Linear.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph linear sync --pull --issues &lt;refs&gt;

```
issuegraph linear pull [refs...] [flags]
```

**Flags:**

```
      --dry-run     Preview pull without making changes
      --relations   Import Linear relations as issuegraph dependencies when pulling
```

## issuegraph linear push

Push one or more issuegraph issues to Linear.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph linear sync --push --issues &lt;ids&gt;

```
issuegraph linear push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

## issuegraph linear status

Show the current Linear sync status, including:
  - Last sync timestamp
  - Configuration status
  - Number of issues with Linear links
  - Issues pending push (no external_ref)

```
issuegraph linear status [flags]
```

## issuegraph linear sync

Synchronize issues between issuegraph and Linear.

Modes:
  --pull              Import issues from Linear into issuegraph
  --push              Export issues from issuegraph to Linear
  --pull-if-stale     Pull only if data is stale (skip if fresh)
  (no flags)          Bidirectional sync: pull then push, with conflict resolution

Staleness (--pull-if-stale):
  --threshold 20m     How old data must be before pulling (default 20m)
  A 5-minute debounce prevents agent loops: if a pull completed within 5 minutes,
  data is always treated as fresh regardless of the threshold.

Team Selection:
  --team ID1,ID2  Override configured team IDs for this sync
  Multiple teams can be configured via linear.team_ids (comma-separated).
  Falls back to linear.team_id for backward compatibility.
  Push requires explicit --team when multiple teams are configured.

Pull Options:
  --milestones       Reconstruct Linear project milestones as local epic parents

Type Filtering (--push only):
  --type task,feature       Only sync issues of these types
  --exclude-type wisp       Exclude issues of these types
  --include-ephemeral       Include ephemeral issues (wisps, etc.); default is to exclude
  --parent TICKET           Only push this ticket and its descendants
  --relations               Import Linear relations as issuegraph dependencies on pull

Conflict Resolution:
  By default, newer timestamp wins. Override with:
  --prefer-local    Always prefer local issuegraph version
  --prefer-linear   Always prefer Linear version

Examples:
  issuegraph linear sync --pull                         # Import from Linear
  issuegraph linear sync --pull-if-stale                # Pull only if data is stale
  issuegraph linear sync --pull-if-stale --threshold 5m # Pull if older than 5 minutes
  issuegraph linear sync --pull --relations             # Import Linear blocking relations as issuegraph deps
  issuegraph linear sync --push --create-only           # Push new issues only
  issuegraph linear sync --push --type=task,feature     # Push only tasks and features
  issuegraph linear sync --push --exclude-type=wisp     # Push all except wisps
  issuegraph linear sync --push --parent=issuegraph-abc123      # Push one ticket tree
  issuegraph linear sync --dry-run                      # Preview without changes
  issuegraph linear sync --prefer-local                 # Bidirectional, local wins

```
issuegraph linear sync [flags]
```

**Flags:**

```
      --create-only            Only create new issues, don't update existing
      --dry-run                Preview sync without making changes
      --exclude-type strings   Exclude issues of these types (can be repeated)
      --include-ephemeral      Include ephemeral issues (wisps, etc.) when pushing to Linear
      --issues string          Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --milestones             Reconstruct Linear project milestones as local epic parents when pulling
      --no-wait                Fail immediately if another sync is running instead of waiting
      --parent string          Limit push to this issuegraph ticket and its descendants
      --prefer-linear          Prefer Linear version on conflicts
      --prefer-local           Prefer local version on conflicts
      --pull                   Pull issues from Linear
      --pull-if-stale          Pull only if Linear data is stale (skip if fresh)
      --push                   Push issues to Linear
      --relations              Import Linear relations as issuegraph dependencies when pulling
      --state string           Issue state to sync: open, closed, all (default "all")
      --team strings           Team ID(s) to sync (overrides configured team_id/team_ids)
      --threshold duration     Staleness threshold for --pull-if-stale (default 20m) (default 20m0s)
      --type strings           Only sync issues of these types (can be repeated)
      --update-refs            Update external_ref after creating Linear issues (default true)
```

## issuegraph linear teams

List all teams accessible with your Linear API key.

Use this to find the team ID (UUID) needed for configuration.

Example:
  issuegraph linear teams
  issuegraph config set linear.team_id "12345678-1234-1234-1234-123456789abc"

```
issuegraph linear teams [flags]
```
