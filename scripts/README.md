# IssueGraph Scripts

Utility scripts for maintaining the issuegraph project.

## ci/

Repository-owned CI command wrappers. These scripts are the source of truth for
the target CI tiers; Make targets are aliases for local discoverability.

```bash
make ci-pr-core
make ci-pr-policy
make ci-pr-lint
make ci-package-mcp
```

Each wrapper auto-detects the repository root, sources `.buildflags` when it
invokes Go in the default build mode, and records per-command timing through
`scripts/ci/lib/timing.sh`.

Broad Go test wrappers also source `scripts/ci/lib/test-env.sh`, which creates a
temporary HOME/XDG/Dolt root, isolates Git global/system config, clears runtime
IssueGraph/Dolt environment variables, and sets `ISSUEGRAPH_TEST_SKIP=dolt` before tests
run. This keeps local `make test` and `make ci-pr-core` results comparable to
the fast PR-core contract even on shared agent hosts. Set
`ISSUEGRAPH_TEST_ENV_RUN_DOLT=1` only when intentionally running the Dolt-dependent
tests through these broad wrappers, or `ISSUEGRAPH_TEST_ENV_DISABLE=1` when debugging
against your real local configuration.

The broad Go wrappers also cap package and test parallelism to `4` by default
(`GO_TEST_PKG_PARALLEL` and `GO_TEST_PARALLEL`). This avoids turning high-core
shared hosts into a different test topology than GitHub Actions.

`make ci-pr-policy` includes `scripts/check-testing-short.sh`, which enforces
that `testing.Short()` is only used for runtime, stress, or large-fixture skips.
Use build tags, environment checks, or named wrappers for integration/e2e/API
boundaries.

Package gate wrappers validate publishable/package-adjacent surfaces:

- `make ci-package-mcp` builds or consumes an `issuegraph` binary, puts it on `PATH` as
  `issuegraph`, then runs locked MCP package `uv sync`, Ruff, mypy, pytest, and build
  checks.
Set `ISSUEGRAPH_TEST_ISSUEGRAPH_BINARY=/path/to/issuegraph` for the MCP package gate to reuse a
prebuilt candidate binary instead of rebuilding it inside the wrapper.

## pr-preflight.sh

Read-only PR safety check for agents and maintainers.

```bash
# Before implementing or opening a related PR
./scripts/pr-preflight.sh --search "topic keywords" --repo borg0ai/dx-issuegraph

# Before changing, closing, or merging an existing PR
./scripts/pr-preflight.sh 123 --repo borg0ai/dx-issuegraph
```

It reports contributor/fork status, draft/review/merge/check state, risky diff
signals such as `.issuegraph/` changes or missing tests, and the required
contributor-protection next steps. It does not replace code review or local
validation.

Base-branch health is supplemented by a warn-only PR-gate sample: the last 60
completed `pull_request`-event runs, grouped per workflow. A single workflow
whose decisive runs (>= 5, across >= 3 distinct head branches) are all
failure-class is reported as a broken PR gate — the case where a job that
exists only in the PR workflow is red for every PR while the base branch shows
green. This detector never blocks (deliberately: automation classifies
unrecognized `[block]` lines as genuine merge blockers and would park merge
lanes on a false positive); it is skipped entirely while the base branch is
red.

## gh-body-lint

Lint Markdown files before posting them with `gh ... --body-file`.

```bash
./scripts/gh-body-lint body.md
./scripts/gh-body-lint --fix body.md
```

The lint catches literal `\n` sequences, which render poorly on GitHub, and
`GH#123` references, which do not auto-link like `#123` or `owner/repo#123`.

## update-versions.sh

Updates version numbers across all issuegraph components. It performs no git
operations; review the diff and commit yourself.

```bash
./scripts/update-versions.sh 0.9.3
./scripts/update-versions.sh 1.1.0-rc.1
```

Run `./scripts/check-versions.sh` afterwards to verify the version files agree.
Releases themselves run through the `shipify` skill.

---

## Future Scripts

Additional maintenance scripts may be added here as needed.
