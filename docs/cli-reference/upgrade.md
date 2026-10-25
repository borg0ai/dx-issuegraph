---
title: "issuegraph upgrade"
description: "Check and manage issuegraph version upgrades"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc upgrade`.

Commands for checking issuegraph version upgrades and reviewing changes.

The upgrade command helps you stay aware of issuegraph version changes:
  - issuegraph upgrade status: Check if issuegraph version changed since last use
  - issuegraph upgrade review: Show what's new since your last version
  - issuegraph upgrade ack: Acknowledge the current version

Version tracking is automatic - issuegraph updates metadata.json on every run.

```
issuegraph upgrade [command]
```

## issuegraph upgrade ack

Mark the current issuegraph version as acknowledged.

This updates metadata.json to record that you've seen the current
version. Mainly useful after reviewing upgrade changes to suppress
future upgrade notifications.

Note: Version tracking happens automatically, so you don't need to
run this command unless you want to explicitly mark acknowledgement.

Examples:
  issuegraph upgrade ack
  issuegraph upgrade ack --json

```
issuegraph upgrade ack [flags]
```

## issuegraph upgrade review

Show what's new in issuegraph since the last version you used.

Unlike 'issuegraph info --whats-new' which shows the last 3 versions,
this command shows ALL changes since your specific last version.

If you're upgrading from an old version, you'll see the complete
changelog of everything that changed since then.

Examples:
  issuegraph upgrade review
  issuegraph upgrade review --json

```
issuegraph upgrade review [flags]
```

## issuegraph upgrade status

Check if issuegraph has been upgraded since you last used it.

This command uses the version tracking that happens automatically
at startup to detect if issuegraph was upgraded.

Examples:
  issuegraph upgrade status
  issuegraph upgrade status --json

```
issuegraph upgrade status [flags]
```
