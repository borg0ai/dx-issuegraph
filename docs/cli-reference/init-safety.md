---
title: "issuegraph init-safety"
description: "Explain issuegraph init flag semantics and the destroy-token format"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc init-safety`.

issuegraph init flag safety contract.

Every issuegraph init invocation resolves project_id from exactly one explicitly
named source (local reinit, remote adoption, or a fresh mint). When the
source is ambiguous, issuegraph init refuses.

FLAG SURFACE

  issuegraph init                       Mint a new identity. Bootstraps from
                                origin if it has refs/dolt/data.

  issuegraph init --reinit-local        Re-initialize local .issuegraph/ over existing
                                local data. Does NOT authorize discarding
                                remote history. If origin has Dolt data
                                this will refuse — pair with
                                --discard-remote to override.

  issuegraph init --reinit-local \      Discard the remote's Dolt history and
      --discard-remote          replace it with the local reinit. First
                                issuegraph dolt push after this will be a
                                history-replacing force-push.

  issuegraph init --force               Deprecated alias for --reinit-local.
                                Kept working for ≥2 releases.

  issuegraph init --from-jsonl          Import from configured import.path. If
                                origin has Dolt data, this refuses unless
                                --discard-remote authorizes replacing that
                                remote history.

ADOPTING A REMOTE

  If you want to use the remote's existing history, use:

      issuegraph bootstrap

  issuegraph init will automatically suggest this when a remote is detected.

DESTROY-TOKEN (non-interactive only)

  When running with no TTY (CI, agents, piped input), a destructive
  re-init requires an explicit --destroy-token value. That covers both
  --discard-remote and --reinit-local over existing issues. The token
  format is:

      DESTROY-&lt;issue-prefix&gt;

  For example, if your issue prefix is "issuegraph", the token is "DESTROY-issuegraph":

      issuegraph init --reinit-local --discard-remote --destroy-token=DESTROY-issuegraph

  In interactive (TTY) mode you confirm via a typed prompt instead. The
  token is not echoed by issuegraph's runtime error messages — this is a
  deliberate guard against pattern-matched one-liners (see
  engdocs/adr/0002-init-safety-invariants.md).

EXIT CODES

  10    refused: remote has Dolt history and you selected local history
        without --discard-remote
  11    refused: existing local data and you declined the destroy confirm
        (interactive mode only)
  12    refused: destructive re-init (--discard-remote, or --reinit-local
        over existing issues) without a valid --destroy-token
        (non-interactive mode); also returned when the interactive
        --discard-remote typed-token confirmation is declined

RECOVERY

  If you hit a refusal, see docs/recovery/init-safety.md for step-by-step recovery
  playbooks for each exit code.


```
issuegraph init-safety [flags]
```
