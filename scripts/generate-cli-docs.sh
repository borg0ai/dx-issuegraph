#!/bin/bash
# generate-cli-docs.sh — Generate CLI reference docs from the live issuegraph command tree.
#
# Two stages:
#   1. `issuegraph help --docs-root <root>` emits vendor-neutral output only:
#      docs/CLI_REFERENCE.md plus a generic per-command tree at
#      build/cli-docs/ (uncommitted staging).
#   2. `go run ./tools/docsmint <root>` post-processes the staging tree into
#      the committed Mintlify pages at docs/cli-reference/ and splices the
#      CLI Reference pages array in docs/docs.json.
# issuegraph never emits site-generator-specific formats; all Mintlify targeting
# lives in tools/docsmint.

set -euo pipefail
export LC_ALL=C

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

CHECK_MODE=0
ISSUEGRAPH_ARG=""
TMP_BUILD_DIR=""
TMP_OUTPUT_DIR=""

while [ "$#" -gt 0 ]; do
    case "$1" in
        --check)
            CHECK_MODE=1
            shift
            ;;
        -h|--help)
            cat <<'EOF'
Usage: scripts/generate-cli-docs.sh [--check] [path-to-issuegraph]

Generate the committed CLI docs (docs/CLI_REFERENCE.md, docs/cli-reference/,
and the CLI pages array in docs/docs.json) from one issuegraph process plus the
tools/docsmint post-processor. --check regenerates into a scratch root and
fails if the committed copies are stale.

When docs/cli-docs.pin names a release tag, issuegraph is built from that tag
(the published docs describe the pinned release, not this checkout) and a
supplied binary is ignored; set ISSUEGRAPH_DOCS_IGNORE_PIN=1 to bypass the pin.

If the resolved issuegraph binary is CGO-enabled it emits the full `issuegraph federation`
help tree that CI (CGO_ENABLED=0) stubs out; the script rebuilds a pinned
pure-go binary to keep committed docs in sync. Set ISSUEGRAPH_DOCS_ALLOW_CGO=1 to
bypass that rebuild and trust the supplied binary as-is.
EOF
            exit 0
            ;;
        *)
            if [ -n "$ISSUEGRAPH_ARG" ]; then
                echo "Error: multiple issuegraph binaries supplied: $ISSUEGRAPH_ARG and $1" >&2
                exit 2
            fi
            ISSUEGRAPH_ARG="$1"
            shift
            ;;
    esac
done

cleanup() {
    if [ -n "$TMP_BUILD_DIR" ]; then
        rm -rf "$TMP_BUILD_DIR"
    fi
    if [ -n "$TMP_OUTPUT_DIR" ]; then
        rm -rf "$TMP_OUTPUT_DIR"
    fi
}
trap cleanup EXIT

# Release pin (docs/cli-docs.pin): the published docs describe the pinned
# release, so the pipeline builds issuegraph from that tag and ignores any supplied
# binary. Set ISSUEGRAPH_DOCS_IGNORE_PIN=1 to bypass (e.g. to preview docs for an
# unreleased emitter).
PINNED_BD=""
if [ "${ISSUEGRAPH_DOCS_IGNORE_PIN:-0}" != "1" ]; then
    PINNED_BD="$("$SCRIPT_DIR/resolve-docs-issuegraph.sh")"
fi

if [ -n "$PINNED_BD" ]; then
    if [ -n "$ISSUEGRAPH_ARG" ] && [ "$ISSUEGRAPH_ARG" != "$PINNED_BD" ]; then
        echo "Note: docs are pinned via docs/cli-docs.pin; ignoring supplied binary $ISSUEGRAPH_ARG." >&2
    fi
    ISSUEGRAPH="$PINNED_BD"
elif [ -n "$ISSUEGRAPH_ARG" ]; then
    ISSUEGRAPH="$ISSUEGRAPH_ARG"
elif [ "$CHECK_MODE" -eq 0 ] && [ -x "$PROJECT_ROOT/issuegraph" ]; then
    # Convenience for regeneration only. --check never trusts a repo-root
    # ./issuegraph it wasn't explicitly given: the guard below detects CGO-ness, not
    # staleness, so a stale pure-go ./issuegraph would produce a false "fresh".
    ISSUEGRAPH="$PROJECT_ROOT/issuegraph"
else
    TMP_BUILD_DIR="$(mktemp -d)"
    ISSUEGRAPH="$TMP_BUILD_DIR/issuegraph"
    echo "Building temporary issuegraph for docs generation..."
    (cd "$PROJECT_ROOT" && CGO_ENABLED=0 go build -tags gms_pure_go -o "$ISSUEGRAPH" github.com/borg0ai/dx-issuegraph/modules/cli)
fi

