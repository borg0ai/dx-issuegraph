---
title: "issuegraph label"
description: "Manage issue labels"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc label`.

Manage issue labels

```
issuegraph label [flags]
```

## issuegraph label add

Add a label to one or more issues

```
issuegraph label add [issue-id...] [label] [flags]
```

## issuegraph label list

List labels for an issue

```
issuegraph label list [issue-id] [flags]
```

## issuegraph label list-all

List all unique labels in the database

```
issuegraph label list-all [flags]
```

## issuegraph label propagate

Push a label from a parent down to all direct children that don't already have it. Useful for applying branch: labels across an epic's subtasks.

```
issuegraph label propagate [parent-id] [label] [flags]
```

## issuegraph label remove

Remove a label from one or more issues

```
issuegraph label remove [issue-id...] [label] [flags]
```
