# RFC 0010: IssueGraph 集成资产目录迁移 (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

Make root `issuegraph/*` the single source of truth for product skills, agent metadata, and plugin manifests; remove `plugins/beads/*` as a canonical path.

## Problem

Product assets currently sit under a legacy `plugins/beads/` path, which couples the product name to its former brand and plugin packaging layout. Go embedding also needs a deterministic boundary that does not maintain a second manually copied asset tree.

## Goals

Move product assets to the agreed `issuegraph/*` tree, update all references, and keep one authoritative copy. Ensure generated Go embedding data derives reproducibly from those files.

## Non-goals

Does not define pnpm/npm workspace layout (RFC 0006), Go module/source layout or CLI build output (RFC 0009), product command naming (RFC 0002), or issue data sync behavior (RFCs 0003, 0004, 0008).

## Design

Move plugin and skill content out of `plugins/beads/*` into root `issuegraph/*`, with the product skill at `issuegraph/skills/issuegraph`. Move or update associated marketplace/plugin manifests to use IssueGraph paths. Update Go asset consumers to read generated Go source produced deterministically from canonical `issuegraph/*` inputs; generated files stay ignored and are rebuilt by supported Go build/test entry points. Update docs, CI, tests, package references, and CODEOWNERS. Do not keep duplicated source assets or compatibility copies at the old path.

## Acceptance

No product asset is sourced from or copied manually under `plugins/beads/*`; all plugin, skill, and agent asset references resolve under canonical `issuegraph/*`; generated Go assets are reproducible from a clean checkout; Go build/test consumes the generated package; search and package checks show no active stale path references.
