---
title: "issuegraph ado"
description: "Commands for syncing issues between issuegraph and Azure DevOps."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc ado`.

Commands for syncing issues between issuegraph and Azure DevOps.

Configuration can be set via 'issuegraph config' or environment variables:
  ado.org / AZURE_DEVOPS_ORG              - Organization name
  ado.project / AZURE_DEVOPS_PROJECT      - Project name (single)
  ado.projects / AZURE_DEVOPS_PROJECTS    - Project names (comma-separated)
  ado.pat / AZURE_DEVOPS_PAT              - Personal access token
  ado.url / AZURE_DEVOPS_URL              - Custom base URL (on-prem)

```
issuegraph ado [flags]
```

## issuegraph ado projects

List Azure DevOps projects that the configured token has access to.

```
issuegraph ado projects [flags]
```

## issuegraph ado pull

Pull one or more items from Azure DevOps.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph ado sync --pull-only --issues &lt;refs&gt;

```
issuegraph ado pull [refs...] [flags]
```

**Flags:**

```
      --dry-run   Preview pull without making changes
```

## issuegraph ado push

Push one or more issuegraph issues to Azure DevOps.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph ado sync --push-only --issues &lt;ids&gt;

```
issuegraph ado push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

## issuegraph ado status

Display current Azure DevOps configuration and sync status.

```
issuegraph ado status [flags]
```

## issuegraph ado sync

Synchronize issues between issuegraph and Azure DevOps.

By default, performs bidirectional sync:
- Pulls new/updated work items from Azure DevOps to issuegraph
- Pushes local issuegraph issues to Azure DevOps

Use --pull-only or --push-only to limit direction.

Filters (--area-path, --iteration-path, --types, --states) restrict
which work items are synced. On pull, they limit the WIQL query. On push,
--types and --states filter local issuegraph before pushing to ADO. Use
--no-create with push to skip creating new ADO work items (only update
existing linked items). Filters can also be persisted via config:
  ado.filter.area_path, ado.filter.iteration_path,
  ado.filter.types, ado.filter.states
CLI flags override config values when both are set.

```
issuegraph ado sync [flags]
```

**Flags:**

```
      --area-path string        Filter to ADO area path (e.g., "Project\Team")
      --bootstrap-match         Enable heuristic matching for first sync
      --dry-run                 Show what would be synced without making changes
      --issues string           Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --iteration-path string   Filter to ADO iteration path (e.g., "Project\Sprint 1")
      --no-create               Never create new items in either direction (pull or push)
      --parent string           Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-ado              On conflict, use Azure DevOps version
      --prefer-local            On conflict, keep local issuegraph version
      --prefer-newer            On conflict, use most recent version (default)
      --project strings         Project name(s) to sync (overrides configured project/projects)
      --pull-only               Only pull issues from Azure DevOps
      --push-only               Only push issues to Azure DevOps
      --reconcile               Force reconciliation scan for deleted items
      --states string           Filter to ADO states, comma-separated (e.g., "New,Active,Resolved")
      --types string            Filter to work item types, comma-separated (e.g., "Bug,Task,User Story")
```
