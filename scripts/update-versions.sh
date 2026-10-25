#!/bin/bash
set -e

# =============================================================================
# Quick version bump utility (no git operations)
# =============================================================================
#
# Updates version numbers across all issuegraph components without any git
# operations. Use this for local testing or when you want manual control
# over commits.
#
# Releases run through the shipify skill.
#
# =============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

usage() {
    echo "Usage: $0 <version>"
    echo ""
    echo "Updates version numbers across all components (no git operations)."
    echo ""
    echo "Examples:"
    echo "  $0 0.47.1"
    echo "  $0 1.1.0-rc.1"
    echo ""
    echo "Releases run through the shipify skill."
}

NEW_VERSION=""
for arg in "$@"; do
    case "$arg" in
        -h|--help) usage; exit 0 ;;
        -*) echo "Unknown option: $arg" >&2; usage; exit 1 ;;
        *)
            if [ -n "$NEW_VERSION" ]; then
                echo "Error: multiple versions given" >&2; usage; exit 1
            fi
            NEW_VERSION="$arg"
            ;;
    esac
done

if [ -z "$NEW_VERSION" ]; then
    usage
    exit 1
fi

# Validate semantic versioning. Accept prerelease identifiers so release
# candidates can be cut without pretending to be stable package releases.
if ! [[ $NEW_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
    echo -e "${RED}Error: Invalid version format '$NEW_VERSION'${NC}"
    echo "Expected: MAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH-prerelease (e.g., 0.47.1 or 1.1.0-rc.1)"
    exit 1
fi

# Check we're in repo root
if [ ! -f "apps/cli/version.go" ]; then
    echo -e "${RED}Error: Must run from repository root${NC}"
    exit 1
fi

# Get current version
CURRENT_VERSION=$(grep 'Version = ' apps/cli/version.go | sed 's/.*"\(.*\)".*/\1/')
# Base (prerelease-stripped) form of the current version. The Windows PE
# numeric fields (file_version/product_version, manifest version) only ever
echo -e "${YELLOW}Bumping: $CURRENT_VERSION → $NEW_VERSION${NC}"
echo ""

# Cross-platform sed helper
update_file() {
    local file=$1
    local old=$2
    local new=$3
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s|$old|$new|g" "$file"
    else
        sed -i "s|$old|$new|g" "$file"
    fi
}

echo "Updating version files..."

# 1. apps/cli/version.go
echo "  • apps/cli/version.go"
update_file "apps/cli/version.go" "Version = \"$CURRENT_VERSION\"" "Version = \"$NEW_VERSION\""

# 2. Plugin JSON files
echo "  • plugin metadata"
update_file "issuegraph/.claude-plugin/plugin.json" "\"version\": \"$CURRENT_VERSION\"" "\"version\": \"$NEW_VERSION\""
update_file "issuegraph/.codex-plugin/plugin.json" "\"version\": \"$CURRENT_VERSION\"" "\"version\": \"$NEW_VERSION\""
update_file "issuegraph/.copilot-plugin/plugin.json" "\"version\": \"$CURRENT_VERSION\"" "\"version\": \"$NEW_VERSION\""
update_file ".claude-plugin/marketplace.json" "\"version\": \"$CURRENT_VERSION\"" "\"version\": \"$NEW_VERSION\""
update_file ".codex-plugin/plugin.json" "\"version\": \"$CURRENT_VERSION\"" "\"version\": \"$NEW_VERSION\""

# 3. MCP Python package
echo "  • integrations/issuegraph-mcp/*"
update_file "integrations/issuegraph-mcp/pyproject.toml" "version = \"$CURRENT_VERSION\"" "version = \"$NEW_VERSION\""
update_file "integrations/issuegraph-mcp/src/issuegraph_mcp/__init__.py" "__version__ = \"$CURRENT_VERSION\"" "__version__ = \"$NEW_VERSION\""
# The release workflow's MCP package gate runs `uv sync --locked`, so a
# pyproject bump without a lock refresh fails the release only in the
# tag-triggered run — after the tag exists and can no longer be rewritten.
# That cost v1.1.0 and v1.1.2 their first release runs and burned the v1.1.1
# tag outright. Regenerate the lock as part of the bump. A failure here must
# not abort the bump half-applied (set -e): warn and let check-versions.sh
# hold the gate.
if command -v uv >/dev/null 2>&1; then
    echo "  • integrations/issuegraph-mcp/uv.lock"
    if ! uv lock --directory integrations/issuegraph-mcp; then
        echo -e "${RED}✗ uv lock failed — integrations/issuegraph-mcp/uv.lock NOT refreshed.${NC}"
        echo "  Fix and rerun before tagging, or check-versions.sh and the release MCP gate will fail:"
        echo "    uv lock --directory integrations/issuegraph-mcp"
    fi
else
    echo -e "${RED}✗ uv not found — integrations/issuegraph-mcp/uv.lock NOT refreshed.${NC}"
    echo "  Run this before tagging, or check-versions.sh and the release MCP gate will fail:"
    echo "    uv lock --directory integrations/issuegraph-mcp"
fi

# 5. README badge
echo "  • README.md"
update_file "README.md" "Alpha (v$CURRENT_VERSION)" "Alpha (v$NEW_VERSION)"

# 7. Tracked managed git-hook sections. The hooks a fresh `issuegraph init` installs
# are generated dynamically by apps/cli/hooks.go, but this repo also TRACKS
# rendered copies in .githooks/, whose BEGIN/END markers embed the binary
# Version (hookSectionBeginLine), and TestTrackedManagedHookSectionsMatchGenerator
# holds them byte-equal to the generator's output. Rewrite the marker version
# wholesale rather than from CURRENT_VERSION — the v1.2.0 bump proved the
# markers can already be drifted when the bump runs (they still said v1.1.0).
echo "  • .githooks/* managed section markers"
for hook in .githooks/*; do
    [ -f "$hook" ] || continue
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' -E "s#(--- (BEGIN|END) ISSUEGRAPH INTEGRATION) v[^ ]+ ---#\1 v$NEW_VERSION ---#g" "$hook"
    else
        sed -i -E "s#(--- (BEGIN|END) ISSUEGRAPH INTEGRATION) v[^ ]+ ---#\1 v$NEW_VERSION ---#g" "$hook"
    fi
done

echo ""
echo -e "${GREEN}✓ Version constants updated to $NEW_VERSION${NC}"
echo ""
echo "Changed files:"
git diff --stat 2>/dev/null || true
echo ""
echo "Next steps:"
echo "  • Update CHANGELOG.md with release notes"
echo "  • Update apps/cli/info.go versionChanges"
echo "  • Releases run through the shipify skill"
