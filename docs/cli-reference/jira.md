---
title: "issuegraph jira"
description: "Synchronize issues between issuegraph and Jira."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc jira`.

Synchronize issues between issuegraph and Jira.

Configuration:
  issuegraph config set jira.url "https://company.atlassian.net"
  issuegraph config set jira.project "PROJ"
  issuegraph config set jira.projects "PROJ1,PROJ2"   # Multiple projects
  issuegraph config set jira.api_token "YOUR_TOKEN"
  issuegraph config set jira.username "your_email@company.com"  # For Jira Cloud
  issuegraph config set jira.push_prefix "hippo"       # Only push hippo-* issues to Jira
  issuegraph config set jira.push_prefix "proj1,proj2" # Multiple prefixes (comma-separated)

Environment variables (alternative to config):
  JIRA_API_TOKEN  - Jira API token
  JIRA_USERNAME   - Jira username/email
  JIRA_PROJECTS   - Comma-separated project keys

Examples:
  issuegraph jira sync --pull         # Import issues from Jira
  issuegraph jira sync --push         # Export issues to Jira
  issuegraph jira sync                # Bidirectional sync (pull then push)
  issuegraph jira sync --dry-run      # Preview sync without changes
  issuegraph jira status              # Show sync status

```
issuegraph jira [flags]
```

## issuegraph jira pull

Pull one or more items from Jira.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph jira sync --pull --issues &lt;refs&gt;

```
issuegraph jira pull [refs...] [flags]
```

**Flags:**

```
      --dry-run   Preview pull without making changes
```

## issuegraph jira push

Push one or more issuegraph issues to Jira.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph jira sync --push --issues &lt;ids&gt;

```
issuegraph jira push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

## issuegraph jira status

Show the current Jira sync status, including:
  - Last sync timestamp
  - Configuration status
  - Number of issues with Jira links
  - Issues pending push (no external_ref)

```
issuegraph jira status [flags]
```

## issuegraph jira sync

Synchronize issues between issuegraph and Jira.

Modes:
  --pull         Import issues from Jira into issuegraph
  --push         Export issues from issuegraph to Jira
  (no flags)     Bidirectional sync: pull then push, with conflict resolution

Conflict Resolution:
  By default, newer timestamp wins. Override with:
  --prefer-local   Always prefer local issuegraph version
  --prefer-jira    Always prefer Jira version

Examples:
  issuegraph jira sync --pull                # Import from Jira
  issuegraph jira sync --push --create-only  # Push new issues only
  issuegraph jira sync --dry-run             # Preview without changes
  issuegraph jira sync --prefer-local        # Bidirectional, local wins

```
issuegraph jira sync [flags]
```

**Flags:**

```
      --create-only       Only create new issues, don't update existing
      --dry-run           Preview sync without making changes
      --issues string     Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --parent string     Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-jira       Prefer Jira version on conflicts
      --prefer-local      Prefer local version on conflicts
      --project strings   Project key(s) to sync (overrides configured project/projects)
      --pull              Pull issues from Jira
      --push              Push issues to Jira
      --state string      Issue state to sync: open, closed, all (default "all")
```
