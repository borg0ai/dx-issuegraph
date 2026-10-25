# RFC 0013: Move Go CLI into apps/cli (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

After RFC 0012 removes the npm CLI package, move the native Go CLI module and
its source into `apps/cli`. Preserve the core Go module under `modules/core`
and retain the CLI's Go `internal` package access through a repository-nested
module path.

## Problem

The approved RFCs 0006 and 0009 place a TypeScript/npm CLI package in
`apps/cli` and Go CLI source in `modules/cli`. RFC 0012 removes the npm CLI
package, leaving the native CLI split away from the product's primary app
path. Keeping the executable implementation under `apps/cli` gives builds,
release automation, and source ownership one canonical CLI location.

## Goals

- Move Go CLI source, tests, generated assets, and `go.mod`/`go.sum` from
  `modules/cli` to `apps/cli` after RFC 0012 removes the npm package.
- Set the CLI module path to
  `github.com/borg0ai/dx-issuegraph/apps/cli` so it remains under the parent
  path required to import `modules/core/internal/*` packages.
- Keep the public core module path and physical location unchanged.
- Update root `go.work`, build scripts, CI, GoReleaser, release workflow,
  tests, and documentation to use `apps/cli`.
- Remove the old `modules/cli` tree and npm-only files; provide no legacy
  module or command aliases.
- Delete the root `go.mod`. The layout rule is: `apps/` holds exported
  binaries (CLI, app, daemon), `modules/` holds packages and submodules. Every
  Go module lives under one of them, and `go.work` lists only those modules.

## Non-goals

- Choosing or changing the release channel; RFC 0012 owns Homebrew delivery.
- Moving reusable Go packages out of `modules/core`.
- Changing CLI behavior, command identity, issue storage, or schema.
- Removing pnpm/Turbo if other repository workspaces still use them.
- Introducing Docker as a source, build, or CLI runtime dependency.

## Design

Complete RFC 0012 first. Move the Go CLI module into `apps/cli`, preserving
package-relative paths where practical. Update its `go.mod` module path and
all imports of the CLI module; keep the core module path at
`github.com/borg0ai/dx-issuegraph` so public imports remain stable and the
CLI's new module path stays within Go's `internal` visibility boundary.

Update `go.work` to use `./apps/cli` and remove `./modules/cli`. Redirect all
build and test entry points to `apps/cli`, including Make targets, scripts,
GitHub Actions, GoReleaser, docs, and generated CLI assets. Delete the npm
manifest, Node launcher, postinstall code, and npm-specific release wiring
before placing Go files in the same directory.

RFC 0013 supersedes the `apps/cli` npm placement in RFC 0006 and the
`modules/cli` source-location decision in RFC 0009. Keep their unrelated
workspace and core-module decisions intact; update or archive those RFCs when
this RFC is approved for implementation.

## Acceptance

- `apps/cli/go.mod` declares
  `github.com/borg0ai/dx-issuegraph/apps/cli`; `modules/core/go.mod` retains
  `github.com/borg0ai/dx-issuegraph`.
- `go.work` resolves the core and CLI modules at `modules/core` and
  `apps/cli`; no `modules/cli` module and no root `go.mod` remain.
- CLI can import required core `internal` packages and builds from
  `go -C apps/cli build`.
- Test, lint, release, install, and documentation workflows use `apps/cli`.
- `apps/cli` contains no npm CLI manifest or JavaScript runtime shim.
- CLI tests and the repository quality gates pass after the move.
- The moved CLI builds and runs without Docker.
