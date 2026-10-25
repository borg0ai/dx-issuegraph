---
title: "issuegraph label"
description: "Manage issue labels"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc label`.

Manage issue labels

```
issuegraph label [command]
```

## issuegraph label add

Add labels to issues. Issue IDs come first; the final argument is the label. Pass multiple labels comma-separated: issuegraph label add bd-123 label1,label2

```
issuegraph label add [issue-id...] [label[,label...]] [flags]
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

Remove labels from issues. Issue IDs come first; the final argument is the label. Pass multiple labels comma-separated: issuegraph label remove bd-123 label1,label2

```
issuegraph label remove [issue-id...] [label[,label...]] [flags]
```