# Guard against a CGO-enabled issuegraph: it exposes the full `issuegraph federation` subcommand tree
# that CI never produces (scripts/ci/pr-policy.sh build_docs_binary uses env
# CGO_ENABLED=0 go build). A CGO build therefore emits ~hundreds of lines of federation
# help that the committed, CI-built docs stub out ("built without CGO support"), so a
# naive regen on a machine with a C compiler produces spurious federation churn.
#
# The pure-go federation stub prints "Federation commands require CGO" (see
# modules/cli/federation_nocgo.go); a CGO build does not. If the resolved binary is missing
# that stub marker, warn and rebuild a pinned CGO_ENABLED=0 -tags gms_pure_go binary so
# the committed docs always match CI. Set ISSUEGRAPH_DOCS_ALLOW_CGO=1 to bypass the rebuild and
# trust the supplied binary as-is (e.g. to deliberately regenerate the full federation
# tree); you then own any federation churn the diff introduces.
if [ -z "$PINNED_BD" ] && [ -x "$ISSUEGRAPH" ] && ! "$ISSUEGRAPH" federation --help 2>&1 | grep -q "Federation commands require CGO"; then
    if [ "${ISSUEGRAPH_DOCS_ALLOW_CGO:-0}" = "1" ]; then
        echo "WARNING: $ISSUEGRAPH looks CGO-enabled and emits the full 'issuegraph federation' help tree," >&2
        echo "         which CI's pure-go docs build stubs out. ISSUEGRAPH_DOCS_ALLOW_CGO=1 is set," >&2
        echo "         using it anyway; expect federation doc churn unless that is intended." >&2
    else
        echo "WARNING: $ISSUEGRAPH looks CGO-enabled; its 'issuegraph federation' help differs from CI's" >&2
        echo "         pure-go build and would add spurious federation doc churn." >&2
        echo "         Rebuilding a pinned pure-go (CGO_ENABLED=0 -tags gms_pure_go) binary" >&2
        echo "         for CI-consistent docs. Set ISSUEGRAPH_DOCS_ALLOW_CGO=1 to use it as-is." >&2
        if [ -z "$TMP_BUILD_DIR" ]; then
            TMP_BUILD_DIR="$(mktemp -d)"
        fi
        ISSUEGRAPH="$TMP_BUILD_DIR/bd-pure"
        (cd "$PROJECT_ROOT" && CGO_ENABLED=0 go build -tags gms_pure_go -o "$ISSUEGRAPH" github.com/borg0ai/dx-issuegraph/modules/cli)
    fi
fi

if [ ! -x "$ISSUEGRAPH" ]; then
    echo "Error: issuegraph binary not found or not executable: $ISSUEGRAPH" >&2
    echo "Usage: $0 [--check] [path-to-issuegraph]" >&2
    exit 1
fi

generate_all() {
    local root="$1"
    local had_website=0
    [ -d "$root/website" ] && had_website=1
    # Clear stale staging so docsmint can only consume what this issuegraph emitted
    # (a pinned legacy issuegraph stages under website/, not build/cli-docs).
    rm -rf "$root/build/cli-docs"
    "$ISSUEGRAPH" help --docs-root "$root"
    (cd "$PROJECT_ROOT" && go run -tags=gms_pure_go ./tools/docsmint "$root")
    # A pinned legacy emitter (issuegraph <= v1.1.0) stages Docusaurus pages under
    # website/; docsmint has consumed them, so drop the scaffolding.
    if [ "$had_website" -eq 0 ] && [ -d "$root/website" ]; then
        rm -rf "$root/website"
    fi
}

if [ "$CHECK_MODE" -eq 1 ]; then
    TMP_OUTPUT_DIR="$(mktemp -d)"
    mkdir -p "$TMP_OUTPUT_DIR/docs"
    # Seed the committed artifacts the pipeline rewrites, then regenerate and diff.
    cp -f "$PROJECT_ROOT/docs/docs.json" "$TMP_OUTPUT_DIR/docs/docs.json"
    if [ -d "$PROJECT_ROOT/docs/cli-reference" ]; then
        cp -Rf "$PROJECT_ROOT/docs/cli-reference" "$TMP_OUTPUT_DIR/docs/cli-reference"
    fi

    generate_all "$TMP_OUTPUT_DIR"

    if ! diff -q \
        "$PROJECT_ROOT/docs/CLI_REFERENCE.md" \
        "$TMP_OUTPUT_DIR/docs/CLI_REFERENCE.md" >/dev/null; then
        echo "FAIL: docs/CLI_REFERENCE.md is out of sync with live CLI help."
        echo "Run: ./scripts/generate-cli-docs.sh ${ISSUEGRAPH_ARG:-}"
        diff -u "$PROJECT_ROOT/docs/CLI_REFERENCE.md" "$TMP_OUTPUT_DIR/docs/CLI_REFERENCE.md" | sed -n '1,120p' || true
        exit 1
    fi

    if ! diff -q "$PROJECT_ROOT/docs/docs.json" "$TMP_OUTPUT_DIR/docs/docs.json" >/dev/null; then
        echo "FAIL: docs/docs.json CLI Reference navigation is out of sync with live CLI help."
        echo "Run: ./scripts/generate-cli-docs.sh ${ISSUEGRAPH_ARG:-}"
        diff -u "$PROJECT_ROOT/docs/docs.json" "$TMP_OUTPUT_DIR/docs/docs.json" | sed -n '1,80p' || true
        exit 1
    fi

    if ! diff -qr "$PROJECT_ROOT/docs/cli-reference" "$TMP_OUTPUT_DIR/docs/cli-reference" >/dev/null; then
        echo "FAIL: docs/cli-reference is out of sync with live CLI help."
        echo "Run: ./scripts/generate-cli-docs.sh ${ISSUEGRAPH_ARG:-}"
        diff -ur "$PROJECT_ROOT/docs/cli-reference" "$TMP_OUTPUT_DIR/docs/cli-reference" | sed -n '1,160p' || true
        exit 1
    fi

    echo "PASS: generated CLI docs are fresh"
else
    generate_all "$PROJECT_ROOT"
    echo "Generated CLI docs from: $($ISSUEGRAPH version 2>/dev/null | head -1 || echo "$ISSUEGRAPH")"
    echo "Updated docs/CLI_REFERENCE.md, docs/cli-reference/, and the docs.json CLI nav"
fi
