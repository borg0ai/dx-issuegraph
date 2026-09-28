---
title: Troubleshooting
description: Fixes for common issuegraph problems across installation, the database and Dolt server, sync, git hooks, dependencies, and platform-specific issues.
---

Common issues and solutions. For step-by-step runbooks, see the
[Recovery section](/recovery/index).

## Installation Issues

### `issuegraph: command not found`

```bash
# Check if installed
which issuegraph
go list -f {{.Target}} github.com/steveyegge/beads/modules/cli

# Add Go bin to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$PATH:$(go env GOPATH)/bin"

# Or reinstall with the recommended installer
curl -fsSL https://raw.githubusercontent.com/gastownhall/beads/main/scripts/install.sh | bash
```

### Wrong version of issuegraph running

If `issuegraph version` shows an unexpected version (e.g., older than what you just
installed), you likely have multiple `issuegraph` binaries in your PATH:

```bash
# Check all issuegraph binaries in PATH
which -a issuegraph

# Example output showing conflict:
# /Users/you/go/bin/issuegraph        <- From go install (older)
# /opt/homebrew/bin/issuegraph        <- From Homebrew (newer)

# Remove the old go install version
rm ~/go/bin/issuegraph

# Or remove mise-managed Go installs
rm ~/.local/share/mise/installs/go/*/bin/issuegraph

# Verify
which issuegraph
issuegraph version
```

This happens when a binary from an earlier `go install` sits in `~/go/bin/`
ahead of a newer package-manager install. Choose one installation method
(Homebrew recommended) and stick with it.

### `zsh: killed issuegraph` on macOS

CGO/SQLite compatibility issue:

```bash
CGO_ENABLED=1 GOFLAGS=-tags=gms_pure_go go install github.com/steveyegge/beads/modules/cli@latest

# Or if building from source
git clone https://github.com/gastownhall/beads
cd beads
CGO_ENABLED=1 go build -tags gms_pure_go -o issuegraph ./modules/cli
sudo mv issuegraph /usr/local/bin/
```

