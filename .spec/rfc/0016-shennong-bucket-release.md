# RFC 0016: Shennong Scoop bucket delivery for IssueGraph CLI on Windows (child of 0001)

**Status:** Draft

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

Distribute the native IssueGraph CLI on Windows through the `shennong-bucket`
Scoop bucket. A Scoop manifest there points at the versioned Windows zip in the
tagged GitHub Release and pins its sha256. It is the Windows counterpart of
[0012](0012-luban-homebrew-release.md) and [0015](0015-cangjie-nix-release.md).

## Problem

0012 delivers macOS only. Windows users have no managed install path, and the
npm installer that used to reach them is gone ([0011](rejected/0011-go-cli-npm-delivery.md)).
The release already builds signed Windows binaries, so a bucket manifest is the
smallest addition that makes them installable and upgradable.

## Goals

- Publish checksummed Windows `amd64` and `arm64` zip archives in each tagged
  GitHub Release, produced and signed by the `shipify` skill.
- Maintain an `issuegraph` manifest in `shennong-bucket` with one URL and hash
  per architecture, `bin` set to `issuegraph.exe`, and the version taken from
  the Git tag.
- Open each bucket update as a pull request from a release-specific branch,
  only after the GitHub Release is public and its assets verified.
- Verify `scoop bucket add` and `scoop install issuegraph` on Windows amd64 and
  arm64, and `scoop update` to a newer tag.

## Non-goals

- macOS or Linux delivery (0012 and 0015).
- Winget, Chocolatey, or MSI packaging.
- Changing Authenticode signing; the existing secrets and script stay as is.
- Changing CLI behavior, storage, or build targets.
- Docker as a build, release, or install dependency.

## Design

The `shipify` skill is the single producer of `issuegraph_<version>_windows_<arch>.zip`
and `SHA256SUMS`. The manifest uses those exact asset names, copies each hash
from `SHA256SUMS`, and may include `checkver`/`autoupdate` stanzas so the
bucket's own tooling can propose updates; `shipify` remains the
primary updater.

Open items to confirm before approval, none verified yet:

- The owner and default branch of `shennong-bucket`; this RFC assumes it lives
  beside `luban-homebrew` under the same owner and takes PRs rather than direct
  pushes.
- Whether the zip root holds `issuegraph.exe` directly, which decides the
  manifest `bin` path.
- Whether `LICENSE`, `README.md`, and `CHANGELOG.md` in the zip need any
  `persist` or `extract_dir` handling.

After the release assets are public, `shipify` checks each checksum against
`SHA256SUMS`, updates the manifest, and opens the pull request in the bucket. Do not advertise the Scoop install command until the PR has merged.

## Acceptance

- Tagged release contains checksummed, signed Windows `amd64` and `arm64` zips.
- The manifest lists both architectures with hashes that match `SHA256SUMS`.
- On Windows amd64 and arm64, `scoop install` then `issuegraph version`
  reports the tagged version; `scoop update` moves to a newer tag.
- The manifest change is merged through the bucket's PR workflow before docs
  name the install command.
- No credential or mutable local path appears in the manifest.
- Build, release, and install paths need no Docker daemon or image.
