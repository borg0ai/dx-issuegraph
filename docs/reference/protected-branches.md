---
title: Protected branches
description: How IssueGraph stores and shares issue snapshots with Git.
---

IssueGraph writes issue data to its local database and refreshes tracked files in
`.issuegraph/`. It does not create Git commits or push issue data to a Dolt
remote. Use your normal Git workflow to commit and share repository snapshots.

```bash
issuegraph init
issuegraph create "Implement feature X" -t feature -p 1
git status --short .issuegraph/
git add .issuegraph/
git commit -m "Update issue snapshot"
```

GitHub, GitLab, and Bitbucket branch protection applies to those ordinary Git
commits. IssueGraph does not bypass protected branches or push a separate issue
ref. Commands for remote issue sync, including `issuegraph dolt push` and
`issuegraph dolt pull`, fail before opening the database.

Linked Git worktrees in one checkout share the local database. Separate clones
share only tracked snapshot files through Git; they do not share Dolt history.

## See also

- [Git worktrees guide](/reference/worktrees)
- [Git integration](/reference/git-integration)
- [Sync setup](/getting-started/sync-setup)
