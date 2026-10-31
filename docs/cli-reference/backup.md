---
title: "issuegraph backup"
description: "Back up your issuegraph database"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc backup`.

Back up your issuegraph database to a local filesystem destination.

This is a Dolt-native database backup. It preserves the database state,
including tables, branches, commit history, and working-set data. This is
different from 'issuegraph export', which writes issue records to JSONL for migration
and interoperability.

Commands:
  issuegraph backup init &lt;path&gt;    Set up a local filesystem backup destination
  issuegraph backup sync           Write to configured local backup destination
  issuegraph backup restore [path] Restore from a backup directory
  issuegraph backup remove         Remove backup destination
  issuegraph backup status         Show backup status

Remote backup destinations are disabled.

Auto-backup default:
  When backup.enabled is unset, auto-backup turns ON in embedded mode if a
  git remote exists, and stays OFF in sql-server / shared-server mode. In
  server mode many issuegraph clients share one Dolt server, and each would register
  a server-side backup remote under the same name pointing at its own local
  dir and full-sync the whole database — a self-amplifying storm. To back up
  a shared server, run 'issuegraph backup' explicitly (or set backup.enabled=true and
  coordinate destinations). 'issuegraph config get backup.enabled' shows the effective
  value and its source.

  On a proxied-server workspace that opt-in does not apply: auto-backup has
  no post-command hook on that path, so backup.enabled=true is inert there.
  The five backup verbs above are honored on a managed-local proxied server —
  run 'issuegraph backup sync' explicitly to back one up.

```
issuegraph backup [command]
```

## issuegraph backup init

Configure a local filesystem path as a backup destination.

The path can be a local directory on this machine or a mounted NAS. Remote
destinations are disabled. If a destination was previously configured, it is
updated to the new path.

Filesystem examples:
  issuegraph backup add /mnt/usb/issuegraph-backup
  issuegraph backup add ~/Dropbox/issuegraph-backup

After adding, run 'issuegraph backup sync' to write a local backup.

```
issuegraph backup init <path> [flags]
```

**Aliases:** add

## issuegraph backup remove

Remove the configured backup destination.

This unregisters the backup remote from Dolt and removes the local
backup configuration. The backup data at the destination is not deleted.

```
issuegraph backup remove [flags]
```

**Aliases:** rm

## issuegraph backup restore

Restore the issuegraph database from a Dolt-native backup.

By default, reads from .issuegraph/backup/ (or the configured backup directory).
Optionally specify a path to a directory containing a Dolt backup.

This restores a full database backup created by 'issuegraph backup sync' or an
equivalent Dolt backup. JSONL files produced by 'issuegraph export' are issue exports,
not restore targets for this command.

Use --force to overwrite an existing database with the backup contents.

The database must already be initialized (run 'issuegraph init' first if needed).
To initialize and restore in one step, use: issuegraph init &amp;&amp; issuegraph backup restore

```
issuegraph backup restore [path] [flags]
```

**Flags:**

```
      --force   Overwrite existing database with backup contents
```

## issuegraph backup status

Show last backup status

```
issuegraph backup status [flags]
```

## issuegraph backup sync

Write the current issuegraph database to the configured local backup destination.

This copies the entire database state (all branches, full history) to the
filesystem path configured with 'issuegraph backup init'. Remote destinations
are disabled.

The backup is atomic — if the sync fails, the previous backup state is preserved.

Run 'issuegraph backup init &lt;path&gt;' first to configure a destination.

```
issuegraph backup sync [flags]
```
