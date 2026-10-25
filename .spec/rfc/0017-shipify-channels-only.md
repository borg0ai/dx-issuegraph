# RFC 0017: Ship only through the three shipify channels (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

IssueGraph is delivered through exactly the three channels the `shipify` skill
defines in `~/.shipify/setting.json`: `luban-homebrew` (macOS, [0012](0012-luban-homebrew-release.md)),
`cangjie-nix` (Linux, [0015](0015-cangjie-nix-release.md)) and `shennong-bucket`
(Windows, [0016](0016-shennong-bucket-release.md)), all owned by `systembugtj`.
Every other distribution path is removed from this repository, with no legacy
alias or redirect.

## Problem

The repository still carries release paths from before the rebrand: PyPI
publishing of the MCP package, shell and PowerShell install scripts, a winget
update script, an in-repo Nix flake with its own CI and vendor-hash automation,
and install instructions for each in the docs. Each one needs its own secrets
and upkeep, can drift from the tagged release, and tells users there are more
supported ways to install than there are.

## Goals

- The only release path is the `shipify` skill: GitHub Release assets plus the
  three channel pull requests; nothing else publishes anywhere. The repository
  itself holds no release workflow ([0019](completed/0019-shipify-only-release.md)).
- Remove the PyPI publish path: the `release-package-mcp` and `publish-pypi`
  jobs (they lived in `release.yml`, which 0019 deletes), `.github/workflows/test-pypi.yml`,
  and the `PYPI_API_TOKEN` dependency.
- Remove `scripts/install.sh`, `install.ps1`, `scripts/update-winget.sh`,
  `scripts/check-go-install-guidance.sh`, and the in-repo Nix delivery files
  (`flake.nix`, `flake.lock`, `default.nix`, `overlay.nix`, `packages.nix`,
  `scripts/update-nix-vendorhash.sh`, and the `nix-build`, `update-flake-lock`
  and `update-vendor-hash` workflows).
- Releases carry a `SHA256SUMS` file, as shipify requires; no consumer in the
  repository refers to `checksums.txt`.
- Rewrite the install sections of the README, `docs/getting-started/installation.md`,
  and `docs/index.md` to name only the three channels, and only once each channel's pull request has merged.
- Leave no reference to a removed path: grep for each removed file name, job
  name, and install command returns nothing outside `CHANGELOG.md`.

## Non-goals

- Unpublishing or deprecating anything already on PyPI, npm, or winget; those
  registry actions need separate authorization.
- Changing CLI behavior, storage, or build targets.
- Deleting the Python MCP server source in `integrations/issuegraph-mcp`. Only
  its publishing goes; whether the source stays is a separate decision.
- Changing GitHub Release creation, signing, notarization, or attestation.
- Editing the three channel repositories; that is the work of 0012, 0015, 0016.
- Docker as a build, release, or install dependency.

## Design

Order matters: land 0012, 0015 and 0016 as merged channel pull requests first,
or at least keep their docs unpublished, then remove the old paths so no user
is left without an install route. Removal is deletion, not deprecation: no stub
script, redirect, or "moved" notice.

Release verification follows the shipify contract: every supported OS and
architecture asset is attached to the tagged release together with `SHA256SUMS`;
each channel checksum is computed from the published asset, never from a local
build.

Decisions:

- `go install` and building from source are not install routes; install docs
  do not mention them. Contributor build steps stay in contributor docs only.
- The in-repo Nix files are removed; no dev shell stays. Delivery is
  `cangjie-nix` only.
- `integrations/issuegraph-mcp` stays in the repository, unpublished, until
  [0018](0018-mcp-go-subcommand.md) replaces it with `issuegraph mcp` and
  deletes it. No Python package is delivered.

## Acceptance

- No workflow in the repository publishes to PyPI, npm, or winget.
- Every file listed under Goals as removed is absent from the tree, and no
  workflow, Makefile target, or script references it.
- Grep for `checksums.txt` finds nothing outside `CHANGELOG.md` and archived RFCs.
- README and installation docs list exactly macOS Homebrew, Linux Nix, and
  Windows Scoop, each with its real command, and nothing else.
- `sync-check` and the repository's workflow lint pass.
- No registry package is unpublished or modified by this RFC.
