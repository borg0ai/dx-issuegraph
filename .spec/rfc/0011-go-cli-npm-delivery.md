# RFC 0011: `@borg0ai/issuegraph` npm installer (child of 0001)

**Status:** Draft

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

Publish `apps/cli` to npm as `@borg0ai/issuegraph`. Its installer CLI is named
`ig-installer`; it downloads the exact versioned IssueGraph Go executable for
the user's OS and architecture. The installed product command is
`issuegraph`, which delegates to that local Go executable. GitHub Release
contains only versioned native Go CLI assets and checksums; the npm installer
is published separately to npm. Both use the same version.

## Problem

IssueGraph Go CLI is built and released as native archives. `apps/cli` is the
npm installer, not a second implementation of IssueGraph CLI behavior. Its
`ig-installer` command must install a version-matched native executable for
the host, and the installed `issuegraph` command must invoke that executable.
Unpinned or `latest` downloads can mismatch installer and executable
versions; missing OS/architecture assets can produce an installation that
cannot run.

## Goals

- Keep npm installer and its `ig-installer` command under `apps/cli`.
- Publish npm package as `@borg0ai/issuegraph`; reserve `issuegraph` command
  for the installed Go product CLI.
- On npm install, select and download the Go CLI asset matching the host OS,
  architecture, and exact npm package version.
- Verify the downloaded release asset against that release's checksum before
  extracting the executable.
- Keep the `issuegraph` command shim thin: forward arguments, environment,
  standard streams, and exit status to the local native executable.
- Build and publish Linux, macOS, and Windows Go CLI assets and checksums to
  GitHub Release before publishing the npm installer to npm.
- Build each native binary once; reuse the same release artifact for npm
  installation instead of producing a second, potentially different binary.
- Derive release version from the triggering `vMAJOR.MINOR.PATCH` tag and use
  it consistently for GitHub Release metadata, Go binaries, asset names,
  checksums, and the npm installer package.

## Non-goals

- Does not move Go source or change Go module boundaries (RFC 0009).
- Does not decide product, npm package, repository, or command branding beyond
  using the identity defined by RFC 0002.
- Does not publish the npm installer as a GitHub Release asset; npm and GitHub
  Releases remain separate distribution channels.
- Does not bundle every platform binary into one npm tarball.
- Does not make the npm package compile Go source on the user's machine.
- Does not publish an npm package or GitHub Release as part of this RFC's
  implementation work.

## Design

`apps/cli` owns npm package `@borg0ai/issuegraph`, the `ig-installer` command,
its postinstall downloader, and the `issuegraph` command shim. `ig-installer`
is the npm installer's explicit command name; `issuegraph` is the product
command exposed by the package and backed by the downloaded Go executable.
npm package version equals native CLI release version and selects that exact
GitHub Release tag. Installer must never silently substitute `latest` or
another version. A single platform mapping translates Node OS and architecture
identifiers to release asset names. Unsupported tuples fail with an error
listing supported tuples. Product command identity follows RFC 0002.

During npm installation, the installer downloads the matching archive and
`checksums.txt` from that version's GitHub Release, validates the archive
SHA-256, extracts only the expected executable into the package's private
native-binary directory, and sets executable permissions where required. It
must reject checksum mismatch, missing assets, malformed archives, and
unsupported hosts with actionable errors. After installation, the
`issuegraph` command shim runs the local executable and forwards arguments and
process behavior; command execution does not make network requests.
`ig-installer` is the installation command and is not an alias for the Go
product command.

The triggering Git tag is the single release-version source. A release
preflight derives `RELEASE_VERSION` from `vMAJOR.MINOR.PATCH`, validates the
tag, and checks that Go version metadata and `apps/cli/package.json` match. A
mismatch stops release before publishing. GoReleaser, the native macOS build,
and the npm publish job all consume that same version. The version appears in
the GitHub Release tag and title, Go binary metadata, native asset names,
checksums, and npm installer metadata. The workflow does not publish the npm
package to GitHub Releases or commit a version bump back to the repository.

Release workflow keeps GoReleaser responsible for Linux and Windows artifacts
and the native macOS job responsible for macOS artifacts. Both upload assets
and checksums to the same tagged GitHub Release. The npm publish job depends on
both jobs, verifies required platform assets and checksums exist, then tests
`apps/cli` installation against those exact release downloads before publishing
npm. The pre-release installer gate may build a local candidate for tests, but
it cannot substitute that candidate for versioned release assets during
publication. No platform is advertised by the installer unless matching
GitHub Release asset exists.

Alternatives considered:

1. Put every OS/architecture binary in one npm tarball. This avoids install-time
   GitHub access, but duplicates every native binary for every npm user and
   increases package size substantially.
2. Install the exact versioned Go CLI release asset during npm installation.
   This keeps the npm installer small and reuses release binaries, while
   requiring network access at install time and reliable GitHub availability.

Recommended: option 2. npm's role is to install IssueGraph Go CLI. The exact
versioned release executable becomes the installed `issuegraph` command, with
one native binary per machine rather than a bundle of every platform binary.

## Acceptance

- Tag release workflow builds native Go CLI assets for Linux, macOS, and
  Windows, covering every OS/architecture tuple advertised by `apps/cli`.
- Release workflow derives one version from the `v` tag and fails before
  publishing if Go version metadata or `apps/cli/package.json` disagree.
- GitHub Release tag/title, Go binary version, native asset names, checksums,
  and npm installer version all match the tag-derived version.
- Linux and Windows assets plus checksums appear in the tagged GitHub Release;
  macOS assets plus checksums are uploaded to that same release by the native
  macOS job.
- npm publication waits for successful Linux/Windows and macOS release jobs
  and verifies required release assets and checksums before publishing.
- Installing the versioned npm package downloads only the matching release
  asset, validates its checksum, and installs IssueGraph Go CLI locally.
- npm package identity is `@borg0ai/issuegraph`; installer command is
  `ig-installer`; product command is `issuegraph`.
- The `issuegraph` command runs the local executable and preserves arguments, environment,
  standard streams, and exit status; runtime performs no network download.
- Unsupported OS/architecture, unavailable release asset, and checksum failure
  produce clear installation errors; installer never falls back to `latest`.
- Package tests cover OS/architecture mapping, exact-version URL selection,
  checksum verification, extraction, launcher forwarding, and failure paths.
- `npm pack --dry-run` confirms npm package contains installer and command shim,
  not Go source or all platform binaries.
- GitHub Release contains native Go CLI assets and checksums; npm installer is
  published only to npm.
