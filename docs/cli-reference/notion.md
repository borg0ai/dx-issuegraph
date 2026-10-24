---
title: "issuegraph notion"
description: "Commands for syncing issues between issuegraph and Notion."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc notion`.

Commands for syncing issues between issuegraph and Notion.

```
issuegraph notion [flags]
```

## issuegraph notion connect

Connect issuegraph to an existing Notion database or data source

```
issuegraph notion connect [flags]
```

**Flags:**

```
      --url string   Existing Notion database or data source URL
```

## issuegraph notion init

Create a dedicated IssueGraph database in Notion

```
issuegraph notion init [flags]
```

**Flags:**

```
      --parent string   Parent page ID
      --title string    Database title (default "IssueGraph Issues")
```

## issuegraph notion pull

Pull one or more items from Notion.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph notion sync --pull --issues &lt;refs&gt;

```
issuegraph notion pull [refs...] [flags]
```

**Flags:**

```
      --dry-run   Preview pull without making changes
```

## issuegraph notion push

Push one or more issuegraph issues to Notion.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph notion sync --push --issues &lt;ids&gt;

```
issuegraph notion push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

## issuegraph notion status

Show Notion sync status

```
issuegraph notion status [flags]
```

## issuegraph notion sync

Synchronize issues between issuegraph and Notion.

By default this performs bidirectional sync. Use --pull or --push to limit direction.

```
issuegraph notion sync [flags]
```

**Flags:**

```
      --create-only     Only create missing remote pages, do not update existing ones
      --dry-run         Preview changes without making mutations
      --issues string   Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --parent string   Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-local    On conflict, keep the local issuegraph version
      --prefer-notion   On conflict, use the Notion version
      --pull            Only pull issues from Notion
      --push            Only push issues to Notion
      --state string    Issue state to sync: open, closed, or all (default "all")
```
