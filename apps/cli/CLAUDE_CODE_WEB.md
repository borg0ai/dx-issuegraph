# Using issuegraph in Claude Code for Web

This guide shows how to automatically install and use issuegraph (IssueGraph issue tracker) in Claude Code for Web sessions using SessionStart hooks.

## What is Claude Code for Web?

Claude Code for Web provides full Linux VM sandboxes with npm support. Each session is a fresh environment, so tools need to be installed at the start of each session.

## Why npm Package Instead of Direct Binary?

Claude Code for Web environments:
- ✅ Have npm pre-installed and configured
- ✅ Can install global npm packages easily
- ❌ May have restrictions on direct binary downloads
- ❌ Don't persist installations between sessions

The `@beads/bd` npm package solves this by:
1. Installing via npm (which is always available)
2. Downloading the native binary during postinstall
3. Providing a CLI wrapper that "just works"

## Setup

### Option 1: SessionStart Hook (Recommended)

Create or edit `.claude/hooks/session-start.sh` in your project:

```bash
#!/bin/bash
# .claude/hooks/session-start.sh

# Install issuegraph globally (only takes a few seconds)
echo "Installing issuegraph (IssueGraph issue tracker)..."
npm install -g @beads/bd

# Initialize issuegraph in the project (if not already initialized)
if [ ! -d .beads ]; then
  issuegraph init --quiet
fi

echo "✓ issuegraph is ready! Use 'issuegraph ready' to see available work."
```

Make it executable:

```bash
chmod +x .claude/hooks/session-start.sh
```

### Option 2: Manual Installation Each Session

If you prefer not to use hooks, you can manually install at the start of each session:

```bash
npm install -g @beads/bd
issuegraph init --quiet
```

### Option 3: Project-Local Installation

