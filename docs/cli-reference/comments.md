---
title: "issuegraph comments"
description: "View or manage comments on an issue."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc comments`.

View or manage comments on an issue.

Examples:
  # List all comments on an issue (issue id is required — there is no "comments list")
  issuegraph comments issuegraph-123

  # List comments in JSON format
  issuegraph comments issuegraph-123 --json

  # Add a comment
  issuegraph comments add issuegraph-123 "This is a comment"

  # Add a comment from a file
  issuegraph comments add issuegraph-123 -f notes.txt

```
issuegraph comments [issue-id] [flags]
```

**Flags:**

```
      --local-time   Show timestamps in local time instead of UTC
```

## issuegraph comments add

Add a comment to an issue.

Examples:
  # Add a comment
  issuegraph comments add issuegraph-123 "Working on this now"

  # Add a comment from a file
  issuegraph comments add issuegraph-123 -f notes.txt

```
issuegraph comments add [issue-id] [text] [flags]
```

**Flags:**

```
  -a, --author string   Add author to comment
  -f, --file string     Read comment text from file
```

## issuegraph comments list

Invalid — use issuegraph comments &lt;issue-id&gt; to list comments

```
issuegraph comments list [flags]
```
