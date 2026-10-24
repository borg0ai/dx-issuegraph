---
title: "issuegraph backup"
description: "Back up your issuegraph database for off-machine recovery."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc backup`.

Back up your issuegraph database for off-machine recovery.

This is a Dolt-native database backup. It preserves the database state,
including tables, branches, commit history, and working-set data. This is
different from 'issuegraph export', which writes issue records to JSONL for migration
and interoperability.

Commands:
  issuegraph backup init &lt;path&gt;    Set up a backup destination (filesystem or DoltHub)
  issuegraph backup sync           Push to configured backup destination
  issuegraph backup restore [path] Restore from a backup directory
  issuegraph backup remove         Remove backup destination
  issuegraph backup status         Show backup status

DoltHub is recommended for cloud backup:
  issuegraph backup init https://doltremoteapi.dolthub.com/&lt;user&gt;/&lt;repo&gt;
  Set DOLT_REMOTE_USER and DOLT_REMOTE_PASSWORD for authentication.

```
issuegraph backup [flags]
```

## issuegraph backup init

Configure a filesystem path or URL as a backup destination.

The path can be a local directory (external drive, NAS, Dropbox folder) or a
DoltHub remote URL. If the destination was previously configured, it is
updated to the new path.

Filesystem examples:
  issuegraph backup add /mnt/usb/issuegraph-backup
  issuegraph backup add ~/Dropbox/issuegraph-backup

DoltHub (recommended for cloud backup):
  issuegraph backup add https://doltremoteapi.dolthub.com/myuser/issuegraph-backup

After adding, run 'issuegraph backup sync' to push your data.

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

Sync the current issuegraph database to the configured Dolt backup destination.

This pushes the entire database state (all branches, full history) to the
backup location configured with 'issuegraph backup init'.

The backup is atomic — if the sync fails, the previous backup state is preserved.

Run 'issuegraph backup init &lt;path&gt;' first to configure a destination.

```
issuegraph backup sync [flags]
```
