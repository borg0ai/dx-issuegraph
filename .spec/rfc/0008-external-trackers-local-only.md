# RFC 0008: Disable external tracker network sync (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

IssueGraph is local-only: runtime commands do not read issues from or write issues to GitHub, GitLab, Jira, Linear, Notion, Azure DevOps, or other network trackers. Git remains the user's only issue-data transport, through ordinary repository files.

## Problem

Disabling Dolt remotes alone does not make the product local-only. Explicit tracker sync commands can still transmit local issue content to external services or import remote issue content into the local graph.

## Goals

Remove or fail closed all runtime tracker network operations, including pull/import-from-remote, push/export-to-remote, bidirectional sync, link resolution, remote label/state lookup, and automatically triggered sync hooks. Offline file import/export remains available. Errors explain that IssueGraph stores issues locally and that users can version local files with Git.

## Non-goals

Does not remove local issue creation, queries, dependencies, file import/export, or snapshot restore. Does not prevent the user from using external tracker clients independently. Does not alter repository Git remotes, package installation downloads, or release publishing; those are not runtime issue synchronization.

## Design

Remove tracker sync commands from the supported CLI surface or return one stable local-only error before loading credentials, opening clients, resolving remote metadata, or making network calls. Remove automatic tracker sync from hooks and create/update lifecycle paths. Keep file-based formats that operate entirely on local files. Remove obsolete integration documentation and examples from supported IssueGraph assets; retain historical records only where explicitly archived and marked obsolete. RFC [0003](0003-drop-remote-sync.md) separately removes Dolt remote replication.

## Acceptance

Every external tracker command and automatic path either no longer exists or exits nonzero with the stable local-only error; tests prove no credentials, HTTP clients, or network calls are attempted; normal local create/read/update/delete and file import/export still work; user Git operations remain unaffected; supported docs describe Git-tracked local files as the only issue-data transport.
