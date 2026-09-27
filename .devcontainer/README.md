# issuegraph Development Container

This devcontainer configuration provides a fully-configured development environment for issuegraph with:

- Go 1.23 development environment
- issuegraph CLI built and installed from source
- Git hooks automatically installed
- All dependencies pre-installed

## Quick Start

### GitHub Codespaces

1. Click the "Code" button on GitHub
2. Select "Create codespace on main"
3. Wait for the container to build (~2-3 minutes)
4. The environment will be ready with issuegraph installed and configured

### VS Code Remote Containers

1. Install the [Remote - Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) extension
2. Open the issuegraph repository in VS Code
3. Click "Reopen in Container" when prompted (or use Command Palette: "Remote-Containers: Reopen in Container")
4. Wait for the container to build

## What Gets Installed

The `setup.sh` script automatically:

1. Builds issuegraph from source (`go build ./modules/cli`)
2. Installs issuegraph to `/usr/local/bin/bd`
3. Runs `issuegraph init --quiet` (non-interactive initialization)
4. Installs git hooks from `examples/git-hooks/`
5. Downloads Go module dependencies

## Verification

After the container starts, verify everything works:

```bash
# Check bd is installed
issuegraph --version

# Check for ready tasks
issuegraph ready

# View project stats
issuegraph stats
```

## Git Configuration

Your local `.gitconfig` is mounted into the container so your git identity is preserved. If you need to configure git:

```bash
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

## Troubleshooting

**bd command not found:**
- The setup script should install issuegraph automatically
- Manually run: `bash .devcontainer/setup.sh`

**Git hooks not working:**
- Check if hooks are installed: `ls -la .git/hooks/`
- Manually install: `bash examples/git-hooks/install.sh`

**Container fails to build:**
- Check the container logs for specific errors
- Ensure Docker/Podman is running and has sufficient resources
- Try rebuilding: Command Palette → "Remote-Containers: Rebuild Container"

## Related Issues

- GitHub Issue [#229](https://github.com/gastownhall/beads/issues/229): Git hooks not available in devcontainers
- bd-ry1u: Publish official devcontainer configuration
