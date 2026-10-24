# Recovery Playbooks

The recovery playbooks moved to [docs/recovery/](recovery/index.md), rendered
on the docs site at
[https://github.com/borg0ai/dx-issuegraph/tree/main/docs/recovery/](https://github.com/borg0ai/dx-issuegraph/tree/main/docs/recovery/).

> **Why this file exists:** released `issuegraph` binaries print URLs into this file —
> v1.1.0's Dolt merge-refusal guidance (`printAncestorPKMismatchGuidance`,
> `modules/cli/dolt.go` at tag `v1.1.0`) prints `docs/RECOVERY.md#pk-fork-refused`.
> This stub keeps those printed links landing on a page that points at the
> live playbooks. Do not delete it while released binaries still print it; it
> is a deliberate exception to the no-pointer-stubs rule, registered in
> `test/docsync`.
