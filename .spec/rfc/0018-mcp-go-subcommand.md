# RFC 0018: Move the MCP server into the Go CLI (child of 0001)

**Status:** Draft

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

Make the MCP server a subcommand of the Go CLI, `issuegraph mcp`, and delete
the Python `issuegraph-mcp` package. IssueGraph becomes a single-language
product: one Go binary, delivered through the three channels in
[0017](0017-shipify-channels-only.md), with MCP included.

## Problem

`integrations/issuegraph-mcp` is a Python package (`fastmcp`) added in October
2025 when the project was Beads. It is a thin layer: every tool shells out to
the `issuegraph` CLI and holds no storage logic of its own. Keeping it means a
second language, a second toolchain (`uv`, `hatchling`, `pytest`, `mypy`), a
second version to keep in step with the binary, and a PyPI release path that
0017 removes. Without PyPI it has no delivery route at all.

## Goals

- `issuegraph mcp` runs an MCP server over stdio and exposes the same tool set
  the Python server exposes today (`discover_tools`, `get_tool_info`, `context`,
  `ready`, `list`, `show`, `create`, `claim`, `update`, `close`, `reopen`, and
  the rest), plus the `issuegraph://quickstart` resource.
- Tools call the CLI's internal packages directly instead of spawning
  `issuegraph` subprocesses, with identical arguments, results, and error
  semantics.
- Preserve the Python server's behavior that agents rely on: workspace
  selection and multi-project switching, compaction and context-size limits,
  and logs written to stderr so stdout stays clean protocol.
- Version, build, and release come from the existing Go pipeline; the MCP
  server needs no separate package or version.
- Delete `integrations/issuegraph-mcp` and every reference to it once the Go
  server passes the ported tests.

## Non-goals

- Adding, removing, or renaming MCP tools. Parity first; changes are separate
  RFCs.
- Network transports (HTTP, SSE). Stdio only, as today.
- Changing how any other CLI command behaves.
- Publishing anywhere other than the three channels in 0017.
- Unpublishing the existing `issuegraph-mcp` package from PyPI; that registry
  action needs separate authorization.
- Docker as a build, release, or runtime dependency.

## Design

Add the command under `apps/cli`, with the tool handlers in a package that
calls the same internals the CLI commands use. Choose the Go MCP SDK during
implementation; none is in `go.mod` yet, so the choice and its license are
recorded in the pull request, preferring the official SDK if it covers stdio
tools and resources.

Port in two steps so parity can be checked: first the tool schemas and result
shapes, captured from the running Python server as golden files; then the
handlers, tested against those golden files. The Python tests in
`integrations/issuegraph-mcp/tests` (client, config, compaction, lifecycle,
multi-project switching, server integration) are the behavior checklist; each
one is ported to Go or recorded as intentionally dropped with a reason.

Python is removed only after parity passes. Until then the Python package stays
in the repository, unpublished, and is not advertised. Removal covers the
package directory, `scripts/ci/package-mcp.sh`, the MCP lines in
`scripts/check-versions.sh`, `scripts/update-versions.sh`,
`scripts/ci/detect-package-gates.sh`, the release workflow, and the docs and
README sections that tell users to install it (`docs/integrations/mcp-server.md`,
`docs/getting-started/ide-setup.md`, `examples/claude-desktop-mcp`, and
others found by grep). Client config examples change to run
`issuegraph mcp` instead of `issuegraph-mcp`.

## Acceptance

- `issuegraph mcp` starts, completes the MCP initialize handshake over stdio,
  and lists the same tools as the Python server did.
- Each tool returns results equal to the captured golden files for the same
  inputs; the ported test suite passes under `go test`.
- Nothing is written to stdout except protocol messages.
- A Claude Desktop config using `issuegraph mcp` can list and call tools
  against a real workspace.
- `integrations/issuegraph-mcp` is gone, and grep for `issuegraph-mcp`,
  `fastmcp`, and `uv tool install` finds nothing outside `CHANGELOG.md`.
- No Python toolchain is needed to build, test, or release the repository.
- `sync-check` passes, and no registry package is unpublished or modified.
