# issuegraph contract corpus — producer catalog

This package is the **producer** half of the IssueGraph ↔ consumer cross-version
contract-test system. It pins the `issuegraph` `--json` wire surface that a downstream
consumer's decoder consumes, by generating a canonicalized golden-JSON **corpus**
under `testdata/corpus/` and failing CI on any unreviewed change to it.

A downstream consumer vendors this corpus and replays it against its own decoder,
so a `issuegraph` release can never silently change a wire shape the consumer parses.

## How it works

- `corpus.go` — the deterministic command **plan**, the **canonicalizer**
  (timestamps → `<TS>`, object arrays stable-sorted, keys sorted, 2-space
  indent), and the provenance **manifest**.
- `corpus_test.go` — `TestCorpusGolden` runs the plan against a live `issuegraph`
  (built from source, Dolt-backed) in both flat and `BD_JSON_ENVELOPE=1`
  variants, canonicalizes, and **byte-compares** against the committed corpus.
  A diff is a hard failure (`make corpus-regen` to update); a Dolt-boot failure
  is an infra skip, never a silent pass.
- `canonicalize_test.go` — unit tests for the canonicalizer plus
  `TestCorpusDoubleRunByteIdentical`, the determinism backstop that generates
  the corpus twice and asserts byte-identity.

Regenerate after any deliberate wire change: `make corpus-regen`, review the
diff, **and bump `JSONSchemaVersion` (`modules/cli/output.go`)** — the `schema_version`
field in every blob is the coordination canary a downstream consumer keys its
pinned-decoder migration off.

## Coverage

| Blob (flat + envelope) | issuegraph command | Contract domain |
| ---------------------- | ---------- | --------------- |
| `create_root`, `create_dep`, `create_closed`, `create_deleted` | `issuegraph create … --json` | json-output-shapes (object) |
| `show` | `issuegraph show <id> --json` | json-output-shapes (array-of-one), dependency shape |
| `update` | `issuegraph update … --json` | json-output-shapes (mutation array), label add + metadata coercion (`phase` → string) |
| `close` | `issuegraph close --reason … --json` | close semantics (`close_reason`, `closed_at`) |
| `reopen` | `issuegraph reopen … --json` | reopen semantics (status back to open) |
| `list` | `issuegraph list --all --json` | json-output-shapes (array / list envelope) |
| `ready` | `issuegraph ready --json` | ready-projection-semantics |
| `dep_add`, `dep_list` | `issuegraph dep add/list --json` | json-output-shapes (dependency) |
| `dep_remove` | `issuegraph dep remove … --json` | dependency-edge removal confirmation |
| `delete` | `issuegraph delete --force … --json` | delete confirmation shape |
| `count` | `issuegraph count --json` | json-output-shapes (scalar) |
| `version` | `issuegraph version --json` | version-compat, the `schema_version` canary |
| `error` | `issuegraph show <missing> --json` | exit-codes-and-errors (`{error, schema_version}`) |

## Known gaps (tracked, not silent)

- `issuegraph sql` is **not supported in embedded mode** (this harness's mode), so it is
  not in the corpus. A downstream consumer's ready-projection enrichment depends on `issuegraph sql`
  against a managed Dolt server; covering it needs a server-mode generation path.
- The `show` blob pins the **count-only** default payload (`comment_count`,
  `dependent_count`); the opt-in `--include-comments` / `--include-dependents`
  shapes are not in the corpus. They are covered at the CLI level by the
  preservation and round-trip tests, not byte-pinned for a consumer.
- The `error` blob pins the not-found envelope; other error classifiers
  (claim-conflict, silent-fallback auto-import, `issuegraph sql` unsupported) are
  plain-text on stderr and belong in a separate error-string fixture set.
