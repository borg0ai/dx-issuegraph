# RFC 0009: Independent Go modules and CLI build (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

Split Go implementation into independent modules under `modules/*`, coordinate them with root Go workspace files, and keep the TypeScript npm app in `apps/cli` separate from native Go code.

## Problem

Current in-progress layout put all Go code and module metadata under one monolithic `modules/issuegraph-go`. The target needs separately owned Go modules, root workspace orchestration, a clean repository root, and a Go CLI binary built for the TypeScript app to launch.

## Goals

Place Go modules under `modules/<name>`, starting with `modules/core` and `modules/cli`. Keep existing public Go module path on `modules/core` to preserve package imports. Keep root `go.mod` and `go.work` as workspace entry files; root `go.work` coordinates independent modules. Build the native CLI from `modules/cli` into `apps/cli/bin/cli`. Keep `apps/cli` source TypeScript/npm-only.

## Non-goals

Does not move TypeScript implementation into Go modules; does not define npm workspace layout (RFC 0006), product asset paths (RFC 0010), command branding (RFC 0002), or issue storage/sync behavior. Root `go.mod` cannot manage nested Go modules; root `go.work` is the Go workspace manager.

## Design

Move existing root Go module contents, Go-only configs, tests, and module dependency metadata into `modules/core`, preserving the existing module path and package-relative directories. Move Go CLI source and tests to `modules/cli`, with its own `go.mod`; set its module path beneath the core module path so it can import core `internal/*` packages under Go's import-path boundary. Keep each module's dependencies in its own `go.mod`/`go.sum`. Keep a minimal root `go.mod` workspace anchor and root `go.work` listing the root and every module under `modules/*`. Root scripts, Turbo tasks, CI, GoReleaser, and docs must run Go commands in the owning module. Build CLI with `go -C modules/cli build -o ../../apps/cli/bin/cli .`; the TypeScript npm app invokes that binary. Generated product assets come from RFC 0010 and are generated inside the owning Go module.

## Acceptance

Root `go env GOMOD` and `go env GOWORK` resolve to repository `go.mod` and `go.work`; `modules/core/go.mod` preserves the current public Go module path; `modules/cli/go.mod` is an independent module beneath that path; all production Go source is under `modules/*`; `go -C modules/cli build -o ../../apps/cli/bin/cli .` succeeds after asset generation; TypeScript app in `apps/cli` launches the generated binary; CI and docs build/test each module; no workflow or script expects `modules/issuegraph-go`.
