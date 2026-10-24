#!/usr/bin/env bash
# MCP Python package gate.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MCP_DIR="$REPO_ROOT/integrations/issuegraph-mcp"

# shellcheck source=../../.buildflags
source "$REPO_ROOT/modules/core/.buildflags"
# shellcheck source=lib/timing.sh
source "$REPO_ROOT/scripts/ci/lib/timing.sh"

tmpdir=""
cleanup() {
    if [[ -n "$tmpdir" ]]; then
        rm -rf "$tmpdir"
    fi
}
trap cleanup EXIT

prepare_issuegraph_binary() {
    tmpdir="$(mktemp -d)"
    local target="$tmpdir/issuegraph"

    if [[ -n "${ISSUEGRAPH_TEST_ISSUEGRAPH_BINARY:-}" ]]; then
        cp "$ISSUEGRAPH_TEST_ISSUEGRAPH_BINARY" "$target"
    else
        go build -o "$target" github.com/borg0ai/dx-issuegraph/modules/cli
    fi

    chmod +x "$target"
    export PATH="$tmpdir:$PATH"
    issuegraph version
}

mcp_uv_sync() {
    cd "$MCP_DIR"
    uv sync --all-groups --locked
}

mcp_ruff() {
    cd "$MCP_DIR"
    uv run ruff check src/issuegraph_mcp tests
}

mcp_mypy() {
    cd "$MCP_DIR"
    uv run mypy src/issuegraph_mcp
}

mcp_pytest() {
    cd "$MCP_DIR"
    uv run pytest --durations=50
}

mcp_build() {
    cd "$MCP_DIR"
    rm -rf dist
    uv build
}

cd "$REPO_ROOT"

ci_time "prepare issuegraph for MCP package" -- prepare_issuegraph_binary
ci_time "mcp uv sync" -- mcp_uv_sync
ci_time "mcp ruff check" -- mcp_ruff
ci_time "mcp mypy" -- mcp_mypy
ci_time "mcp pytest" -- mcp_pytest
ci_time "mcp build" -- mcp_build
