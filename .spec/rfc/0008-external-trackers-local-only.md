# RFC 0008: Keep tracker sync local by default (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

IssueGraph is local-first: normal runtime commands do not read issues from or write issues to network trackers. Git remains the default issue-data transport through ordinary repository files. RFC 0014 permits GitHub Issues access only through explicit configuration and manually invoked sync commands. Other network trackers remain unsupported.

## Problem

Disabling Dolt remotes alone does not prevent unexpected tracker access. Explicitly configured GitHub sync may transmit issue content, so it must remain isolated from normal local commands and must never start automatically.

## Goals

Remove or fail closed network operations for all trackers except the manually invoked GitHub Issues integration defined by RFC 0014. GitHub configuration must be absent by default; init, issue lifecycle commands, hooks, and background processes must never sync. Offline file import/export and all local issue operations remain available. Other tracker commands fail closed before loading credentials or creating clients.

## Non-goals

Does not remove local issue creation, queries, dependencies, file import/export, or snapshot restore. Does not prevent the user from using external tracker clients independently. Does not alter repository Git remotes, package installation downloads, or release publishing; those are not runtime issue synchronization. GitHub sync remains opt-in and explicit under RFC 0014.

## Design

Remove unsupported tracker sync commands from the supported CLI surface or return one stable local-only error before loading credentials, opening clients, resolving remote metadata, or making network calls. Keep GitHub tracker clients reachable only from explicit GitHub commands, using RFC 0014's explicit repository, dry-run/apply, mapping, and conflict rules. Remove automatic tracker sync from hooks and issue lifecycle paths. Keep file-based formats that operate entirely on local files. Remove obsolete integration documentation and examples from supported IssueGraph assets; retain historical records only where explicitly archived and marked obsolete. RFC [0003](0003-drop-remote-sync.md) separately removes Dolt remote replication.

## Acceptance

Normal local commands and all automatic paths make no tracker network calls. Unsupported tracker commands fail closed before credential access. GitHub import/export/sync is available only through RFC 0014's explicit commands and user opt-in. Tests prove local-only commands make no HTTP requests; normal local issue operations and file import/export still work; user Git operations remain unaffected; supported docs identify local files as the default issue-data transport.
