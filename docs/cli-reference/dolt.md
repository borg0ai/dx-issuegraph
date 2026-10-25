---
title: "issuegraph dolt"
description: "Configure Dolt database settings"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc dolt`.

Configure and manage Dolt database settings and server lifecycle.

IssueGraph runs Dolt embedded (in-process) by default: there is no sql-server and
nothing is auto-started. A database only uses a dolt sql-server when it is
configured for one: shared-server mode, an explicit server mode, or a
non-localhost dolt_server_host. The server-only commands below fail with
"not supported in embedded mode (no Dolt server)" on an embedded database.

Server lifecycle (server mode only):
  issuegraph dolt start        Start the Dolt server for this project
  issuegraph dolt stop         Stop the Dolt server for this project

Diagnostics (both modes):
  issuegraph dolt status       Show Dolt engine status (embedded: in-process, data dir)
  issuegraph dolt show         Show current Dolt configuration with connection test

Configuration (server mode only):
  issuegraph dolt set &lt;k&gt; &lt;v&gt;  Set a configuration value
  issuegraph dolt test         Test server connection

Version control:
  issuegraph dolt commit       Commit pending changes
  issuegraph dolt push         Push commits to Dolt remote
  issuegraph dolt pull         Pull commits from Dolt remote

Remote management:
  issuegraph dolt remote add &lt;name&gt; &lt;url&gt;   Add a Dolt remote
  issuegraph dolt remote list                List configured remotes
  issuegraph dolt remote remove &lt;name&gt;       Remove a Dolt remote

Configuration keys for 'issuegraph dolt set':
  database  Database name (default: issue prefix or "issuegraph")
  host      Server host (default: 127.0.0.1)
  port      Server port (auto-detected; override with issuegraph dolt set port &lt;N&gt;)
  user      MySQL user (default: root)
  data-dir  Custom dolt data directory (absolute path; default: .issuegraph/dolt)

Remote server authentication (password + TLS) is NOT stored via 'issuegraph dolt set'
(keeps secrets out of metadata.json). Configure them with:

  ISSUEGRAPH_DOLT_PASSWORD       Server password (highest priority)
  ISSUEGRAPH_DOLT_SERVER_TLS     Enable TLS (set to "1" or "true")
  ISSUEGRAPH_DOLT_SERVER_USER    MySQL user override (else use 'issuegraph dolt set user')
  ISSUEGRAPH_CREDENTIALS_FILE    Optional path to credentials file

  Default credentials file: ~/.config/issuegraph/credentials (Linux/macOS)
                            %APPDATA%\issuegraph\credentials (Windows)
  Format (INI, section = host:port of the resolved connection):
    [127.0.0.1:3307]
    password = secret

  Password resolution: ISSUEGRAPH_DOLT_PASSWORD → credentials [host:port] → empty.
  Full reference: docs/architecture/dolt.md (Environment Variables / Credentials).

Flags for 'issuegraph dolt set':
  --update-config  Also write to config.yaml for team-wide defaults

Examples:
  issuegraph dolt set database myproject
  issuegraph dolt set host 192.168.1.100 --update-config
  issuegraph dolt set data-dir /home/user/.issuegraph-dolt/myproject
  `export ISSUEGRAPH_DOLT_PASSWORD=... ISSUEGRAPH_DOLT_SERVER_TLS=1`
  issuegraph dolt test

```
issuegraph dolt [command]
```

## issuegraph dolt clean-databases

Identify and drop leftover test and agent databases that accumulate
on the shared Dolt server from interrupted test runs and terminated agents.

Stale database prefixes: testdb_*, issuegraph_test*, issuegraph_pt*, issuegraph_vr*, doctest_*, doctortest_*, benchdb_*

These waste server memory and can degrade performance under concurrent load.
Use --dry-run to see what would be dropped without actually dropping.

DROP DATABASE only marks a database as dropped; Dolt keeps its directory
under .dolt_dropped_databases/ so it can be restored with
CALL DOLT_UNDROP(name) until an explicit purge — disk is not reclaimed
until then. Pass --purge-dropped to run CALL DOLT_PURGE_DROPPED_DATABASES()
after cleanup.

--purge-dropped is SERVER-GLOBAL and IRREVERSIBLE. Dolt has no way to scope
the purge to only the databases this run dropped: it permanently deletes
every dropped-but-not-yet-purged database on the server, including ones
dropped by something else entirely (e.g. an operator's accidental
DROP DATABASE on an unrelated database that was still recoverable via
DOLT_UNDROP). It also purges pre-existing residue from earlier
clean-databases runs even if this run finds no stale databases to drop.
Only pass it when nothing else on the server may be relying on DOLT_UNDROP
recovery.

```
issuegraph dolt clean-databases [flags]
```

**Flags:**

```
      --dry-run         Show what would be dropped without dropping
      --purge-dropped   After dropping, also run CALL DOLT_PURGE_DROPPED_DATABASES() — server-global and irreversible, see --help
