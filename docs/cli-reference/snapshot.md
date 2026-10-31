---
title: "issuegraph snapshot"
description: "Work with the Git-tracked issue snapshot"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc snapshot`.

The snapshot is .issuegraph/issues.jsonl plus .issuegraph/manifest.json.
Commit and move those two files with Git yourself; IssueGraph never runs Git
or contacts a remote.

```
issuegraph snapshot [command]
```

## issuegraph snapshot restore

Validate .issuegraph/issues.jsonl against .issuegraph/manifest.json, then in one
transaction upsert every snapshot issue and delete local issues the manifest
does not list. A missing manifest, corrupt file or digest mismatch fails
without changing the database.

```
issuegraph snapshot restore [flags]
```
