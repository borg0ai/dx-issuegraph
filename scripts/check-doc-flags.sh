#!/bin/bash
# check-doc-flags.sh — Validate documentation references against actual CLI flags.
#
# This script catches stale doc references by:
# 1. Extracting all flags from `issuegraph help --all`
# 2. Scanning docs for `issuegraph <command> --<flag>` patterns
# 3. Flagging any that don't exist in the CLI
#
# Also checks for references to known-removed commands.
#
# Usage: ./scripts/check-doc-flags.sh [bd-binary]
#
# Exit codes:
#   0 - All docs are consistent with CLI
#   1 - Stale references found

set -euo pipefail

ISSUEGRAPH="${1:-issuegraph}"
ERRORS=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Release pin (docs/cli-docs.pin): the docs describe the pinned release, so
# every check below validates against a issuegraph built from that tag, not the
# supplied binary. Set ISSUEGRAPH_DOCS_IGNORE_PIN=1 to bypass.
if [ "${ISSUEGRAPH_DOCS_IGNORE_PIN:-0}" != "1" ]; then
    PINNED_BD="$("$SCRIPT_DIR/resolve-docs-issuegraph.sh")"
    if [ -n "$PINNED_BD" ]; then
        if [ "$ISSUEGRAPH" != "$PINNED_BD" ]; then
            echo "Note: docs are pinned via docs/cli-docs.pin; validating against the pinned issuegraph."
        fi
        ISSUEGRAPH="$PINNED_BD"
    fi
fi

# Verify issuegraph binary exists and runs
if ! command -v "$ISSUEGRAPH" &>/dev/null && [ ! -x "$ISSUEGRAPH" ]; then
    echo "Error: issuegraph binary not found at '$ISSUEGRAPH'"
    echo "Usage: $0 [path-to-issuegraph]"
    exit 1
fi

# macOS has no coreutils timeout(1) by default; degrade to an unbounded run.
run_bounded() {
    if command -v timeout >/dev/null 2>&1; then
        timeout 30 "$@"
    else
        "$@"
    fi
}

echo "Checking documentation against CLI flags..."
echo "Using: $($ISSUEGRAPH version 2>/dev/null | head -1 || echo "$ISSUEGRAPH")"
echo ""

# --- Check 1: Known-removed commands ---
echo "=== Check 1: Removed commands ==="

# issuegraph sync: the JSONL-era command was removed in v0.51, and a top-level `issuegraph sync`
# was reintroduced (wy-jpd3.4) as the federation loop — pull, positive conflict
# check, recompute-blocked, push. So the stale reference to hunt is no longer
# any mention of the command; it is docs still describing it as removed,
# deprecated, or a no-op, which is exactly what its own command doc said until
# the verb came back. CHANGELOG and the staged-for-removal audits are history
# and legitimately say so.
SYNC_REFS=$(grep -rn 'issuegraph sync\b' \
    "$PROJECT_ROOT"/docs/*.md \
    "$PROJECT_ROOT"/docs/*/*.md \
    "$PROJECT_ROOT"/AGENT_INSTRUCTIONS.md \
    "$PROJECT_ROOT"/AGENTS.md \
    "$PROJECT_ROOT"/README.md \
    "$PROJECT_ROOT"/apps/cli/*.md \
    "$PROJECT_ROOT"/integrations/*/README.md \
    "$PROJECT_ROOT"/plugins/issuegraph/skills/issuegraph/commands/*.md \
    "$PROJECT_ROOT"/plugins/issuegraph/skills/issuegraph/resources/*.md \
    2>/dev/null \
    | grep -v 'CHANGELOG\|audit-sync-mode' \
    | grep -i 'deprecated\|no-op\|removed\|unknown-command' \
    | grep -vi 'JSONL-era' \
    || true)

if [ -n "$SYNC_REFS" ]; then
    echo "FAIL: Found docs describing 'issuegraph sync' as removed/deprecated — it is a live command:"
    echo "$SYNC_REFS" | head -20
    ERRORS=$((ERRORS + 1))
else
    echo "PASS: No stale 'issuegraph sync is removed' references"
fi

echo ""

# --- Check 2: issuegraph init flags ---
echo "=== Check 2: issuegraph init flags ==="

# Get actual init flags. Match both long-only rows and rows with a short alias.
INIT_FLAGS_OK=1
if ! INIT_FLAGS=$("$ISSUEGRAPH" init --help 2>&1 \
    | grep -E '^[[:space:]]+(-[[:alnum:]], )?--[a-z]' \
    | grep -oE -- '--[a-z][a-z0-9-]*' \
    | sort -u); then
    echo "FAIL: Could not extract flags from 'issuegraph init --help'"
    ERRORS=$((ERRORS + 1))
    INIT_FLAGS_OK=0
fi

# Check for --branch on init (removed)
BRANCH_REFS=$(grep -rn 'issuegraph init.*--branch' \
    "$PROJECT_ROOT"/docs/*.md \
    "$PROJECT_ROOT"/docs/*/*.md \
    "$PROJECT_ROOT"/AGENT_INSTRUCTIONS.md \
    "$PROJECT_ROOT"/AGENTS.md \
    "$PROJECT_ROOT"/README.md \
    2>/dev/null \
    | grep -v 'CHANGELOG\|removed\|was removed\|no longer\|deprecated' \
    || true)

