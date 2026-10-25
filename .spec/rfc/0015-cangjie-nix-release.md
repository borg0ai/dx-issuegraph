# RFC 0015: Cangjie Nix delivery for IssueGraph CLI on Linux (child of 0001)

**Status:** Draft

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

Distribute the native IssueGraph CLI on Linux through the `cangjie-nix`
repository. A Nix package there installs the versioned Linux archive from the
tagged GitHub Release and verifies it against that release's checksum. It is
the Linux counterpart of the macOS channel in [0012](0012-luban-homebrew-release.md).

## Problem

0012 delivers macOS only. Linux users have no managed install path: the
repository's own `flake.nix` builds from source and needs a vendor hash that
must be refreshed for every dependency change. Users who only want the
released binary should not compile Go or CGO dependencies.

## Goals

- Publish checksummed Linux `amd64` and `arm64` archives in each tagged GitHub
  Release, produced by the `shipify` skill.
- Maintain an `issuegraph` package in `cangjie-nix` that fetches those exact
  release assets and their sha256 checksums for the matching architecture.
- Derive package version and asset URLs from the Git tag; never track `latest`.
- Open each `cangjie-nix` update as a pull request from a release-specific
  branch, only after the GitHub Release is public and its assets verified.
- Verify `nix run` / `nix profile install` of the package on x86_64-linux and
  aarch64-linux.

## Non-goals

- Replacing the in-repo `flake.nix`, `overlay.nix`, or `packages.nix`; they
  remain the build-from-source path for contributors.
- macOS or Windows delivery (0012 and 0016).
- Submitting to nixpkgs.
- Changing CLI behavior, storage, or build targets.
- Docker as a build, release, or install dependency.

## Design

The `shipify` skill is the single producer of `issuegraph_<version>_linux_<arch>.tar.gz`
and `SHA256SUMS`. The package in `cangjie-nix` is a binary-fetch derivation:
`fetchurl` per architecture with the sha256 taken from `SHA256SUMS`
(converted to Nix SRI form), installing `issuegraph` into `$out/bin`.

Open items to confirm before approval, none verified yet:

- Whether the Linux binaries are dynamically linked (the build uses CGO). If
  so the derivation needs `autoPatchelfHook` plus the matching libraries.
- The owner and default branch of `cangjie-nix`; this RFC assumes it lives
  beside `luban-homebrew` under the same owner and uses a PR base chosen by
  the repository, never a direct push to its default branch.
- Whether shell completions ship in the archive or are generated at build.

After the release assets are public, `shipify` checks each checksum, updates
the package expression, and opens the pull request in `cangjie-nix`.
Do not advertise the Nix install command until the PR has merged.

## Acceptance

- Tagged release contains checksummed Linux `amd64` and `arm64` archives.
- The `cangjie-nix` package selects the correct architecture and verifies the
  release checksum.
- On x86_64-linux and aarch64-linux, installing through Nix and running
  `issuegraph version` reports the tagged version.
- The package change is merged through the repository's PR workflow before
  docs name the install command.
- No credential or mutable local path appears in the package expression.
- Build, release, and install paths need no Docker daemon or image.
