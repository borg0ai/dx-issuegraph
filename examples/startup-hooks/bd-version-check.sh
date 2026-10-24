#!/bin/bash
#
# bd-version-check.sh - Automatic issuegraph upgrade detection for AI agent sessions
#
# This script detects when issuegraph (issuegraph) has been upgraded and automatically shows
# what changed, helping AI agents adapt their workflows without manual intervention.
#
# FEATURES:
# - Detects issuegraph version changes by comparing to last-seen version
# - Shows 'issuegraph info --whats-new' output when upgrade detected
# - Auto-updates git hooks if outdated
# - Persists version in .issuegraph/metadata.json
# - Zero issuegraph code changes required - works today!
#
# INTEGRATION:
# Add this script to your AI environment's session startup:
#
# Claude Code:
#   Add to .claude/hooks/session-start (if supported)
#   Or manually source at beginning of work
#
# GitHub Copilot:
#   Add to your shell initialization (.bashrc, .zshrc)
#   Or manually run at session start
#
# Cursor:
#   Add to workspace settings or shell init
#
# Generic:
#   source /path/to/bd-version-check.sh
#
# USAGE:
#   # Option 1: Source it (preferred)
#   source examples/startup-hooks/bd-version-check.sh
#
#   # Option 2: Execute it
#   bash examples/startup-hooks/bd-version-check.sh
#
# REQUIREMENTS:
# - issuegraph (issuegraph) installed and in PATH
# - jq for JSON manipulation
# - .issuegraph directory exists in current project
#

# Exit early if not in a issuegraph project
if [ ! -d ".issuegraph" ]; then
  return 0 2>/dev/null || exit 0
fi

# Check if issuegraph is installed
if ! command -v issuegraph &> /dev/null; then
  return 0 2>/dev/null || exit 0
fi

# Check if jq is installed (required for JSON manipulation)
if ! command -v jq &> /dev/null; then
  echo "⚠️  bd-version-check: jq not found. Install jq to enable automatic upgrade detection."
  return 0 2>/dev/null || exit 0
fi

# Get current issuegraph version
CURRENT_VERSION=$(issuegraph --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)

if [ -z "$CURRENT_VERSION" ]; then
  # issuegraph command failed, skip
  return 0 2>/dev/null || exit 0
fi

# Path to metadata file
METADATA_FILE=".issuegraph/metadata.json"

# Initialize metadata.json if it doesn't exist
if [ ! -f "$METADATA_FILE" ]; then
  echo '{"database": "issuegraph.db", "jsonl_export": "issuegraph.jsonl"}' > "$METADATA_FILE"
fi

# Read last-seen version from metadata.json
LAST_VERSION=$(jq -r '.last_issuegraph_version // "unknown"' "$METADATA_FILE" 2>/dev/null)

# Detect version change
if [ "$CURRENT_VERSION" != "$LAST_VERSION" ] && [ "$LAST_VERSION" != "unknown" ]; then
  echo ""
  echo "🔄 issuegraph upgraded: $LAST_VERSION → $CURRENT_VERSION"
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

  # Show what's new
  issuegraph info --whats-new 2>/dev/null || echo "⚠️  Could not fetch what's new (run 'issuegraph info --whats-new' manually)"

  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo "💡 Review changes above and adapt your workflow accordingly"
  echo ""
fi

# Check for outdated git hooks (works even if version didn't change)
if issuegraph hooks list 2>&1 | grep -q "outdated"; then
  echo "🔧 Git hooks outdated. Updating to match issuegraph v$CURRENT_VERSION..."
  if issuegraph hooks install 2>/dev/null; then
    echo "✓ Git hooks updated successfully"
  else
    echo "⚠️  Failed to update git hooks. Run 'issuegraph hooks install' manually."
  fi
  echo ""
fi

# Update metadata.json with current version
# Use a temp file to avoid corruption if jq fails
TEMP_FILE=$(mktemp)
if jq --arg v "$CURRENT_VERSION" '.last_issuegraph_version = $v' "$METADATA_FILE" > "$TEMP_FILE" 2>/dev/null; then
  mv "$TEMP_FILE" "$METADATA_FILE"
else
  # jq failed, clean up temp file
  rm -f "$TEMP_FILE"
fi

# Clean exit for sourcing
return 0 2>/dev/null || exit 0
