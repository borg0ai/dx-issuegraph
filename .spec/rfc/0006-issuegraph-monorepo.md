# RFC 0006: pnpm and Turborepo workspace layout (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

Move the npm CLI app to `apps/cli`, reusable npm packages to `packages/*`, and coordinate JavaScript/TypeScript workspace tasks with pnpm and Turborepo. Go source layout and product asset relocation belong to RFCs [0009](0009-go-module-cli-build.md) and [0010](0010-issuegraph-asset-layout.md).

## Problem

The npm CLI package currently lives in `npm-package/`; repository has no pnpm workspace or Turbo task graph. This obscures the boundary between the installable CLI app and reusable npm packages.

## Goals

Define the root pnpm workspace, Turbo task graph, CLI npm app location, and reusable npm package locations. Keep package scripts directly callable through pnpm.

## Non-goals

This RFC does not change Go module structure or Go source paths (RFC 0009), product assets or plugin paths (RFC 0010), product naming (RFC 0002), CLI behavior, release service, version policy, or issue sync behavior.

## Design

Add root `package.json`, `pnpm-workspace.yaml`, `turbo.json`, and pnpm lockfile. Move the existing npm package into `apps/cli`; it owns the published CLI package and npm executable shim. Place each reusable npm package under `packages/<name>`. Configure Turbo tasks for package scripts and workspace dependency ordering. Keep Go commands as direct Go toolchain invocations coordinated by root scripts where needed; detailed Go layout and binary output path are specified by RFC 0009. Update package metadata, workspace references, CI commands, and developer documentation to the new npm paths.

## Acceptance

pnpm resolves every declared workspace; Turbo lists and runs workspace tasks in dependency order; the installable CLI package resides in `apps/cli`; reusable npm packages reside in `packages/*`; no active copy remains in `npm-package/*`; a clean checkout can install dependencies and run documented workspace scripts. Go module layout, product asset paths, product identity, and sync behavior remain governed by their own RFCs.
