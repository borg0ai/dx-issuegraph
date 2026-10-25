---
title: "issuegraph remember"
description: "Store a persistent memory"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc remember`.

Store a memory that persists across sessions and account rotations.

Memories are injected at prime time (issuegraph prime) so you have them
in every session without manual loading.

The positional arg is the memory CONTENT (the key is auto-generated from it
unless --key is given). As a convenience, if the arg is a bare key naming an
existing memory, it is RECALLED instead of stored (same as 'issuegraph recall');
a bare key naming nothing is refused. Use --key to store slug-like content.

Examples:
  issuegraph remember "always run tests with -race flag"
  issuegraph remember "Dolt phantom DBs hide in three places" --key dolt-phantoms
  issuegraph remember "auth module uses JWT not sessions" --key auth-jwt
  issuegraph remember dolt-phantoms        # bare existing key: reads it (= issuegraph recall)

```
issuegraph remember "<insight>" [flags]
```

**Flags:**

```
      --key string   Explicit key for the memory (auto-generated from content if not set). If a memory with this key already exists, it will be updated in place
```
