# Publishing @borg0ai/issuegraph to npm

This guide covers how to publish the @borg0ai/issuegraph package to npm.

## Prerequisites

1. **npm account**: You need an npm account. Create one at https://www.npmjs.com/signup
2. **@issuegraph organization**: The package is scoped to `@issuegraph`, so you need to either:
   - Create the @issuegraph organization on npm (if it doesn't exist)
   - Be a member of the @issuegraph organization with publish rights

## Setup

### 1. Login to npm

```bash
npm login
```

This will prompt for:
- Username
- Password
- Email
- OTP (if 2FA is enabled)

### 2. Verify authentication

```bash
npm whoami
```

Should output your npm username.

### 3. Create organization (if needed)

If the `@issuegraph` organization doesn't exist:

```bash
npm org create issuegraph
```

Or manually at: https://www.npmjs.com/org/create

## Publishing

### 1. Update version (if needed)

The version in `package.json` should match the issuegraph release version.

```bash
# Edit package.json manually or use npm version
npm version patch  # or minor, major
```

### 2. Test the package

```bash
# From the apps/cli directory
npm test

# Test installation locally
npm link

# Verify the global install works
issuegraph version
```

### 3. Publish to npm

For the first publish:

```bash
# Publish as public (scoped packages are private by default)
npm publish --access public
```

For subsequent publishes:

```bash
npm publish
```

### 4. Verify publication

```bash
# Check the package page
open https://www.npmjs.com/package/@borg0ai/issuegraph

# Try installing it
npm install -g @borg0ai/issuegraph
issuegraph version
```

## Publishing Workflow

The typical workflow when a new issuegraph version is released:

1. **Wait for GitHub release**: Ensure the new version is released on GitHub with binaries
2. **Update package.json**: Update version to match the GitHub release
3. **Test locally**: Run `npm install` and `npm test` to ensure binaries download correctly
4. **Publish**: Run `npm publish --access public` (or just `npm publish` after first release)
5. **Verify**: Install globally and test

## Automation (Future)

In the future, this could be automated with GitHub Actions:

```yaml
# .github/workflows/publish-npm.yml
name: Publish to npm
on:
  release:
    types: [published]
jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
          registry-url: 'https://registry.npmjs.org'
      - run: cd apps/cli && npm publish --access public
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

## Troubleshooting

### "EEXIST: package already published"

You're trying to publish a version that already exists. Bump the version:

```bash
npm version patch
npm publish
```

### "ENEEDAUTH: need auth"

You're not logged in:

```bash
npm login
npm whoami  # verify
```

### "E403: forbidden"

You don't have permission to publish to @issuegraph. Either:
- Create the organization
- Ask the organization owner to add you
- Change the package name to something you own

### Binary download fails during postinstall

The version in package.json doesn't match a GitHub release, or the release doesn't have the required binary assets.

Check: https://github.com/borg0ai/dx-issuegraph/releases/v{VERSION}

## Version Sync

Keep these in sync:
- `apps/cli/package.json` version
- GitHub release tag (e.g., `v0.21.5`)
- IssueGraph binary version

The postinstall script downloads binaries from:
```
https://github.com/borg0ai/dx-issuegraph/releases/download/v{VERSION}/issuegraph_{VERSION}_{platform}_{arch}.{ext}
```
