---
title: "issuegraph github"
description: "GitHub integration commands"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc github`.

Commands for syncing issues between issuegraph and GitHub.

Repository can be set via 'issuegraph config' or environment variables:
  github.owner / GITHUB_OWNER           - Repository owner
  github.repo / GITHUB_REPO             - Repository name
  github.repository / GITHUB_REPOSITORY - Combined "owner/repo" format
  github.url / GITHUB_API_URL           - Custom API URL (GitHub Enterprise)

Authenticate with 'gh auth login' or set GITHUB_TOKEN. IssueGraph does not
store GitHub tokens. Local commands never contact GitHub.

```
issuegraph github [command]
```

## issuegraph github pull

Pull one or more items from GitHub.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph github sync --pull-only --issues &lt;refs&gt;

```
issuegraph github pull [refs...] [flags]
```

**Aliases:** import

**Flags:**

```
      --apply     Apply planned local changes
      --dry-run   Preview pull without applying changes
```

## issuegraph github push

Push one or more issuegraph issues to GitHub.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph github sync --push-only --issues &lt;ids&gt;

```
issuegraph github push [issue-ids...] [flags]
```

**Aliases:** export

**Flags:**

```
      --apply     Apply planned GitHub changes
      --dry-run   Preview push without applying changes
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

By default, previews bidirectional sync without writing:
- Pulls new/updated issues from GitHub to issuegraph
- Pushes local issuegraph issues to GitHub

Use --pull-only or --push-only to limit direction. Pass --apply to write changes.
Conflicts preserve both current versions and are reported.

```
issuegraph github sync [flags]
```

**Flags:**

```
      --apply           Apply planned local and GitHub changes
      --dry-run         Preview changes without applying them
      --issues string   Comma-separated issue IDs to sync selectively (e.g., bd-abc,bd-def). Mutually exclusive with --parent.
      --parent string   Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-github   On conflict, use GitHub version
      --prefer-local    On conflict, keep local issuegraph version
      --pull-only       Only pull issues from GitHub
      --push-only       Only push issues to GitHub
```
