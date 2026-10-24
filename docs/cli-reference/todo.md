---
title: "issuegraph todo"
description: "Manage TODO items as lightweight task issues."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc todo`.

Manage TODO items as lightweight task issues.

TODOs are regular task-type issues with convenient shortcuts:
  issuegraph todo add "Title"    -&gt; issuegraph create "Title" -t task -p 2
  issuegraph todo                -&gt; issuegraph list --type task --status open
  issuegraph todo done &lt;id&gt;      -&gt; issuegraph close &lt;id&gt;

TODOs can be promoted to full issues by changing type or priority:
  issuegraph update todo-123 --type bug --priority 0

```
issuegraph todo [flags]
```

## issuegraph todo add

Add a new TODO item

```
issuegraph todo add <title> [flags]
```

**Flags:**

```
  -d, --description string   Description
  -p, --priority int         Priority (0-4, default 2) (default 2)
```

## issuegraph todo done

Mark TODO(s) as done

```
issuegraph todo done <id> [<id>...] [flags]
```

**Flags:**

```
      --reason string   Reason for closing (default: Completed)
```

## issuegraph todo list

List TODO items

```
issuegraph todo list [flags]
```

**Flags:**

```
      --all   Show all TODOs including completed
```
