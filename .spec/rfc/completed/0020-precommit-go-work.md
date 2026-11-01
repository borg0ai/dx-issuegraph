# RFC 0020: Pre-commit hook reads go version from go.work

**Status:** Implemented

## Summary

`.githooks/pre-commit` pins the lint toolchain from the `go` directive of a root `go.mod`. RFC 0013 removed that file, so every commit staging a `.go` file fails. Read the directive from root `go.work` and lint per workspace module instead.

## Problem

`awk: can't open file go.mod` aborts any commit that stages a Go file (reproduced committing the 1.3.1 version bump, which touches `apps/cli/version.go`).

## Goals

Commits staging Go files pass the hook's toolchain-pinning step again.

## Non-goals

Lint rules, golangci-lint version, and the `.github/workflows` references in hook comments.

## Design

`.githooks/pre-commit`: replace `go.mod` with `go.work` in the awk call and the error message. `go.work` is the single root file declaring the `go` directive (1.26.5). Also run `golangci-lint` once per `use` module of `go.work` that has staged Go files, from inside that module, because root `./...` has no module.

## Acceptance

`git commit` with a staged `.go` file runs the hook past the toolchain step with no `awk` error.
