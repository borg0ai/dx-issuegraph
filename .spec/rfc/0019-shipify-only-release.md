# RFC 0019: Release only through shipify, with no release machinery in the repository (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

The repository contains no release workflow, release tooling, or release
documentation. Every release is performed by the `shipify` skill, which reads
`~/.shipify/setting.json`, bumps versions, tags, builds and publishes the GitHub
Release assets for macOS, Linux and Windows, and opens one pull request per
channel repository ([0012](0012-luban-homebrew-release.md),
[0015](0015-cangjie-nix-release.md), [0016](0016-shennong-bucket-release.md)).
GoReleaser is removed.

## Problem

The repository carries its own release system: a tag-triggered `release.yml`,
a GoReleaser config, a long `RELEASING.md` runbook, a release molecule and
gateway script, and a stub `bump-version.sh`. It duplicates what shipify already
does, builds targets no channel installs, and drifts from the channel contract.
Two release systems mean two places to keep correct and a release that depends
on which one a maintainer reaches for.

## Goals

- Delete `.github/workflows/release.yml` and `.goreleaser.yml`.
- Delete `RELEASING.md`, `engdocs/RELEASING.md`, `engdocs/RELEASE-STABILITY-GATE.md`
  and `scripts/bump-version.sh`. The release molecule and `scripts/release.sh`
  are already gone.
- Delete the build and signing assets that only a release build used:
  `scripts/sign-windows.sh`, `scripts/verify-cgo.sh`, `scripts/gen-winres.sh`,
  and `apps/cli/winres/`. Windows signing, PE metadata, and the post-build ICU
  check are left entirely to `shipify`.
- Remove every reference to the deleted files from workflows, scripts, tests,
  and docs, so nothing points at a release path that no longer exists.
- Leave one statement of the release process in the repository: releases run
  through the `shipify` skill. No other release documentation is kept.
- Keep what is not release machinery: `scripts/update-versions.sh` and
  `scripts/check-versions.sh` (version consistency), and the PR and main CI.
  Both version scripts stop touching the deleted `winres` files.

## Non-goals

- Deciding how shipify builds archives; that is shipify's contract.
- Changing CLI behavior or any build flag used by local `make` targets.
- Editing the three channel repositories.
- Unpublishing any existing registry package.
- Docker as a build or release dependency.

## Design

Removal order: write this RFC, delete the files, fix the references that the
deletion breaks (a workflow test that pins `release.yml`, a docs-sync list that
names `RELEASING.md`, a docs workflow path filter), then sweep the docs. The
sweep is limited to current docs; archived RFCs and `CHANGELOG.md` keep their
history.

Open items, none verified yet:

- Whether `shipify` can build the Linux and Windows CGO binaries, sign the
  Windows executable, and check for ICU linkage without a repository workflow
  or these scripts. Until that is shown, Windows releases may ship unsigned and
  without PE metadata, which raises antivirus false-positive risk
  (`docs/reference/antivirus.md`).
- Whether any maintainer still needs the contract-corpus release artifact that
  the old workflow published.

## Acceptance

- `.github/workflows/release.yml`, `.goreleaser.yml`, `RELEASING.md`,
  `engdocs/RELEASING.md`, `engdocs/RELEASE-STABILITY-GATE.md`,
  `scripts/bump-version.sh`, `scripts/sign-windows.sh`, `scripts/verify-cgo.sh`,
  `scripts/gen-winres.sh` and `apps/cli/winres/` do not exist.
- Grep for `goreleaser` and `RELEASING.md` finds nothing outside `CHANGELOG.md`,
  archived RFCs, and audit history under `engdocs/`.
- No workflow, script, or test references a deleted file; `go vet` on the
  affected packages passes.
- `sync-check` passes, and 0012, 0015, 0016, 0017 no longer describe a repository
  release workflow or GoReleaser.
