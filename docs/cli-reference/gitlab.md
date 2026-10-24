---
title: "issuegraph gitlab"
description: "Commands for syncing issues between issuegraph and GitLab."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc gitlab`.

Commands for syncing issues between issuegraph and GitLab.

Configuration can be set via 'issuegraph config' or environment variables:
  gitlab.url / GITLAB_URL                         - GitLab instance URL
  gitlab.token / GITLAB_TOKEN                     - Personal access token
  gitlab.project_id / GITLAB_PROJECT_ID           - Project ID or path
  gitlab.group_id / GITLAB_GROUP_ID               - Group ID for group-level sync
  gitlab.default_project_id / GITLAB_DEFAULT_PROJECT_ID - Project for creating issues in group mode

```
issuegraph gitlab [flags]
```

## issuegraph gitlab projects

List GitLab projects that the configured token has access to.

```
issuegraph gitlab projects [flags]
```

## issuegraph gitlab pull

Pull one or more items from GitLab.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph gitlab sync --pull-only --issues &lt;refs&gt;

```
issuegraph gitlab pull [refs...] [flags]
```

**Flags:**

```
      --dry-run   Preview pull without making changes
```

## issuegraph gitlab push

Push one or more issuegraph issues to GitLab.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph gitlab sync --push-only --issues &lt;ids&gt;

```
issuegraph gitlab push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

## issuegraph gitlab status

Display current GitLab configuration and sync status.

```
issuegraph gitlab status [flags]
```

## issuegraph gitlab sync

Synchronize issues between issuegraph and GitLab.

By default, performs bidirectional sync:
- Pulls new/updated issues from GitLab to issuegraph
- Pushes local issuegraph issues to GitLab

Use --pull-only or --push-only to limit direction.

```
issuegraph gitlab sync [flags]
```

**Flags:**

```
      --assignee string       Filter by assignee username
      --dry-run               Show what would be synced without making changes
      --exclude-type string   Exclude these issue types from sync (comma-separated)
      --issues string         Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --label string          Filter by labels (comma-separated, AND logic)
      --milestone string      Filter by milestone title
      --no-ephemeral          Exclude ephemeral/wisp issues from push (default: true) (default true)
      --parent string         Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-gitlab         On conflict, use GitLab version
      --prefer-local          On conflict, keep local issuegraph version
      --prefer-newer          On conflict, use most recent version (default)
      --project string        Filter to issues from this project ID (group mode)
      --pull-only             Only pull issues from GitLab
      --push-only             Only push issues to GitLab
      --type string           Only sync these issue types (comma-separated, e.g. 'epic,feature,task')
```
