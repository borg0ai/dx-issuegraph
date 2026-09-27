---
title: Local Issue Data and Git Snapshots
description: How IssueGraph stores issues locally and how users carry them with Git
---

IssueGraph keeps its working issue database on the local machine. The CLI reads
and writes that database for `issuegraph list`, `show`, `ready`, and other
commands. It also maintains a complete snapshot under `.issuegraph/` for the
user's repository.

## Git is the transport

The snapshot consists of `.issuegraph/issues.jsonl` and
`.issuegraph/manifest.json`. Together they represent the full issue set,
including deletions. Add these files to ordinary Git commits to carry issues to
another clone. Git handles branch, pull, push, and merge operations when the
user invokes them.

IssueGraph itself does not execute Git network commands, configure Git
remotes, or send issue data to DoltHub, S3, GCS, or external issue trackers.
There is no product-level remote sync command.

## Restoring a snapshot

Restore validates both files before changing the local database. The manifest
defines the complete set of issue IDs: records absent from it are deleted from
the restored local database. A missing or invalid manifest causes restore to
fail without changing data.

The local Dolt database remains the runtime query store. IssueGraph does not
query the JSONL snapshot for ordinary reads.

## Hooks

Repository-native hooks may run local checks or refresh the snapshot. They do
not call remote issue services or push data. Hook setup is explicit; installing
JavaScript dependencies does not rewrite the user's Git configuration.

## File import and export

File-based import and export remain local operations for migration and
interchange. A standalone JSONL import is not a full restore because it has no
manifest and cannot express deletion of records omitted from the file.
