# RFC 0024: Release binaries bundle embedded Dolt (cgo builds)

**Status:** Approved

## Summary

Release binaries are built with cgo so that embedded Dolt is bundled and
`issuegraph init` works on a fresh machine with nothing else installed. `init`
keeps its current behavior: it creates the repository's Dolt database under
`.issuegraph/embeddeddolt/`. `issuegraph install` is unchanged and does not
touch Dolt. A release build script replaces the ad-hoc Docker commands that
produced the broken `CGO_ENABLED=0` builds.

## Problem

- Releases 1.3.1 and 1.3.2 were built with `CGO_ENABLED=0`. On them a bare
  `issuegraph init` fails with "embedded Dolt requires a CGO build, but this
  issuegraph binary was built with CGO_ENABLED=0". The default path of the
  product is broken for every installed user.
- The repository's own `Makefile` already builds with `CGO_ENABLED=1` and says
  embedded Dolt needs it. Only the release builds disagreed, and no script in the
  repository produces them.
- Embedded Dolt is the Dolt engine linked into the `issuegraph` process. What
  `init` writes into `.issuegraph/embeddeddolt/` is data only (a standard Dolt
  database directory with no executable or library, 2 to 5 MB when fresh;
  checked on 2026-11-01). The engine therefore has to come with the program that
  runs it. Downloading it separately was considered and set aside: the
  alternatives (an external `dolt` plus a server, or a separate engine program)
  are different modes with different behavior, not embedded Dolt.

## Goals

1. Every released binary opens embedded Dolt: `init`, `create`, `list` work with
   no other install.
2. One repository script builds all release assets, so a release is reproducible.
3. Asset names, archive layout and `SHA256SUMS` stay as in 1.3.2, so the Homebrew,
   Nix and Scoop channels need only new URLs and hashes.

## Non-goals

- Changing `init`, `install`, or any Dolt mode behavior.
- Making the release binary smaller by removing embedded Dolt.
- Shipping an external `dolt` or a Dolt installer.
- Code signing and notarization. 1.3.2 had none; this RFC does not add it.

## Design

### Build

`scripts/build-release.sh <version> [outdir]` builds, for each target, with
`CGO_ENABLED=1`, `-tags gms_pure_go` and the existing version `-ldflags`:

| Target | C toolchain |
|---|---|
| darwin-arm64 | host `clang` |
| darwin-amd64 | host `clang -arch x86_64` |
| linux-amd64 | `zig cc -target x86_64-linux-gnu.2.17` |
| linux-arm64 | `zig cc -target aarch64-linux-gnu.2.17` |
| windows-amd64 | `zig cc -target x86_64-windows-gnu` |
| windows-arm64 | `zig cc -target aarch64-windows-gnu` |

glibc 2.17 keeps the Linux binaries runnable on older distributions. `zig` is
used only as a C cross-compiler for the cgo parts; the program is still plain Go.
The script needs a macOS host with Xcode command line tools and `zig`, and fails
early with a message if either is missing.

It writes `issuegraph_<version>_<os>-<arch>.tar.gz` (containing `issuegraph`) or
`.zip` (containing `issuegraph.exe`), and `SHA256SUMS` in the `hash  name`
format of 1.3.2.

### Evidence (2026-11-01, this tree, Go 1.26.7, zig 0.17.0)

- darwin-arm64: built; `init`, `create`, `list` ran on the host.
- darwin-amd64: built with `clang -arch x86_64`; ran under Rosetta; `init` and
  `create` worked. A `zig cc` build of this target fails to link (no macOS
  frameworks), which is why the host clang is used.
- linux-amd64 and linux-arm64: built with `zig cc`; ran in Docker
  `debian:bookworm-slim`, `init` and `create` worked.
- windows-amd64 and windows-arm64: built (PE32+); never run, no Windows host.
- Binary sizes: 204 to 283 MB, against 101 MB for the `CGO_ENABLED=0` build. The
  compressed archives will be larger than the 44 to 48 MB of 1.3.2; the size is
  measured when the script first runs.

### Not verified

- Windows binaries at runtime.
- Whether the glibc 2.17 floor holds for the Dolt dependencies on very old
  distributions (checked only on Debian bookworm).
- GitHub's size limit per asset is 2 GB, so size is not a blocker, but download
  size for users grows.

## Acceptance

1. `scripts/build-release.sh` produces the six archives and `SHA256SUMS`; the
   checksums verify with `shasum -a 256 -c`.
2. For darwin-arm64, darwin-amd64 and linux-amd64/arm64, the extracted binary
   run in a temporary `HOME` and a new git repository: `issuegraph init`,
   `create`, `list` succeed with no `dolt` on `PATH`.
3. Windows archives are built and their checksums listed; the release notes and
   PR say plainly that they were not run.
4. `go vet` passes with `CGO_ENABLED=0` and `1`.
5. The next release (1.3.3) uses the script, and the three channel PRs are
   updated with the new URLs and hashes.
