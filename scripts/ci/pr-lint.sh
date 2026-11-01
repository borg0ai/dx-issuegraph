#!/usr/bin/env bash
# Required PR formatting and Go lint contract.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# shellcheck source=../.buildflags
source "$REPO_ROOT/modules/core/.buildflags"
# shellcheck source=lib/timing.sh
source "$REPO_ROOT/scripts/ci/lib/timing.sh"

cd "$REPO_ROOT"

# The PR lane sets this to the branch it merges into so only the issues the PR
# introduces are reported; main's push lane leaves it unset and sweeps the
# whole tree. See the lint comment in .github/workflows/pr.yml.
lint_scope=()
if [[ -n "${ISSUEGRAPH_LINT_NEW_FROM_MERGE_BASE:-}" ]]; then
    lint_scope=(--new-from-merge-base="$ISSUEGRAPH_LINT_NEW_FROM_MERGE_BASE")
fi

ci_time "gofmt check" -- ./scripts/ci/fmt-check.sh

run_lint() {
    local module="$1"
    (
        cd "$REPO_ROOT/$module"
        ci_time "golangci-lint ($module)" -- \
            golangci-lint run --config="$REPO_ROOT/modules/core/.golangci.yml" --modules-download-mode=readonly \
                --timeout=5m --build-tags=gms_pure_go \
                ${lint_scope[@]+"${lint_scope[@]}"} ./...
    )
}

run_lint modules/core
run_lint apps/cli

# Other target tuples may not load files guarded by //go:build windows && !cgo.
# Cross-lint each module for that non-CGO Windows build.
native_goos="$(go env GOOS)"
native_cgo_enabled="$(go env CGO_ENABLED)"
if [[ "$native_goos" != "windows" || "$native_cgo_enabled" != "0" ]]; then
    for module in modules/core apps/cli; do
        (
            cd "$REPO_ROOT/$module"
            ci_time "golangci-lint (windows, $module)" -- \
                env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOWORK=off \
                    golangci-lint run --config="$REPO_ROOT/modules/core/.golangci.yml" --modules-download-mode=readonly \
                        --timeout=5m --build-tags=gms_pure_go \
                        ${lint_scope[@]+"${lint_scope[@]}"} ./...
        )
    done
fi
