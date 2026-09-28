# Project Charter

This document defines the product boundary for issuegraph. It is the source of
truth for deciding whether proposed work belongs in core, belongs in an
integration or plugin, belongs in an orchestration layer, or should stay
outside the project.

IssueGraph is a focused issue tracker for AI-supervised development. It should stay
small enough to remain reliable, understandable, and composable.

## Core Scope

IssueGraph owns local issue tracking primitives:

- issues and issue lifecycle
- dependency relationships and readiness
- labels, comments, status, priority, and assignment
- metadata attached to issues
- local CLI workflows around those concepts
- local import, export, backup, and recovery for issue data
- local issue workflows and Git-tracked snapshot recovery

Within those boundaries, the project should absorb useful contributor work
when practical. If a contribution has value but does not fit as submitted,
prefer preserving the value by simplifying it, moving it to metadata, routing
it to an integration or plugin, cherry-picking the reusable part, or
reimplementing the use case in a smaller design.

## Orchestration Boundary

IssueGraph should not know about orchestration layers built on top of it. Systems
such as schedulers, swarms, release coordinators, and future
workflow engines may use issuegraph, but issuegraph should not encode their concepts in
core.

Core issuegraph can expose stable issue data, metadata, CLI output, and documented
extension points. The orchestration layer owns orchestration policy: agent
routing, task assignment strategy, model choice, retry plans, scheduling,
workflow semantics, and cross-system coordination.

When orchestration needs extra per-issue data, prefer issue metadata before
adding first-class fields or commands.

## Storage Boundary

IssueGraph should not become a storage engine. Dolt provides local storage,
versioning, concurrency, and crash safety. IssueGraph reads and writes through
the storage boundary. IssueGraph does not connect to Dolt remotes or external
trackers to synchronize issue data. A complete snapshot in the user's Git
working tree is the only portable issue-data copy; Git handles commit, pull,
and push when the user chooses.

Storage-engine details should not leak into issuegraph packages unless they are part
of a deliberate storage interface. Avoid issuegraph-side flocks, engine
introspection, storage-specific retry loops, crash-recovery workarounds, or
schema poking that belongs in Dolt or the Dolt driver.

If the current storage interface cannot express a needed operation, widen the
interface or route the issue to the driver instead of embedding storage-engine
logic in core.

This boundary is mechanically enforced for non-test code by a `depguard`
rule in `.golangci.yml` that denies `github.com/dolthub/` imports outside
`internal/storage/` and `internal/doltserver/` (the linter config does not
analyze `_test.go` files). The rule's `files` list documents the only
justified exceptions (the proxied-server surface and DoltHub's `eventkit`
telemetry client) alongside why each one is allowed.

## Schema Boundary

The database schema is considered stable. Schema changes are allowed when there
is a pressing product or correctness need, but they should not be the first
answer to extension requests.

Use issue metadata first when:

- the data is specific to one integration, orchestrator, or team workflow
- the data is advisory rather than part of issuegraph's core issue model
- the data can be represented as JSON without harming queryability
- the shape may evolve before it deserves a stable CLI or schema contract

Promote metadata to first-class schema only when the field has broad, durable
meaning for issuegraph itself and the migration cost is justified.

## Integration Boundary

IssueGraph has no built-in network tracker integration. Import/export operates
on local files. Users who need GitHub, GitLab, Jira, Linear, Azure DevOps, or
similar services can use those clients independently and manage local
IssueGraph snapshots with Git.

## Review Posture

These boundaries are fences, not bounce messages. Maintainers should not reject
useful work reflexively just because the first version crosses a boundary.

For pull requests and proposals:

- identify the contributor value first
- keep the part that belongs in core when possible
- move boundary-crossing behavior to external tools when that preserves the
  use case
- preserve attribution when transforming, cherry-picking, or reimplementing
  contributor work
- explain clearly when a feature belongs outside IssueGraph

Use request-changes or rejection only after considering whether the project can
absorb, transform, or reroute the useful part.

## Related Documents

- [Issue Metadata](../docs/core-concepts/metadata.md) - metadata extension point
- [Architecture](../docs/architecture/index.md) - data model and storage architecture
- [Maintainer PR Guidelines](../PR_MAINTAINER_GUIDELINES.md) - PR triage and
  contributor handling