Install as a dev dependency (slower but doesn't require global install):

```bash
npm install --save-dev @beads/bd

# Use with npx
npx bd version
npx bd ready
```

## Verification

After installation, verify issuegraph is working:

```bash
# Check version
issuegraph version

# Check database info
issuegraph info

# See what work is ready
issuegraph ready --json
```

## Usage in Claude Code for Web

Once installed, issuegraph works identically to the native version:

```bash
# Create issues
issuegraph create "Fix authentication bug" -t bug -p 1

# View ready work
issuegraph ready

# Start work
issuegraph update bd-a1b2 --claim

# Add dependencies
issuegraph dep add bd-f14c bd-a1b2

# Close issues
issuegraph close bd-a1b2 --reason "Fixed"
```

## Agent Integration

Tell your agent to use issuegraph by adding to your AGENTS.md or project instructions:

```markdown
## Issue Tracking

Use the `issuegraph` command for all issue tracking instead of markdown TODOs:

- Create issues: `issuegraph create "Task description" -p 1 --json`
- Find work: `issuegraph ready --json`
- Start work: `issuegraph update <id> --claim --json`
- View details: `issuegraph show <id> --json`

Use `--json` flags for programmatic parsing.
```

## How It Works

1. **SessionStart Hook**: Runs automatically when session starts
2. **npm install**: Downloads the @beads/bd package from npm registry
3. **postinstall**: Package automatically downloads the native binary for your platform
4. **CLI Wrapper**: `issuegraph` command is a Node.js wrapper that invokes the native binary
5. **issuegraph init**: Sets up the .beads directory and imports existing issues from git

## Performance

- **First install**: ~5-10 seconds (one-time per session)
- **Binary download**: ~3-5 seconds (darwin-arm64 binary is ~17MB)
- **Subsequent commands**: Native speed (<100ms)

## Troubleshooting

### "bd: command not found"

The SessionStart hook didn't run or installation failed. Manually run:

```bash
npm install -g @beads/bd
```

### npm postinstall fails with DNS or 403 errors

Some Claude Code web environments have network restrictions that prevent the npm postinstall script from downloading the binary. You'll see errors like:

```
Error installing issuegraph: getaddrinfo EAI_AGAIN github.com
```

or

```
curl: (22) The requested URL returned error: 403
```

**Workaround: Use a supported go install mode**

If Go is available (it usually is in Claude Code web), use the embedded-capable `go install` fallback:

```bash
# Install via Go without ICU
CGO_ENABLED=1 GOFLAGS=-tags=gms_pure_go go install github.com/steveyegge/beads/modules/cli@latest

# Add to PATH (required each session)
export PATH="$PATH:$HOME/go/bin"

# Verify installation
issuegraph version
```

**SessionStart hook with go install fallback:**

```bash
#!/bin/bash
# .claude/hooks/session-start.sh

echo "🔗 Setting up issuegraph (IssueGraph issue tracker)..."

# Try npm first, fall back to a supported Go install
if ! command -v bd &> /dev/null; then
    if npm install -g @beads/bd --quiet 2>/dev/null && command -v bd &> /dev/null; then
        echo "✓ Installed via npm"
    elif command -v go &> /dev/null; then
        echo "npm install failed, trying go install..."
        CGO_ENABLED=1 GOFLAGS=-tags=gms_pure_go go install github.com/steveyegge/beads/modules/cli@latest
        export PATH="$PATH:$HOME/go/bin"
        echo "✓ Installed via go install"
    else
        echo "✗ Installation failed - neither npm nor go available"
        exit 1
    fi
fi

# Verify and show version
issuegraph version
```

### "Error installing issuegraph: HTTP 404"

The version in package.json doesn't match a GitHub release. This shouldn't happen with published npm packages, but if it does, check:

```bash
npm view @beads/bd versions
```

And install a specific version:

```bash
npm install -g @beads/bd@0.21.5
```

### "Binary not found after extraction"

Platform detection may have failed. Check:

```bash
node -e "console.log(require('os').platform(), require('os').arch())"
```

Should output something like: `linux x64`

### Slow installation

The binary download may be slow depending on network conditions. The native binary is ~17MB, which should download in a few seconds on most connections.

If it's consistently slow, consider:
1. Using a different npm registry mirror
2. Caching the installation (if Claude Code for Web supports it)

## Benefits Over WASM

This npm package wraps the **native** issuegraph binary rather than using WebAssembly because:

- ✅ **Full SQLite support**: No custom VFS or compatibility issues
- ✅ **All features work**: 100% feature parity with standalone issuegraph
- ✅ **Better performance**: Native speed vs WASM overhead
- ✅ **Simpler maintenance**: Single binary build, no WASM-specific code
- ✅ **Faster installation**: One binary download vs WASM compilation

## Examples

### Example SessionStart Hook with Error Handling

```bash
#!/bin/bash
# .claude/hooks/session-start.sh

set -e  # Exit on error

echo "🔗 Setting up issuegraph (IssueGraph issue tracker)..."

# Install issuegraph globally
if ! command -v bd &> /dev/null; then
    echo "Installing @beads/bd from npm..."
    npm install -g @beads/bd --quiet
else
    echo "issuegraph already installed"
fi

# Verify installation
if bd version &> /dev/null; then
    echo "✓ issuegraph $(bd version)"
else
    echo "✗ issuegraph installation failed"
    exit 1
fi

# Initialize if needed
if [ ! -d .beads ]; then
    echo "Initializing issuegraph in project..."
    issuegraph init --quiet
else
    echo "issuegraph already initialized"
fi

# Show ready work
echo ""
echo "Ready work:"
issuegraph ready --limit 5

echo ""
echo "✓ issuegraph is ready! Use 'issuegraph --help' for commands."
```

### Example Claude Code Prompt

```
You are working on a project that uses issuegraph (IssueGraph) for issue tracking.

At the start of each session:
1. Run `issuegraph ready --json` to see available work
2. Choose an issue to work on
3. Start work: `issuegraph update <id> --claim`

While working:
- Create new issues for any bugs you discover
- Link related issues with `issuegraph dep add`
- Add comments with `issuegraph comments add <id> "comment text"`

When done:
- Close the issue: `issuegraph close <id> --reason "Description of what was done"`
- Run `issuegraph dolt push` to push issue changes
```

## Alternative: Package as Project Dependency

If you prefer to track issuegraph as a project dependency instead of global install:

```json
{
  "devDependencies": {
    "@beads/bd": "^0.21.5"
  },
  "scripts": {
    "bd": "bd",
    "ready": "bd ready",
    "postinstall": "bd init --quiet || true"
  }
}
```

Then use with npm scripts or npx:

```bash
npm run ready
npx bd create "New issue"
```

## Resources

- [issuegraph GitHub repository](https://github.com/gastownhall/beads)
- [npm package page](https://www.npmjs.com/package/@beads/bd)
- [Complete documentation](https://github.com/gastownhall/beads#readme)
- [Claude Code hooks documentation](https://docs.claude.com/claude-code)
