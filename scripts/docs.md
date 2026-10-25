# Noridoc: scripts

Path: @/scripts

### Overview

The `scripts` directory contains build and release automation utilities for the issuegraph project. These scripts handle version management, installation with embedded build info, and release orchestration across multiple distribution channels.

Key scripts include version bumping, installation helpers that inject git information at build time, and release coordination across GitHub, Luban Homebrew, and PyPI.

## generate-cli-docs.sh

Generates maintained CLI reference docs from the live Cobra command tree in
two stages: `issuegraph help --docs-root` emits vendor-neutral output (the single-file
reference plus a generic per-command tree in uncommitted staging), then
`tools/docsmint` post-processes the staging tree into the committed Mintlify
pages and splices the CLI Reference pages array in `docs/docs.json`.

### Usage

```bash
# Regenerate checked-in CLI docs using a temporary no-cgo build
./scripts/generate-cli-docs.sh

# Regenerate/check against an existing binary
./scripts/generate-cli-docs.sh ./issuegraph
./scripts/generate-cli-docs.sh --check ./issuegraph
```

### Outputs

- `docs/CLI_REFERENCE.md` from the live Cobra command tree
- `docs/cli-reference/*.md` (Mintlify pages) via `tools/docsmint`
- the CLI Reference pages array in `docs/docs.json`

`scripts/check-cli-docs-drift.sh` runs the `--check` mode in CI and fails when
the committed copies are stale relative to the live command tree.

## check-doc-freshness.sh

Validates marker-based freshness for reference-shaped docs that are not generated from a single clean source.

### Usage

```bash
./scripts/check-doc-freshness.sh
make check-docs
```

The script currently checks the reference docs named in `engdocs/DOC_INVENTORY.md`: `CONFIG.md`, `SETUP.md`, `ADO_CONFIG.md`, `JSON_SCHEMA.md`, `RECOVERY.md`, `ERROR_HANDLING.md`, `LINTING.md`, and `design/otel/otel-data-model.md`. Each doc must be listed in the inventory, include a recent `Last reviewed:` marker, include a `Freshness source:` marker, and name source paths or globs that exist in the repository.

Set `DOC_FRESHNESS_MAX_AGE_DAYS` to a nonnegative decimal integer to override
the default 90-day review window. The policy threshold is compared as decimal
text, so values larger than Bash's integer range remain valid and cannot become
arithmetic expressions or overflow. Set `DOC_FRESHNESS_TODAY=YYYY-MM-DD` when
testing date behaviour.
Date validation uses Bash integer arithmetic over the proleptic Gregorian
calendar, preserves the script's four-digit `YYYY-MM-DD` domain from
`0001-01-01` through `9999-12-31`, and does not require Python or
platform-specific date parsing. The native `date` command is used only to
supply today's value when `DOC_FRESHNESS_TODAY` is unset. Pull requests exercise
the real process boundary on Linux, macOS, and Git Bash on Windows.

### How it fits into the larger codebase

- **Version Pipeline**: Works alongside the version infrastructure in `@/apps/cli/version.go` by extracting and passing git information at build time via ldflags.

- **Release Automation**: The `update-versions.sh` script and the `shipify` skill orchestrate the release process, ensuring version consistency across CLI, plugin, and MCP server.

- **CI/CD Integration**: The release workflow uses the same ldflag patterns established by these scripts, ensuring consistency across all installation methods.

- **Multi-Channel Distribution**: These scripts ensure consistent version reporting through GitHub releases and the three package channels.

### Core Implementation

**Makefile Integration** (`@/Makefile`, lines 37-41):

The Makefile's `install` target uses identical logic:
- Extracts git info at build time
- Passes to `go install` via ldflags
- Ensures that standard `make install` produces binaries with full version info, not just `make build`

**Version Bumping** (`update-versions.sh`):

- Coordinates version updates across CLI, plugin metadata, and MCP server files
- Ensures all distribution channels report consistent version numbers

### Things to Know

**Why Explicit Ldflags Are Necessary**:
- `go install` does not automatically embed VCS information like `go build` does (even though Go supports it)
- Without explicit ldflags, binaries lack commit and branch information regardless of installation method
- The Makefile and the release builder compensate for this limitation

**Testing the Version Pipeline**:
- After installing, users should immediately see full version info via `issuegraph version`
- The text output shows format like: `issuegraph version 0.29.0 (dev: main@7e70940)`
- JSON output includes both `commit` and `branch` fields

**Release Coordination**:
- Release orchestration across channels is done by the `shipify` skill

**Platform Compatibility**:
- All scripts use POSIX shell constructs (bash on all platforms)
- Git operations work identically on macOS, Linux, and Windows (with Git for Windows)
- The go install command behaves consistently across all platforms

**Security Considerations**:
- Scripts use `set -e` to fail fast on any errors
- Git commands are defensive (using `2>/dev/null` to suppress errors)
- No shell injection risks as git values are passed as structured arguments

Created and maintained by Nori.
