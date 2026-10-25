# IssueGraph CLI

Native Go command-line application. Module path:
`github.com/borg0ai/dx-issuegraph/apps/cli`.

Build from repository root with `make build`, or run `go build` from this
directory. Release automation creates platform archives from Git tags and
publishes them as GitHub Release assets. macOS Homebrew distribution uses the
separate `systembugtj/luban-homebrew` tap after its formula PR merges.

The CLI uses embedded Dolt by default and does not require Docker to run.
