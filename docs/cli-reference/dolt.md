---
title: "issuegraph dolt"
description: "Configure and manage Dolt database settings and server lifecycle."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc dolt`.

Configure and manage Dolt database settings and server lifecycle.

IssueGraph uses a dolt sql-server for all database operations. The server is
auto-started transparently when needed. Use these commands for explicit
control or diagnostics.

Server lifecycle:
  issuegraph dolt start        Start the Dolt server for this project
  issuegraph dolt stop         Stop the Dolt server for this project
  issuegraph dolt status       Show Dolt server status

Configuration:
  issuegraph dolt show         Show current Dolt configuration with connection test
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

Flags for 'issuegraph dolt set':
  --update-config  Also write to config.yaml for team-wide defaults

Examples:
  issuegraph dolt set database myproject
  issuegraph dolt set host 192.168.1.100 --update-config
  issuegraph dolt set data-dir /home/user/.issuegraph-dolt/myproject
  issuegraph dolt test

```
issuegraph dolt [flags]
```

## issuegraph dolt clean-databases

Identify and drop leftover test and agent databases that accumulate
on the shared Dolt server from interrupted test runs and terminated agents.

Stale database prefixes: testdb_*, doctest_*, doctortest_*, issuegraph_pt*, issuegraph_vr*, issuegraph_t*

These waste server memory and can degrade performance under concurrent load.
Use --dry-run to see what would be dropped without actually dropping.

```
issuegraph dolt clean-databases [flags]
```

**Flags:**

```
      --dry-run   Show what would be dropped without dropping
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

```
issuegraph dolt pull [flags]
```

**Flags:**

```
      --remote string   Pull from a specific named remote instead of the default
```

## issuegraph dolt push

Push local Dolt commits to the configured remote.

Requires a Dolt remote to be configured in the database directory.
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
      --remote string   Push to a specific named remote instead of the default
```

## issuegraph dolt remote

Manage Dolt remotes for push/pull replication.

Subcommands:
  add &lt;name&gt; &lt;url&gt;   Add a new remote
  list               List all configured remotes
  remove &lt;name&gt;      Remove a remote

```
issuegraph dolt remote [flags]
```

### issuegraph dolt remote add

Add a Dolt remote

```
issuegraph dolt remote add <name> <url> [flags]
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

## issuegraph dolt set

Set a Dolt configuration value in metadata.json.

Keys:
  database  Database name (default: issue prefix or "issuegraph")
  host      Server host (default: 127.0.0.1)
  port      Server port (auto-detected; override with issuegraph dolt set port &lt;N&gt;)
  user      MySQL user (default: root)
  data-dir  Custom dolt data directory (absolute path; default: .issuegraph/dolt)

Use --update-config to also write to config.yaml for team-wide defaults.

Examples:
  issuegraph dolt set database myproject
  issuegraph dolt set host 192.168.1.100
  issuegraph dolt set port 3307 --update-config
  issuegraph dolt set data-dir /home/user/.issuegraph-dolt/myproject

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

```
issuegraph dolt start [flags]
```

## issuegraph dolt status

Show the status of the Dolt engine for the current project.

In embedded mode, reports that the Dolt engine runs in-process and shows
the on-disk data directory. For issuegraph-managed (local) servers, displays
PID, port, and data directory from the local PID file. For externally-
managed servers — either a remote dolt_server_host or a local server
managed outside issuegraph (dolt.auto-start: false, e.g. an orchestrator-shared
sql-server) — pings the configured endpoint via SQL and reports
reachability, server version, and database.

```
issuegraph dolt status [flags]
```

## issuegraph dolt stop

Stop the dolt sql-server managed by issuegraph for the current project.

This sends a graceful shutdown signal. The server will restart automatically
on the next issuegraph command unless auto-start is disabled.

```
issuegraph dolt stop [flags]
```

**Flags:**

```
      --force   Force stop the server
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
