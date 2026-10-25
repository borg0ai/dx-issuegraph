# RFC 0012: Luban Homebrew delivery for IssueGraph CLI (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

Replace the npm CLI installer release channel with a native macOS CLI
distribution through `systembugtj/luban-homebrew`. Tagged GitHub releases
provide checksummed Apple Silicon (`arm64`) and Intel (`amd64`) binaries; the
Luban Homebrew formula installs the matching archive.

## Problem

`apps/cli` currently publishes a Node.js wrapper whose only purpose is
downloading and launching the Go CLI. It duplicates release/version logic and
adds npm as a required distribution service. The chosen delivery channel is
Luban Homebrew, which can install the native CLI directly for both macOS CPU
architectures.

## Goals

- Remove the npm CLI package, installer, npm-specific tests, and npm release
  job from this repository.
- Publish native Go CLI archives and checksums as versioned GitHub Release
  assets for macOS arm64 and amd64.
- Maintain the IssueGraph formula in `systembugtj/luban-homebrew`, using those
  exact versioned assets and checksums.
- Verify `brew install systembugtj/luban-homebrew/issuegraph` on Apple Silicon
  and Intel macOS.
- Keep release version and asset names derived from the Git tag.

## Non-goals

- Moving Go CLI source from `modules/cli` to `apps/cli`; RFC 0013 owns that
  layout change.
- Removing npm packages unrelated to the IssueGraph CLI.
- Unpublishing or deprecating an already-published npm package; that registry
  action needs separate authorization.
- Changing IssueGraph command behavior, storage, or supported OS targets.
- Introducing Docker as a build, release, or installation dependency.

## Design

The `shipify` skill produces the versioned native archives and `SHA256SUMS`
and publishes them to the GitHub Release; the repository has no release
workflow ([0019](0019-shipify-only-release.md)). Remove npm package metadata,
postinstall/launcher code, and npm package tests. The GitHub Release is the
immutable source of binary assets.

Add the `issuegraph` formula to `systembugtj/luban-homebrew`. The formula uses
the tagged release archive and checksum for the requested version, selecting
arm64 on Apple Silicon and amd64 on Intel macOS. Keep tap changes in that
repository's normal feature-branch and PR workflow. Do not advertise the
Homebrew install command until the formula has landed.

This RFC precedes RFC 0013: remove the npm app first, then place the Go CLI
module in the now-free `apps/cli` path.

## Acceptance

- Repository contains no IssueGraph npm CLI package, installer, package tests,
  or npm publish job.
- Tagged release publishes checksummed macOS arm64 and amd64 archives.
- Luban Homebrew formula selects the correct architecture and verifies the
  release checksum.
- A clean Intel Mac and Apple Silicon Mac both install the tagged version and
  `issuegraph version` reports that version.
- The tap change is merged through its PR workflow; repository docs point to
  the merged formula.
- No npm registry package is unpublished or modified by this RFC.
- Release builds and Homebrew installation require no Docker daemon or image.
