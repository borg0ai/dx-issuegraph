---
title: "issuegraph kv"
description: "Key-value store commands"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc kv`.

Commands for working with the issuegraph key-value store.

The key-value store is useful for storing flags, environment variables,
or other user-defined data that persists across sessions.

Examples:
  issuegraph kv set mykey myvalue    # Set a value
  issuegraph kv get mykey            # Get a value
  issuegraph kv clear mykey          # Delete a key
  issuegraph kv list                 # List all key-value pairs

```
issuegraph kv [command]
```

## issuegraph kv clear

Delete a key from the issuegraph key-value store.

Examples:
  issuegraph kv clear feature_flag
  issuegraph kv clear api_endpoint

```
issuegraph kv clear <key> [flags]
```

## issuegraph kv get

Get a value from the issuegraph key-value store.

Examples:
  issuegraph kv get feature_flag
  issuegraph kv get api_endpoint

```
issuegraph kv get <key> [flags]
```

## issuegraph kv list

List all key-value pairs in the issuegraph key-value store.

Examples:
  issuegraph kv list
  issuegraph kv list --json

```
issuegraph kv list [flags]
```

## issuegraph kv set

Set a key-value pair in the issuegraph key-value store.

This is useful for storing flags, environment variables, or other
user-defined data that persists across sessions.

Examples:
  issuegraph kv set feature_flag true
  issuegraph kv set api_endpoint https://api.example.com
  issuegraph kv set max_retries 3

```
issuegraph kv set <key> <value> [flags]
```