```

## issuegraph dolt commit

Create a Dolt commit from any uncommitted changes in the working set.

This is the primary commit point for batch mode. When auto-commit is set to
"batch", changes accumulate in the working set across multiple issuegraph commands and
are committed together here with a descriptive summary message.

Also useful before push operations that require a clean working set, or when
auto-commit was off or changes were made externally.

For more options (--stdin, custom messages), see: issuegraph vc commit

```
issuegraph dolt commit [flags]
```

**Flags:**

```
  -m, --message string   Commit message (default: auto-generated)
```

## issuegraph dolt killall

Find and kill orphan dolt sql-server processes not tracked by the
canonical PID file for the current repo's Dolt data directory.

Under an orchestrator, the canonical server lives at $GT_ROOT/.issuegraph/. Any other
dolt sql-server processes using that shared data directory are considered
orphans and will be killed.

In standalone mode, only dolt sql-server processes using the current
project's Dolt data directory are eligible for cleanup. Other projects'
servers are preserved.

```
issuegraph dolt killall [flags]
```

## issuegraph dolt pull

Pull commits from the configured Dolt remote into the local database.

Requires a Dolt remote to be configured in the database directory.
For Hosted Dolt, set DOLT_REMOTE_USER and DOLT_REMOTE_PASSWORD environment
variables for authentication.

Use --remote to pull from a specific named remote instead of the default.
The remote must already exist (see 'issuegraph dolt remote add').

Use --strategy ours|theirs to resolve conflicts the auto-resolver declines
(e.g. both sides edited the same issue since the last sync) instead of
aborting the pull for manual resolution. Embedded storage only (#4992); on
server-mode/sql-server storage use 'issuegraph conflicts resolve' after a pull that
reports conflicts.

```
issuegraph dolt pull [flags]
```

**Flags:**

```
      --remote string     Pull from a specific named remote instead of the default
      --strategy string   Conflict resolution strategy for conflicts the auto-resolver declines: 'ours' or 'theirs' (embedded storage only, #4992)
```

## issuegraph dolt push

Push local Dolt commits to the configured remote.

Requires a Dolt remote to be configured in the database directory. With no
remote configured, issuegraph can adopt one derived from git origin — only with
consent: interactively, or via --yes; --no-adopt or ISSUEGRAPH_NO_REMOTE_ADOPT=1
disables adoption entirely.
For Hosted Dolt, set DOLT_REMOTE_USER and DOLT_REMOTE_PASSWORD environment
variables for authentication.

Use --force to overwrite remote changes (e.g., when the remote has
uncommitted changes in its working set).

Use --remote to push to a specific named remote instead of the default.
The remote must already exist (see 'issuegraph dolt remote add').

```
issuegraph dolt push [flags]
```

**Flags:**

```
      --force           Force push (overwrite remote changes)
      --no-adopt        Never derive a Dolt remote from git origin (also ISSUEGRAPH_NO_REMOTE_ADOPT=1)
      --remote string   Push to a specific named remote instead of the default
  -y, --yes             Consent to adopting a Dolt remote derived from git origin when none is configured
```

## issuegraph dolt remote

Manage Dolt remotes for push/pull replication.

Subcommands:
  add &lt;name&gt; &lt;url&gt;     Add a new remote
  list                 List all configured remotes
  remove &lt;name&gt;        Remove a remote
  reset-data &lt;name&gt;    Replace a remote's data plane after a history squash

```
issuegraph dolt remote [command]
```

### issuegraph dolt remote add

Add a Dolt remote

```
issuegraph dolt remote add <name> <url> [flags]
```

**Flags:**

```
      --allow-git-origin   Allow adding a Dolt remote whose URL matches the git origin (proceed with a warning instead of aborting)
```

### issuegraph dolt remote list

List configured Dolt remotes

```
issuegraph dolt remote list [flags]
```

### issuegraph dolt remote remove

Remove a Dolt remote

```
issuegraph dolt remote remove <name> [flags]
```

### issuegraph dolt remote reset-data

Replace a Dolt remote's stored data with a fresh copy of local HEAD.

After a history squash (see the History Bloat recovery runbook), a plain
'issuegraph dolt push --force' re-points the remote's refs but deletes nothing:
Dolt remotes accumulate chunks monotonically, so the remote keeps the full
pre-squash store. This command rebuilds the remote's data plane so it holds
only live chunks:

  - Git-backed remotes (issue data riding a git remote under refs/dolt/data):
    deletes the Dolt data refs on the git remote, then force-pushes to
    rebuild a fresh store. Code branches are untouched.
  - Native file remotes (file:// paths): clears the store directory, then
    force-pushes to rebuild it.
  - Cloud/hosted remotes (aws://, gs://, dolthub://, ...): issuegraph cannot clear
    the stored data safely — replace the remote with a fresh URL or prefix:
      issuegraph dolt remote remove &lt;name&gt;
      issuegraph dolt remote add &lt;name&gt; &lt;fresh-url&gt;
      issuegraph dolt push --force

This rewrites the remote's data plane. Every other clone must re-clone from
the reset remote (that is already true after the squash itself). Refuses to
run with uncommitted working-set changes: the rebuilt remote holds exactly
HEAD, and anything uncommitted would not be part of it.

Examples:
  issuegraph dolt remote reset-data origin          # prompts for confirmation
  issuegraph dolt remote reset-data origin --yes    # no prompt (scripts, agents)

```
issuegraph dolt remote reset-data <name> [flags]
```

**Flags:**

```
  -y, --yes   Skip the confirmation prompt (required in non-interactive use)
```

## issuegraph dolt set

Set a Dolt configuration value in metadata.json.

Keys:
  database  Database name (default: issue prefix or "issuegraph")
  host      Server host (default: 127.0.0.1)
  port      Server port (auto-detected; override with issuegraph dolt set port &lt;N&gt;)
  user      MySQL user (default: root)
  data-dir  Custom dolt data directory (absolute path; default: .issuegraph/dolt)

There is no 'password' or 'tls' key here on purpose — secrets and TLS must
not land in metadata.json. Use environment variables or the credentials file:

  ISSUEGRAPH_DOLT_PASSWORD     Server password (highest priority)
  ISSUEGRAPH_DOLT_SERVER_TLS   Enable TLS ("1" or "true")
  ISSUEGRAPH_CREDENTIALS_FILE  Optional override path for credentials

  Default credentials file: ~/.config/issuegraph/credentials
  Format:
    [host:port]
    password = secret

  See: issuegraph dolt --help and docs/architecture/dolt.md

Use --update-config to also write to config.yaml for team-wide defaults.

Examples:
  issuegraph dolt set database myproject
  issuegraph dolt set host 192.168.1.100
  issuegraph dolt set port 3307 --update-config
  issuegraph dolt set data-dir /home/user/.issuegraph-dolt/myproject
  `export ISSUEGRAPH_DOLT_PASSWORD=... ISSUEGRAPH_DOLT_SERVER_TLS=1`

```
issuegraph dolt set <key> <value> [flags]
```

**Flags:**

```
      --update-config   Also write to config.yaml for team-wide defaults
```

## issuegraph dolt show

Show current Dolt configuration with connection status

```
issuegraph dolt show [flags]
```

## issuegraph dolt start

Start a dolt sql-server for the current issuegraph project.

The server runs in the background on a per-project port derived from the
project path. PID and logs are stored in .issuegraph/.

The server auto-starts transparently when needed, so manual start is rarely
required. Use this command for explicit control or diagnostics.

Not available in proxied-server mode: there the proxy owns its dolt backend's
lifecycle and starts it on demand, so a server started here would be a second,
unsupervised one over the same data directory. Use 'issuegraph dolt status' to see what
is running and 'issuegraph dolt stop' to shut it down.

```
issuegraph dolt start [flags]
```

## issuegraph dolt status

Show the status of the Dolt engine for the current project.

In embedded mode, reports that the Dolt engine runs in-process and shows
the on-disk data directory. For issuegraph-managed (local) servers, displays
PID, port, and data directory from the local PID file. For externally-
managed servers — a shared server (dolt.shared-server: true), a remote
dolt_server_host, or a local server managed outside issuegraph (dolt.auto-start:
false, e.g. an orchestrator-shared sql-server) — pings the configured
endpoint via SQL and reports reachability, server version, and database.

In proxied-server mode, reports the proxy (the endpoint every issuegraph command
connects through) and the dolt backend behind it separately, read from the
proxy's own process records. Reporting starts nothing: a quiesced workspace
is shown as not running, and the next issuegraph command launches it.

```
issuegraph dolt status [flags]
```

## issuegraph dolt stop

Stop the dolt sql-server managed by issuegraph for the current project.

This sends a graceful shutdown signal. The server will restart automatically
on the next issuegraph command unless auto-start is disabled.

For a managed proxied server, --force can recover unverifiable or legacy
process records (both the proxy and its backend) only after each live process
executable is matched to issuegraph or dolt and its command line ties it to this
workspace. In that recovery path, force still refuses to signal a process
whose executable identity cannot be matched to issuegraph or dolt, or whose workspace
scope cannot be established.

```
issuegraph dolt stop [flags]
```

**Flags:**

```
      --force   Force stop (proxied recovery still requires a issuegraph/dolt executable match)
```

## issuegraph dolt test

Test the connection to the configured Dolt server.

This verifies that:
  1. The server is reachable at the configured host:port
  2. The connection can be established

Use this before switching to server mode to ensure the server is running.

```
issuegraph dolt test [flags]
```