if [ "$INIT_FLAGS_OK" -ne 1 ]; then
    echo "SKIP: Cannot validate 'issuegraph init --branch' without the live init flag list"
elif printf '%s\n' "$INIT_FLAGS" | grep -qx -- '--branch'; then
    echo "PASS: 'issuegraph init --branch' is a live flag"
elif [ -n "$BRANCH_REFS" ]; then
    echo "FAIL: Found references to removed 'issuegraph init --branch' flag:"
    echo "$BRANCH_REFS" | head -20
    ERRORS=$((ERRORS + 1))
else
    echo "PASS: No stale 'issuegraph init --branch' references"
fi

echo ""

# --- Check 3: SQLite/legacy database paths ---
echo "=== Check 3: Legacy storage references ==="

SQLITE_REFS=$(grep -rn 'issuegraph\.db\|default\.db\|sqlite3.*\.issuegraph\|\.issuegraph/.*\.db' \
    "$PROJECT_ROOT"/docs/*.md \
    "$PROJECT_ROOT"/docs/*/*.md \
    "$PROJECT_ROOT"/AGENT_INSTRUCTIONS.md \
    "$PROJECT_ROOT"/AGENTS.md \
    "$PROJECT_ROOT"/README.md \
    2>/dev/null \
    | grep -v 'CHANGELOG\|removed\|legacy\|migration\|migrate\|was removed\|pre-\|old\|deprecated' \
    || true)

if [ -n "$SQLITE_REFS" ]; then
    echo "WARN: Found possible legacy SQLite/database references:"
    echo "$SQLITE_REFS" | head -20
    # Don't increment ERRORS — these may be intentional migration docs
else
    echo "PASS: No stale SQLite references"
fi

echo ""

# --- Check 4: CLI command docs coverage and freshness ---
echo "=== Check 4: CLI command docs coverage and freshness ==="

