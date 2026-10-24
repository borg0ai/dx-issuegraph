---
title: "issuegraph mail"
description: "Delegates mail operations to an external mail provider."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc mail`.

Delegates mail operations to an external mail provider.

Agents often type 'issuegraph mail' when working with issuegraph, but mail functionality
is typically provided by the orchestrator. This command bridges that gap
by delegating to the configured mail provider.

Configuration (checked in order):
  1. ISSUEGRAPH_MAIL_DELEGATE or ISSUEGRAPH_MAIL_DELEGATE environment variable
  2. 'mail.delegate' config setting (issuegraph config set mail.delegate "gt mail")

Examples:
  # Configure delegation (one-time setup)
  `export ISSUEGRAPH_MAIL_DELEGATE="gt mail"`
  # or
  issuegraph config set mail.delegate "gt mail"

  # Then use issuegraph mail as if it were gt mail
  issuegraph mail inbox                    # Lists inbox
  issuegraph mail send mayor/ -s "Hi"      # Sends mail
  issuegraph mail read msg-123             # Reads a message

```
issuegraph mail [subcommand] [args...] [flags]
```
