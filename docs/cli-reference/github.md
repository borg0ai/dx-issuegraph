---
title: "issuegraph github"
description: "Commands for syncing issues between issuegraph and GitHub."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc github`.

Commands for syncing issues between issuegraph and GitHub.

Configuration can be set via 'issuegraph config' or environment variables:
  github.token / GITHUB_TOKEN           - Personal access token
  github.owner / GITHUB_OWNER           - Repository owner
  github.repo / GITHUB_REPO             - Repository name
  github.repository / GITHUB_REPOSITORY - Combined "owner/repo" format
  github.url / GITHUB_API_URL           - Custom API URL (GitHub Enterprise)

```
issuegraph github [flags]
```

## issuegraph github pull

Pull one or more items from GitHub.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph github sync --pull-only --issues &lt;refs&gt;

```
issuegraph github pull [refs...] [flags]
```

**Flags:**

```
      --dry-run   Preview pull without making changes
```

## issuegraph github push

Push one or more issuegraph issues to GitHub.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph github sync --push-only --issues &lt;ids&gt;

```
issuegraph github push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

## issuegraph github repos

List GitHub repositories that the configured token has access to.

```
issuegraph github repos [flags]
```

## issuegraph github status

Display current GitHub configuration and sync status.

```
issuegraph github status [flags]
```

## issuegraph github sync

Synchronize issues between issuegraph and GitHub.

By default, performs bidirectional sync:
- Pulls new/updated issues from GitHub to issuegraph
- Pushes local issuegraph issues to GitHub

Use --pull-only or --push-only to limit direction.

```
issuegraph github sync [flags]
```

**Flags:**

```
      --dry-run         Show what would be synced without making changes
      --issues string   Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --parent string   Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-github   On conflict, use GitHub version
      --prefer-local    On conflict, keep local issuegraph version
      --prefer-newer    On conflict, use most recent version (default)
      --pull-only       Only pull issues from GitHub
      --push-only       Only push issues to GitHub
```