CLI_REF="$PROJECT_ROOT/docs/CLI_REFERENCE.md"
if [ -f "$CLI_REF" ]; then
    TMPDIR_CHECK=$(mktemp -d)
    trap "rm -rf $TMPDIR_CHECK" EXIT
    if run_bounded "$ISSUEGRAPH" help --list > "$TMPDIR_CHECK/help-cmds.txt" 2>/dev/null; then
        sort -u "$TMPDIR_CHECK/help-cmds.txt" -o "$TMPDIR_CHECK/help-cmds.txt"

        grep -oE '\bbd [a-z][a-z0-9-]*\b' "$CLI_REF" \
            | sed 's/^issuegraph //' | sort -u > "$TMPDIR_CHECK/cli-reference-cmds.txt" || true

        MISSING_CLI_REF=$(comm -23 "$TMPDIR_CHECK/help-cmds.txt" "$TMPDIR_CHECK/cli-reference-cmds.txt" || true)
        if [ -n "$MISSING_CLI_REF" ]; then
            echo "FAIL: Live CLI commands missing from docs/CLI_REFERENCE.md:"
            echo "$MISSING_CLI_REF" | sed 's/^/  issuegraph /' | head -50
            ERRORS=$((ERRORS + 1))
        else
            echo "PASS: docs/CLI_REFERENCE.md covers all live top-level CLI commands"
        fi

        WEBSITE_DIRS=("$PROJECT_ROOT/docs/cli-reference")
        for dir in "${WEBSITE_DIRS[@]}"; do
            if [ ! -d "$dir" ]; then
                continue
            fi
            grep -rhoE '\bbd [a-z][a-z0-9-]*\b' "$dir"/*.md \
                | sed 's/^issuegraph //' | sort -u > "$TMPDIR_CHECK/website-cmds.txt" || true

            MISSING_WEBSITE=$(comm -23 "$TMPDIR_CHECK/help-cmds.txt" "$TMPDIR_CHECK/website-cmds.txt" || true)
            if [ -n "$MISSING_WEBSITE" ]; then
                echo "FAIL: Live CLI commands missing from ${dir#$PROJECT_ROOT/}:"
                echo "$MISSING_WEBSITE" | sed 's/^/  issuegraph /' | head -50
                ERRORS=$((ERRORS + 1))
            else
                echo "PASS: ${dir#$PROJECT_ROOT/} covers all live top-level CLI commands"
            fi
        done

        if [ -x "$PROJECT_ROOT/scripts/generate-cli-docs.sh" ]; then
            ISSUEGRAPH_FOR_GEN="$ISSUEGRAPH"
            if ! [ -x "$ISSUEGRAPH_FOR_GEN" ]; then
                ISSUEGRAPH_FOR_GEN="$(command -v "$ISSUEGRAPH" 2>/dev/null || true)"
            fi
            if [ -z "$ISSUEGRAPH_FOR_GEN" ]; then
                echo "FAIL: Could not resolve issuegraph binary path for CLI docs freshness check"
                ERRORS=$((ERRORS + 1))
            elif "$PROJECT_ROOT/scripts/check-cli-docs-drift.sh" "$ISSUEGRAPH_FOR_GEN"; then
                echo "PASS: CLI docs freshness gate (see drift check output above)"
            else
                ERRORS=$((ERRORS + 1))
            fi
        else
            echo "FAIL: scripts/generate-cli-docs.sh is missing"
            ERRORS=$((ERRORS + 1))
        fi
    else
        echo "FAIL: issuegraph help --list timed out or unavailable"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "FAIL: docs/CLI_REFERENCE.md not found"
    ERRORS=$((ERRORS + 1))
fi

echo ""

# --- Check 5: Plugin CLI reference must stay a pointer ---
echo "=== Check 5: Plugin CLI reference pointer policy ==="

PLUGIN_CLI_REF="$PROJECT_ROOT/plugins/issuegraph/skills/issuegraph/resources/CLI_REFERENCE.md"
if [ -f "$PLUGIN_CLI_REF" ]; then
    if grep -q 'This skill does not bundle a copied CLI command reference' "$PLUGIN_CLI_REF" \
        && grep -q 'docs/CLI_REFERENCE.md' "$PLUGIN_CLI_REF"; then
        echo "PASS: plugin CLI resource is a pointer to live/canonical references"
    else
        echo "FAIL: plugin CLI resource drifted from pointer policy"
        echo "  Expected pointer language and canonical docs/CLI_REFERENCE.md reference in:"
        echo "  plugins/issuegraph/skills/issuegraph/resources/CLI_REFERENCE.md"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "FAIL: plugin CLI resource missing: plugins/issuegraph/skills/issuegraph/resources/CLI_REFERENCE.md"
    ERRORS=$((ERRORS + 1))
fi

echo ""

# --- Summary ---
echo "=== Summary ==="
if [ $ERRORS -gt 0 ]; then
    echo "FAILED: $ERRORS stale reference category(ies) found"
    echo ""
    echo "To fix:"
    echo "  1. Update stale prose references to use current CLI commands."
    echo "  2. Regenerate CLI docs with: ./scripts/generate-cli-docs.sh $ISSUEGRAPH"
    echo "  3. Re-run: ./scripts/check-doc-flags.sh $ISSUEGRAPH"
    exit 1
else
    echo "PASSED: All documentation references are consistent with CLI"
    exit 0
fi