Homebrew builds already enable CGO, so this shouldn't be necessary there. If
you still see crashes with the Homebrew version, please
[file an issue](https://github.com/gastownhall/beads/issues).

### Permission denied

```bash
chmod +x $(which issuegraph)

# Or install to a user directory instead
mkdir -p ~/.local/bin
mv issuegraph ~/.local/bin/
export PATH="$HOME/.local/bin:$PATH"
```

### Antivirus flags issuegraph as malware

Kaspersky, Windows Defender, and others sometimes flag `issuegraph` as a generic
trojan. This is a **false positive** — Go binaries commonly trigger antivirus
heuristics. Verify the binary's SHA256 checksum against the
[GitHub release page](https://github.com/gastownhall/beads/releases) before
adding an exclusion. See [Antivirus False Positives](/reference/antivirus) for
per-vendor instructions.

## Database Issues

### Database not found

```bash
# Initialize issuegraph
issuegraph init --quiet

# Or point issuegraph at an existing .beads directory
BEADS_DIR=/path/to/.beads issuegraph list
```

### Database locked

```bash
# Stop the Dolt server if running (server mode)
issuegraph dolt stop

# Find and kill hanging issuegraph processes
ps aux | grep issuegraph
kill <pid>

# Try again
issuegraph list
```

<Warning>
Do NOT remove files inside `.dolt/` directories (including `noms/LOCK`).
These are Dolt-internal files — removing them WILL cause unrecoverable data
corruption. Dolt manages these files itself.
</Warning>

For high-concurrency scenarios (multiple agents), server mode
(`issuegraph init --server`) handles concurrent access natively via `dolt sql-server`.

### `issuegraph init` refuses to run

`issuegraph init` and `issuegraph dolt` refuse operations that could destroy local or remote
history, printing a pattern code such as `init-local-exists` or
`pk-fork-refused`. Each code has a runbook — see
[Recovery Playbooks](/recovery/init-safety). Export first
(`issuegraph export -o backup.jsonl`) if you intend to re-initialize over existing
data.

### Corrupted database

Distinguish **logical consistency issues** (ID collisions, wrong prefixes)
from **physical database corruption** (disk failures, power loss, filesystem
errors).

For logical consistency issues — this is not corruption:

```bash
issuegraph doctor --fix
```

For physical corruption, rebuild from a Dolt remote or a backup:

```bash
# Move the damaged data directory aside:
mv .beads/embeddeddolt .beads/embeddeddolt.backup   # embedded mode (default)
mv .beads/dolt .beads/dolt.backup                   # server mode

issuegraph init
issuegraph dolt pull    # Pull from Dolt remote if configured

# Or restore from a backup:
# issuegraph backup restore [path] --force
```

See [Database Corruption](/recovery/database-corruption) for the full runbook.

### Dolt journal corruption after restart

**Symptom (server mode):** After a system restart, `issuegraph` reports that the Dolt
server started but is not accepting connections, and `.beads/dolt-server.log`
contains:

```text
possible data loss detected in journal file at offset ...: corrupted journal
```

**Cause:** Dolt detected damaged journal blocks after an unclean shutdown.
This is not the same as a stale PID, stale port, or stale lock file. `issuegraph`
will not run Dolt's data-loss repair mode automatically.

**Safe recovery when your remote is current:**

```bash
# Server mode data lives at .beads/dolt; embedded mode at .beads/embeddeddolt
mv .beads/dolt .beads/dolt.corrupt.$(date +%Y%m%dT%H%M%S)
issuegraph bootstrap --dry-run
issuegraph bootstrap --yes
issuegraph stats
```

If the remote may be stale, keep the corrupt directory for forensics and
inspect it with `dolt fsck` before considering
`dolt fsck --revive-journal-with-data-loss`. Only use the revive path after
reviewing Dolt's data-loss warning.

### `failed to import: issue already exists`

You're trying to bootstrap a database with issues that conflict with existing
ones. Clear the local database and re-initialize from an export:

```bash
# DESTROYS the local database — export first if unsure
rm -rf .beads/embeddeddolt   # embedded mode (default)
rm -rf .beads/dolt           # server mode

issuegraph init --from-jsonl
```

### Imported children whose parent is gone

Bootstrapping from JSONL or pulling hierarchical issues (e.g., `bd-abc.1`) can
land children whose parent `bd-abc` no longer exists — typically after
`issuegraph delete` on a parent, a branch merge where one side deleted it, or an
incomplete import.

Import accepts these orphans rather than failing, so the children still arrive
and stay usable. Recreate the parent (or close out the orphaned children) once
the import finishes.

**Prevention:** use `issuegraph delete --cascade` to also delete children, and review
children first with `issuegraph children <parent-id>`.

### Old data returns after reset

`issuegraph admin reset --force` only removes **local** issuegraph data. Old issues can
return from configured Dolt remotes or from other machines that push after
you reset. For a complete clean slate, reset every clone (or clear the
remote's issuegraph data) before re-running `issuegraph init`.

If you previously used the removed legacy sync-branch feature, also delete
its branch and worktrees — see
[Worktrees: Legacy Cleanup](/reference/worktrees#legacy-cleanup).

### `issuegraph` shows 0 issues but the database has data

**Symptom (server mode):** All `issuegraph` commands return empty results even though
your data exists.

**Cause:** `issuegraph` is connecting to a different Dolt server or database than
expected — an empty "shadow" database on the wrong server.

**Diagnosis:**

```bash
# Check what mode and server issuegraph is using
cat .beads/metadata.json | grep -E "dolt_mode|dolt_server_port"

# Run server-mode health checks
issuegraph doctor --server

# Confirm what the connected database contains
issuegraph sql 'SELECT COUNT(*) FROM issues'
```

**Fix:** ensure your Dolt server is running from the correct data directory
and that `metadata.json` points at the right server and port. If a stale
`.beads/dolt/` directory exists alongside an external-server configuration,
it can shadow the real database — confirm your real data lives on the server
before removing the stale directory.

### Configured server unreachable (auto-start disabled)

**Symptom (server mode):** `issuegraph` returns "database not found on Dolt server"
when the configured server is down.

**Cause:** When `metadata.json` has an explicit `dolt_server_port`, issuegraph treats
the server as externally managed and intentionally disables auto-start —
spawning a different server would create a shadow database.

**Fix:**

```bash
# Start your configured Dolt server
issuegraph dolt start

# Or start manually with the correct data directory
dolt sql-server --host 127.0.0.1 --port 3307 --data-dir /path/to/your/dolt/data
```

If you want auto-start behavior, remove `dolt_server_port` from
`.beads/metadata.json`.

### Port conflicts with multiple projects

**Symptom (server mode):** Commands in a second project fail or connect to the
wrong database, and multiple `dolt sql-server` processes are running.

**Cause:** Each server-mode project starts its own Dolt server by default,
which can conflict on machines with many projects.

**Fix:** Enable shared server mode so all projects use a single Dolt server:

```bash
# Option 1: Machine-wide (add to ~/.bashrc or ~/.zshrc)
export BEADS_DOLT_SHARED_SERVER=1

# Option 2: Per-project
issuegraph config set dolt.shared-server true
```

After enabling, existing projects may need `issuegraph init --reinit-local -q` to
create their database on the shared server.

**Verify:** `issuegraph dolt status` from any project should show the same server,
port 3308, and `~/.beads/shared-server/` as the data directory.

### Multiple databases detected warning

issuegraph warns when it finds more than one `.beads` directory in your directory
hierarchy, marking the one in use with `▶` (usually the closest to your
current directory). Multiple databases risk working in the wrong one or
tracking the same work twice.

- **Nested projects (intentional):** this is supported — just note which
  database is active, or pin it explicitly.
- **Accidental duplicates:** export from the unwanted database
  (`issuegraph export -o issue-export.jsonl`), then remove its `.beads` directory.
- **Override selection:**

  ```bash
  # Point issuegraph at a specific .beads directory (recommended)
  export BEADS_DIR=/path/to/.beads

  # Legacy method (deprecated, points at the database file directly)
  export BEADS_DB=/path/to/db
  ```

### Circuit breaker: "server appears down, failing fast"

**Symptom (server mode):** Every `issuegraph` command fails with
`dolt circuit breaker is open: server appears down, failing fast (cooldown 30s)`,
persisting across repeated invocations.

**Cause:** The circuit breaker tripped after repeated connection failures.
Its state lives in a file under `/tmp/beads-circuit/` (named
`beads-dolt-circuit-<host>-<port>[-<db>].json`, keyed on host:port) and is
shared across all `issuegraph` processes. Once tripped, all commands to that host:port
are rejected until a successful probe resets it.

For issuegraph-managed local servers, `issuegraph dolt status` reports from the server's
PID file — a "running" status does not guarantee the server is actually
accepting connections on the expected port.

**Diagnosis:**

```bash
# Check circuit breaker state
cat /tmp/beads-circuit/beads-dolt-circuit-*.json

# Check if the Dolt server is actually listening
lsof -i :<port>

# Compare the configured port with what's running
cat .beads/metadata.json | grep port
```

**Fix:**

```bash
rm /tmp/beads-circuit/beads-dolt-circuit-*.json
issuegraph dolt stop
issuegraph dolt start
issuegraph list
```

On macOS, `/tmp` is a symlink to `/private/tmp`, which is not always cleared
on restart — the state file can persist across reboots.

## Dolt Server Issues

### Server not starting

```bash
# Check server health
issuegraph doctor

# Check server logs (server mode; embedded mode runs in-process, no server log)
cat .beads/dolt-server.log

# Restart the server
issuegraph dolt stop
issuegraph dolt start
```

### Version mismatch

After upgrading issuegraph:

```bash
issuegraph dolt stop
issuegraph dolt start
```

### Proxied-server mode: "dolt is older than the recommended minimum" warning

Managed proxied-server mode spawns an external `dolt` CLI it finds via
`BEADS_DOLT_BIN` or PATH (in that order — see
[Environment Variables](/reference/configuration#environment-variables); a
clone-local sidecar setting will slot between the two when the sidecar
reader lands in contract part 2). On
startup it probes that binary with `dolt version` and recommends
**dolt >= 2.0.0**: the 2026-07-25 cross-version compatibility matrix found
that cross-reading storage written by the issuegraph Dolt Go module (as opposed
to writing it, which older dolt CLIs can also do) requires dolt >= 2.0.0 —
dolt 1.85 can *serve* proxied mode but cannot *read* storage the module
wrote, and dolt 1.52.1 fails at both serving and reading.

This is a warning, not a hard failure — there is deliberately no hard
version floor, so an older dolt can still be used at your own risk. To
resolve it, install the pinned dolt version — see
[Which Dolt version to install](/architecture/dolt#which-dolt-version-to-install)
— and either update PATH or set `BEADS_DOLT_BIN` to the new binary's path.
Install that specific version rather than `latest`: 2.3.x is a newer release
that satisfies this warning but carries a
[separate data-operation defect](/architecture/dolt#which-dolt-version-to-install).

The advisory repeats at most once per day, not on every command: the probe
result and the warning timestamp are cached (keyed by the binary's path,
size, and mtime, in the user cache directory), so day-to-day `issuegraph` use stays
quiet while the reminder still resurfaces until the binary is upgraded.
Upgrading or replacing the dolt binary re-probes immediately.

If the probe can't parse `dolt version`'s output at all (most commonly a
dev/custom build with non-standard version output), that also only warns —
proxied-server mode still starts, since an unparseable version means "we
don't know", not "this is definitely broken". A genuinely missing or
broken `dolt` binary (not found, not executable, or the probe itself
fails/times out) is a hard error, not a warning.

### Proxied-server mode: closing a bead gate fails with "no local store available"

```
gate condition not satisfied: bead gate "bd-a1b2": no local store available (use --force to override)
```

`issuegraph close` on a bead gate refuses this way in proxied-server mode even when
the awaited bead is closed. Proxied-server commands run against a shared
`dolt sql-server` and never open a local store, so the close path has
nothing to read the awaited bead's status from.

`issuegraph gate check` does evaluate bead gates in proxied-server mode, and closes
the ones whose target has closed:

```bash
issuegraph gate check --type=bead      # closes bead gates whose awaited bead is closed
issuegraph close <gate-id> --force     # or close without verifying the condition
```

Tracked in [#5861](https://github.com/gastownhall/beads/issues/5861).

### Prefix routing: "proxy server store needs to be uow provider"

A lookup routed through the orchestrator's `routes.jsonl` prefix routes
fails with this error when the rig that owns the prefix is itself running in
proxied-server mode — routed reads cannot open a proxied-server rig. In an
all-proxied shared-server topology that applies to every routed target, so
cross-rig bead gates and a plain `issuegraph show <routed-id>` both dead-end here.

Run the command from the owning rig's own workspace instead, where the ID
resolves locally rather than through a route. Tracked in
[#5861](https://github.com/gastownhall/beads/issues/5861).

## Sync Issues

### Changes not syncing

```bash
# Force push to Dolt remote
issuegraph dolt push

# Check hooks
issuegraph hooks list
```

### Recovery from backup

```bash
# Restore from a Dolt backup
issuegraph backup restore [path] --force

# Or pull from Dolt remote
issuegraph dolt pull
```

### Merge conflicts

Dolt merges at the cell level, so concurrent changes conflict only when they
touch the same field of the same issue. Hash-based IDs mean different issues
never collide on ID.

```bash
# Check for and fix Dolt conflicts
issuegraph doctor --fix

# Re-push
issuegraph dolt push
```

See [Merge Conflicts](/recovery/merge-conflicts) for the full runbook.

## Git Hook Issues

### Hooks not running

```bash
# Check if installed
ls -la .git/hooks/

# Reinstall
issuegraph hooks install
```

### Hook errors

```bash
# Check hook script
cat .git/hooks/pre-commit

# Run manually
.git/hooks/pre-commit
```

### Hook timeout kills chained pre-commit hooks

**Symptom:** After `issuegraph hooks install`, chained pre-commit hooks (eslint,
prettier, ruff, etc.) stop running, with:
`beads: hook 'pre-commit' timed out after 300s -- continuing without beads`.

**Cause:** The issuegraph hook shim wraps `issuegraph hooks run` with an OS-level timeout.
Since `issuegraph hooks run` chains to your original hook internally, the timeout
covers both issuegraph's own work and your entire hook pipeline.

**Fix:** Increase the timeout (default 300 seconds):

```bash
# Add to ~/.bashrc or ~/.zshrc
export BEADS_HOOK_TIMEOUT=600  # 10 minutes (in seconds)
```

The value must be a positive whole number of seconds. Invalid values and zero
warn and fall back to 300 seconds. IssueGraph accepts `timeout` or `gtimeout` only
when a successful version probe identifies GNU coreutils; native Windows
`timeout.exe` is not compatible. If neither GNU timeout nor Perl is available,
the hook warns that it is running directly without a deadline.

GNU timeout sends `TERM`; on POSIX hosts, Perl's alarm applies to the direct
`issuegraph` process. Git for Windows Perl does not guarantee that alarm across
`exec`, so GNU coreutils is preferred there. TERM-resistant work and
descendant processes are not guaranteed to stop. After upgrading from a
version with the name-only timeout check, run `issuegraph hooks install` once to
refresh existing canonical sections.

### Permission denied on git hooks

Git hooks need execute permissions:

```bash
chmod +x .git/hooks/pre-commit
chmod +x .git/hooks/post-merge
chmod +x .git/hooks/post-checkout
```

### Corrupted symlinked `CLAUDE.md`

**Symptom:** Git reports `CLAUDE.md` as a symlink entry (mode `120000`), but
the indexed blob contains multi-line Markdown instead of a one-line symlink
target. On macOS this can make clones or checkouts fail.

This affects repositories corrupted by older setup behavior (fixed in
[#4192](https://github.com/gastownhall/beads/pull/4192)). To repair an
existing bad index entry:

```bash
# Confirm the bad entry: mode 120000 but Markdown content
git ls-files -s CLAUDE.md
git cat-file -p :CLAUDE.md | sed -n '1,5p'

# Convert the blob to a regular tracked file without changing content
sha=$(git rev-parse :CLAUDE.md)
git update-index --cacheinfo 100644,$sha,CLAUDE.md
git checkout-index -f -- CLAUDE.md

# Verify: first column should now be 100644
git ls-files -s CLAUDE.md
git diff -- CLAUDE.md
```

Commit the mode repair after review.

### "Branch already checked out" or unexpected `.git/beads-worktrees/`

Older issuegraph versions created hidden git worktrees for a removed sync-branch
feature; leftovers can lock branches (`fatal: 'main' is already checked out
at .../beads-worktrees/...`). Remove them:

```bash
rm -rf .git/beads-worktrees
rm -rf .git/worktrees/beads-*
git worktree prune
```

See [Worktrees: Legacy Cleanup](/reference/worktrees#legacy-cleanup).

## Dependency Issues

### `issuegraph ready` shows nothing but I have open issues

Those issues probably have open blockers:

```bash
# See blocked issues
issuegraph blocked

# Show the dependency tree (default max depth: 50)
issuegraph dep tree <issue-id>
issuegraph dep tree <issue-id> --max-depth 10

# Remove a blocking dependency if needed
issuegraph dep remove <from-id> <to-id>
```

Remember: only `blocks` dependencies affect ready work.

### Circular dependencies

issuegraph prevents dependency cycles, which break ready work detection:

```bash
# Detect cycles
issuegraph dep cycles

# Remove one dependency
issuegraph dep remove bd-A bd-B
```

See [Circular Dependencies](/recovery/circular-dependencies) for the full
runbook.

### Dependencies not showing up

```bash
# Show full issue details including dependencies
issuegraph show <issue-id>

# Visualize the dependency tree
issuegraph dep tree <issue-id>
```

Different dependency types have different meanings — only `blocks` gates
ready work. See [Dependencies](/core-concepts/dependencies).

## Performance Issues

### Slow queries

```bash
# Check database stats
issuegraph stats

# Check on-disk size
du -sh .beads/embeddeddolt   # embedded mode (default)
du -sh .beads/dolt           # server mode

# Preview compaction candidates
issuegraph admin compact --dry-run --all

# Compact if large
issuegraph admin compact --analyze
```

Consider splitting very large projects into multiple databases:

```bash
cd ~/project/component1 && issuegraph init --prefix comp1
cd ~/project/component2 && issuegraph init --prefix comp2
```

### High memory usage

```bash
# Run Dolt garbage collection to compact storage
issuegraph admin compact --dolt
```

If this or `issuegraph flatten` stops with `Error 1105 (HY000): context canceled`,
see [Storage reclaim fails with "context canceled"](#storage-reclaim-fails-with-context-canceled)
below.

### Storage reclaim fails with "context canceled"

`issuegraph flatten` and the Dolt-history compaction in `issuegraph admin compact` finish by
hard-resetting `main` onto a temporary branch; the merge-settle path behind
`issuegraph dolt pull` / `issuegraph sync` falls back to a hard reset when it abandons a
merge. On Dolt 2.3.x a few percent of freshly created databases come up with
`CALL DOLT_RESET('--hard')` broken for the life of the server process, so on
an affected database those commands stop with:

```
Error 1105 (HY000): context canceled
```

Nothing else looks wrong — ordinary queries, commits, soft resets,
`CALL DOLT_CLEAN()` and `CALL DOLT_CHECKOUT('.')` all still work — so the
problem only shows up when something needs a hard reset. Confirm with the
check in
[Which Dolt version to install](/architecture/dolt#which-dolt-version-to-install),
which also covers the fix: restarting `dolt sql-server` clears it for now,
and installing the pinned Dolt version keeps it clear.

This applies to server and proxied-server mode, which use the standalone
`dolt` CLI. Embedded mode links its own Dolt engine at the version pinned in
`go.mod` and is not affected by which `dolt` CLI is on your PATH.

## Agent Issues

### Agent creates duplicate issues

Agents may not realize an issue already exists. Prevention strategies:

- Have agents search first: `issuegraph list --json | grep "title"`
- Label auto-created issues: `issuegraph create "..." -l auto-generated`
- Consolidate duplicates: `issuegraph duplicate <dup-id> --of <canonical-id>` closes
  the duplicate with a reference to the canonical issue

### Agent gets confused by complex dependencies

Simplify the dependency structure:

```bash
# Check for overly complex trees
issuegraph dep tree <issue-id>

# Remove unnecessary dependencies
issuegraph dep remove <from-id> <to-id>

# Use labels instead of dependencies for loose relationships
issuegraph label add <issue-id> related-to-feature-X
```

### MCP server not working

```bash
# Verify the MCP server is installed
pip list | grep beads-mcp

# Check MCP configuration (Claude Desktop on macOS)
cat ~/Library/Application\ Support/Claude/claude_desktop_config.json

# Test that the CLI itself works
issuegraph version
issuegraph ready
issuegraph doctor
```

See [MCP Server](/integrations/mcp-server) for setup and configuration.

### Sandboxed environments (Codex, Claude Code, etc.)

Sandboxes that restrict process and network permissions can prevent issuegraph from
controlling a Dolt server, causing persistent "database out of sync" errors
or `issuegraph dolt stop` failing with "operation not permitted".

issuegraph auto-detects sandboxed environments and prints
`Sandbox detected, using direct mode`. If auto-detection fails, pass the
global `--sandbox` flag explicitly:

```bash
issuegraph --sandbox ready
issuegraph --sandbox create "Fix bug" -p 1
```

Sandbox mode disables Dolt auto-push so issuegraph works without server control or
network access. Sync manually once outside the sandbox:

```bash
issuegraph dolt push
```

If staleness errors persist, `issuegraph doctor --fix` forces a metadata refresh
(low risk — it updates tracking metadata, not issues). Background:
[GH#353](https://github.com/gastownhall/beads/issues/353).

## Platform-Specific Issues

### Windows: Path issues

```pwsh
# Check if issuegraph.exe is in PATH
where.exe issuegraph

# Add Go bin to PATH (permanently)
[Environment]::SetEnvironmentVariable(
    "Path",
    $env:Path + ";$env:USERPROFILE\go\bin",
    [EnvironmentVariableTarget]::User
)

# Reload PATH in current session
$env:Path = [Environment]::GetEnvironmentVariable("Path", "User")
```

### Windows: Firewall blocking the Dolt server

In server mode, the Dolt server listens on loopback TCP. Allow `issuegraph.exe`
through Windows Firewall: Windows Security → Firewall & network protection →
"Allow an app through firewall" → add `issuegraph.exe` for Private networks.

### Windows: Controlled Folder Access blocks `issuegraph init`

**Symptom:** `issuegraph init` hangs indefinitely with high CPU usage, and CTRL+C
doesn't work. Controlled Folder Access may block issuegraph without showing a
notification, making this hard to diagnose without the `-v` flag:

```pwsh
issuegraph init -v
# Error: failed to create .beads directory: mkdir .beads: The system cannot find the file specified
```

**Solution:** Whitelist `issuegraph.exe`: Windows Security → Virus & threat
protection → Ransomware protection → Controlled folder access → "Allow an
app through Controlled folder access" → browse to `issuegraph.exe` (typically
`%USERPROFILE%\go\bin\issuegraph.exe`). Then retry `issuegraph init`.

### Windows: `ENOENT` when a Node program spawns `bd`

**Symptom:** An editor extension, MCP server, or script that shells out to
`bd` fails on Windows with `spawn bd ENOENT`, even though `bd` runs fine in
the same terminal.

The npm package installs `bd` as a generated `bd.cmd` shim, not as an
executable named `bd`. Node's `execFile()` and `spawn()` run their target
directly instead of through a shell, so they never apply the `PATHEXT`
resolution that finds `bd.cmd` — and a batch file is not directly executable
in the first place.

**Solution:** Name the shim explicitly and give it a shell:

```js
const { execFile } = require('node:child_process');

const isWindows = process.platform === 'win32';

execFile(
  isWindows ? 'bd.cmd' : 'bd',
  ['ready', '--json'],
  { shell: isWindows },
  (err, stdout) => { /* ... */ },
);
```

To avoid a shell — and the argument quoting that comes with it — spawn the
native binary the shim wraps, at `node_modules/@beads/bd/bin/bd.exe`.

### Windows: `/tmp` paths silently land in the drive root

**Symptom:** An `issuegraph` command given a `/tmp/...` path reports success, but
the file is not where you look for it — it was written to `C:\tmp\...`.

`issuegraph.exe` is a native Windows binary and does not share Git Bash's emulated
POSIX filesystem. When a literal `/tmp/...` string reaches `issuegraph`, it resolves
against the current drive root.

In a default interactive Git Bash this usually does *not* happen — Git for
Windows converts standalone POSIX-path arguments to Windows paths before
`issuegraph.exe` sees them. The trap appears when that conversion is out of play:

- `MSYS_NO_PATHCONV=1` or `MSYS2_ARG_CONV_EXCL` is set (common in
  Docker-heavy environments)
- the path comes from a config value or a file, not a command-line argument
- `issuegraph` is spawned by a non-MSYS parent — a Node script, editor extension,
  or MCP server — which passes the string through verbatim:

```js
// From Node on Windows: no path conversion happens
execFile('bd.cmd', ['export', '-o', '/tmp/issues.jsonl'], { shell: true }, ...);
// bd writes C:\tmp\issues.jsonl — and exits 0
```

**Solution:** Hand `issuegraph` a Windows path — `os.tmpdir()` from Node,
`"$(cygpath -w /tmp)\issues.jsonl"` from Git Bash scripts. This applies to
any path argument, including `--db` and config values.

### macOS: Gatekeeper blocking execution

1. Verify the downloaded binary checksum matches the release `checksums.txt`.
2. If you used `scripts/install.sh`, note that macOS ad-hoc re-signing is
   opt-in (`BEADS_INSTALL_RESIGN_MACOS=1`).
3. Approve the binary:

```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine /usr/local/bin/issuegraph

# Or: System Preferences → Security & Privacy → General → "Allow anyway"
```

## Debug Environment Variables

issuegraph supports environment variables for debugging specific subsystems. Enable
them when troubleshooting or when requested by maintainers.

| Variable | Purpose | Output |
|----------|---------|--------|
| `BD_DEBUG` | General debug logging | stderr |
| `BD_DEBUG_RPC` | RPC communication between CLI and Dolt server | stderr |
| `BD_DEBUG_SYNC` | Sync and import timestamp protection | stderr |
| `BD_DEBUG_ROUTING` | Issue routing and multi-repo resolution | stderr |
| `BD_DEBUG_FRESHNESS` | Database file replacement detection | server log |

Set any of them to `1` to enable; `unset` to disable.

```bash
# General debugging
BD_DEBUG=1 issuegraph ready

# Capture debug output to a file
BD_DEBUG=1 issuegraph dolt push 2> debug.log

# Sync timestamp protection, e.g.:
# [debug] Protected bd-123: local=2024-01-20T10:00:00Z >= incoming=2024-01-20T09:55:00Z
BD_DEBUG_SYNC=1 issuegraph dolt push

# Freshness output goes to the server log (server mode), not stderr
BD_DEBUG_FRESHNESS=1 issuegraph dolt start
tail -f .beads/dolt-server.log | grep freshness
```

For multi-repo routing configuration, see [Routing](/multi-agent/routing).

## Getting Help

### Debug output

```bash
issuegraph --verbose list
```

### Logs

```bash
# Server mode (embedded mode runs in-process, no server log)
cat .beads/dolt-server.log
```

### System info

```bash
issuegraph info --json
```

### File an issue

```bash
# Include this info
issuegraph version
issuegraph info --json
uname -a
```

Report at: https://github.com/gastownhall/beads/issues — or ask in
[GitHub Discussions](https://github.com/gastownhall/beads/discussions).
