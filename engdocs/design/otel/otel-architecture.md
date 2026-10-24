# OpenTelemetry Architecture

## Overview

IssueGraph uses OpenTelemetry (OTel) for structured observability of all database operations, CLI commands, and Dolt version control. Telemetry is emitted via standard OTLP HTTP to any compatible backend (metrics, traces).

**Backend-agnostic design**: The system emits standard OpenTelemetry Protocol (OTLP) — any OTLP v1.x+ compatible backend can consume it. You are **not obligated** to use VictoriaMetrics/VictoriaLogs; these are simply development defaults.

**Best-effort design**: Telemetry initialization errors are returned but do not affect normal `issuegraph` operation. The system remains functional even when telemetry is unavailable.

---

## Implementation Status

### Core Telemetry (Implemented ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Core OTel initialization | ✅ Implemented | `telemetry.Init()`, providers setup |
| Metrics export (counters) | ✅ Implemented | Storage operations, Dolt operations |
| Metrics export (histograms) | ✅ Implemented | Operation durations, query latency |
| Traces (stdout only) | ✅ Implemented | OTLP traces via stdout (dev mode) |
| Storage layer instrumentation | ✅ Implemented | `InstrumentedStorage` wrapper for all storage ops |
| Command lifecycle tracing | ✅ Implemented | Per-command spans with arguments |
| Dolt version control tracing | ✅ Implemented | Commit, push, pull, merge operations |

### Dolt Backend Telemetry (Implemented ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| SQL query tracing | ✅ Implemented | All Dolt queries wrapped with spans |
| Dolt lock wait timing | ✅ Defined | `issuegraph.db.lock_wait_ms` histogram registered; `.Record()` not yet called |
| Dolt retry counting | ✅ Implemented | `issuegraph.db.retry_count` counter |
| Dolt circuit breaker | ✅ Implemented | `issuegraph.db.circuit_trips`, `issuegraph.db.circuit_rejected` counters |
| Auto-commit tracking | ✅ Implemented | Per-command auto-commit events |
| Working set flush tracking | ✅ Implemented | Flush on shutdown/signal |

### Server Lifecycle Telemetry (Not yet instrumented ❌)

`internal/doltserver/` has no OTel imports. Server lifecycle spans and metrics are roadmap items (see Tier 1 below).

---

## Roadmap

Current coverage: ~40% of the codebase. Below is a prioritized plan based on operational value vs. implementation effort.

### Tier 1 — High value, moderate effort

#### Tracker integrations (`internal/linear/`, `internal/jira/`, `internal/gitlab/`)

External API calls are currently a black box. No visibility into latency, rate-limiting, or sync volume.

New metrics:
- `issuegraph_tracker_api_calls_total` (Counter) — by `tracker`, `method`, `status`
- `issuegraph_tracker_api_latency_ms` (Histogram) — by `tracker`, `method`
- `issuegraph_tracker_errors_total` (Counter) — by `tracker`, `error_type`
- `issuegraph_tracker_issues_synced_total` (Counter) — by `tracker`, `direction`

New spans: `tracker.<name>.pull_issues`, `tracker.<name>.push_issue`, `tracker.<name>.resolve_state`

#### Git operations (`internal/git/`)

Git push/pull can dominate wall-clock time but is currently invisible.

New metrics:
- `issuegraph_git_operation_duration_ms` (Histogram) — by `operation`, `status`
- `issuegraph_git_errors_total` (Counter) — by `operation`, `error_type`

New spans: `git.clone`, `git.pull`, `git.push`, `git.commit`, `git.merge`

#### Dolt server lifecycle (`internal/doltserver/`)

Server crashes and restarts are silent. No alerting possible.

New metrics:
- `issuegraph_doltserver_status` (Gauge, 1=running/0=stopped)
- `issuegraph_doltserver_startup_ms` (Histogram)
- `issuegraph_doltserver_restarts_total` (Counter)
- `issuegraph_doltserver_errors_total` (Counter) — by `error_type`

New spans: `doltserver.start`, `doltserver.stop`

---

### Tier 2 — Medium value, low effort

#### Query engine (`internal/query/`)

Distinguishes whether slowness is client-side (parsing/compilation) or DB-side.

New spans: `query.parse`, `query.compile`
New metrics: `issuegraph_query_duration_ms` (Histogram), `issuegraph_query_parse_errors_total` (Counter)

#### Validation engine (`internal/validation/`)

Data integrity errors are currently silent until they surface as user-visible failures.

New spans: `validation.check_dependencies`, `validation.check_schema`
New metrics: `issuegraph_validation_errors_total` (Counter) — by `error_type`

#### Dolt version control (`internal/storage/dolt/versioned.go`)

`versioned.go` has no OTel imports yet. Future spans:
- `History`: Query complete version history for an issue
- `AsOf`: Query state at specific commit or branch
- `Diff`: Cell-level diff between two commits
- `ListBranches`: Enumerate all branches
- `GetCurrentCommit`: Get HEAD commit hash
- `GetConflicts`: Check for merge conflicts

#### Dolt system table polling

Periodic SQL queries against Dolt system tables to surface metrics unavailable via OTLP (Dolt has no native OTel export):

| Metric | Source | Frequency |
|--------|--------|-----------|
| `issuegraph_dolt_commits_per_hour` | `dolt_log` GROUP BY hour | 5 min |
| `issuegraph_dolt_working_set_size` | `dolt_status` COUNT(*) | 1 min |
| `issuegraph_dolt_branch_count` | `dolt_branches` COUNT(*) | 5 min |
| `issuegraph_dolt_conflict_count` | `dolt_conflicts` COUNT(*) | 5 min |

---

### Tier 3 — Low priority / future

- **Command-level sub-spans**: Instrument validation vs. DB vs. render breakdown per command (`issuegraph create`, `issuegraph list`, `issuegraph compact`, etc.)
- **Molecules & recipes**: `molecule.create`, `recipe.execute` spans
- **Hook duration metrics**: Currently only spans (`hook.exec`), no histogram for aggregation
- **OTel test suite**: Integration tests that verify telemetry output (currently none)
- **Lock wait recording**: `issuegraph.db.lock_wait_ms` histogram is registered but `.Record()` is not yet called

---

## Components

### 1. Initialization (`internal/telemetry/telemetry.go`)

The `telemetry.Init()` function sets up OTel providers on process startup and returns only an `error`:

```go
if err := telemetry.Init(ctx, "issuegraph", version); err != nil {
    // Log and continue — telemetry is best-effort
}
defer telemetry.Shutdown(ctx)
```

**Providers:**
- **Metrics**: Any OTLP-compatible metrics backend via `otlpmetrichttp` exporter
- **Traces**: Stdout only (local debug). No remote trace backend in default stack.

**Default endpoints** (when `ISSUEGRAPH_OTEL_METRICS_URL` is not set):
- Metrics: `http://localhost:8428/opentelemetry/api/v1/push`
- Traces: stdout (via `ISSUEGRAPH_OTEL_STDOUT=true`)

> **Note**: These defaults target VictoriaMetrics for local development convenience. IssueGraph uses standard OTLP — you can override endpoints to use any OTLP v1.x+ compatible backend (Prometheus, Grafana Mimir, Datadog, New Relic, Grafana Cloud, Loki, OpenTelemetry Collector, etc.).

**OTLP Compatibility**:
- Uses standard OpenTelemetry Protocol (OTLP) over HTTP
- Protobuf encoding (VictoriaMetrics, Prometheus, and others accept this)
- Compatible with any backend that supports OTLP v1.x+

**Resource attributes** (set at init time):
- `service.name`: "issuegraph"
- `service.version`: issuegraph binary version
- `host`: system hostname
- `os`: system OS info

**Custom resource attributes** (via `OTEL_RESOURCE_ATTRIBUTES` env var or `ISSUEGRAPH_ACTOR`):
- `issuegraph.actor`: Actor identity (from git config or env) — set after actor resolution
- `issuegraph.command`: Current command name
- `issuegraph.args`: Full arguments passed to command

---

### 2. Storage Instrumentation (`internal/telemetry/storage.go`)

The `InstrumentedStorage` wraps `storage.Storage` with OTel tracing and metrics:
- Every storage method gets a span
- Counters track operation counts
- Histograms track operation duration
- Error counters track failures

```go
func WrapStorage(s storage.Storage) storage.Storage {
    if !Enabled() {
        return s  // Zero overhead when telemetry disabled
    }
    // Wrap with instrumentation
    return &InstrumentedStorage{inner: s, tracer, ops, dur, errs, issueGauge}
}
```

**Metric names in code** (OTel SDK notation with dots):
- `issuegraph.storage.operations` → exported as `issuegraph_storage_operations_total` by Prometheus/VM
- `issuegraph.storage.operation.duration` → `issuegraph_storage_operation_duration_ms`
- `issuegraph.storage.errors` → `issuegraph_storage_errors_total`
- `issuegraph.issue.count` → `issuegraph_issue_count`

**Instrumented Storage Operations:**
- Issue CRUD: `CreateIssue`, `GetIssue`, `UpdateIssue`, `CloseIssue`, `DeleteIssue`
- Dependencies: `AddDependency`, `RemoveDependency`, `GetDependencies`
- Labels: `AddLabel`, `RemoveLabel`, `GetLabels`
- Queries: `SearchIssues`, `GetReadyWork`, `GetBlockedIssues`
- Statistics: `GetStatistics` (also emits gauge of issue counts by status)
- Transactions: `RunInTransaction`

---

### 3. Dolt Backend Telemetry (`internal/storage/dolt/store.go`)

Dolt storage layer emits metrics for:
- `issuegraph.db.retry_count`: SQL retries in server mode (recorded in `withRetry` when `attempts > 1`)
- `issuegraph.db.lock_wait_ms`: Histogram registered but `.Record()` not yet called (stub)
- `issuegraph.db.circuit_trips`: Circuit breaker trips to open state (recorded in `withRetry`)
- `issuegraph.db.circuit_rejected`: Requests rejected by open circuit breaker (fail-fast path)
- SQL query spans via `queryContext()`, `execContext()`, `queryRowContext()` wrappers using `doltTracer`
- Dolt version control spans: `dolt.commit`, `dolt.push`, `dolt.pull`, `dolt.merge`, `dolt.branch`, `dolt.checkout`

**SQL Span pattern (`queryContext`):**
```go
func (s *DoltStore) queryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
    ctx, span := doltTracer.Start(ctx, "dolt.query",
        trace.WithSpanKind(trace.SpanKindClient),
        trace.WithAttributes(append(s.doltSpanAttrs(),
            attribute.String("db.operation", "query"),
            attribute.String("db.statement", spanSQL(query)),
        )...),
    )
    var rows *sql.Rows
    err := s.withRetry(ctx, func() error {
        rows, queryErr = s.db.QueryContext(ctx, query, args...)
        return queryErr
    })
    endSpan(span, wrapLockError(err))
    return rows, err
}
```

---

### 4. Dolt Version Control Telemetry (`internal/storage/dolt/store.go`)

Version control operations emit spans directly in `store.go` via `doltTracer.Start()`. These are **not** in `versioned.go` (which has no OTel imports).

Implemented spans (see Appendix for exact source locations):
- `dolt.commit` — `CALL DOLT_COMMIT`
- `dolt.push` — `CALL DOLT_PUSH`
- `dolt.pull` — `CALL DOLT_PULL`
- `dolt.merge` — `CALL DOLT_MERGE`
- `dolt.branch` — `CALL DOLT_BRANCH`
- `dolt.checkout` — `CALL DOLT_CHECKOUT`

---

### 5. Hook Telemetry (`internal/hooks/`)

Hooks emit a single root span per execution (`hook.exec`). There are no metric counters or histograms for hooks — only span-level observability. Duration metrics are a roadmap item (Tier 3).

---

## Metric Naming Convention

OTel SDK uses dot-notation internally. Prometheus-compatible backends (VictoriaMetrics, Prometheus) export these as underscore-separated names with type suffixes:

| Code name | Exported name |
|-----------|---------------|
| `issuegraph.storage.operations` | `issuegraph_storage_operations_total` |
| `issuegraph.storage.operation.duration` | `issuegraph_storage_operation_duration_ms` |
| `issuegraph.storage.errors` | `issuegraph_storage_errors_total` |
| `issuegraph.issue.count` | `issuegraph_issue_count` |
| `issuegraph.db.retry_count` | `issuegraph_db_retry_count_total` |
| `issuegraph.db.lock_wait_ms` | `issuegraph_db_lock_wait_ms` |
| `issuegraph.db.circuit_trips` | `issuegraph_db_circuit_trips_total` |
| `issuegraph.db.circuit_rejected` | `issuegraph_db_circuit_rejected_total` |
| `issuegraph.ai.input_tokens` | `issuegraph_ai_input_tokens_total` |
| `issuegraph.ai.output_tokens` | `issuegraph_ai_output_tokens_total` |
| `issuegraph.ai.request.duration` | `issuegraph_ai_request_duration_ms` |

---

## Environment Variables

### IssueGraph-Level Variables

| Variable | Set by | Description |
|-----------|----------|-------------|
| `ISSUEGRAPH_OTEL_METRICS_URL` | Operator | OTLP metrics endpoint (default: localhost:8428) |
| `ISSUEGRAPH_OTEL_LOGS_URL` | Operator | OTLP logs endpoint (reserved for future log export) |
| `ISSUEGRAPH_OTEL_STDOUT` | Operator | **Opt-in**: Write spans and metrics to stderr (dev/debug). Also activates telemetry. |

### Context Variables

| Variable | Source | Used By |
|-----------|--------|----------|
| `ISSUEGRAPH_ACTOR` | Git config / env var | Actor identity for audit trails (ISSUEGRAPH_ACTOR still works as deprecated alias) |
| `ISSUEGRAPH_NAME` | Environment | Binary name override (for multi-instance setups) |
| `OTEL_RESOURCE_ATTRIBUTES` | Operator | Custom resource attributes for all spans |

### Dolt-Specific Variables (See DOLT.md)

| Variable | Purpose |
|-----------|----------|
| `ISSUEGRAPH_DOLT_PASSWORD` | Server mode password |
| `ISSUEGRAPH_DOLT_SERVER_MODE` | Enable server mode |
| `ISSUEGRAPH_DOLT_SERVER_HOST` | Server host (default: 127.0.0.1) |
| `ISSUEGRAPH_DOLT_SERVER_PORT` | Server port (default: 3307, 3308 in shared mode, or derived) |
| `ISSUEGRAPH_DOLT_SERVER_TLS` | Enable TLS for server connections |
| `ISSUEGRAPH_DOLT_SERVER_USER` | MySQL connection user |
| `DOLT_REMOTE_USER` | Push/pull auth user |
| `DOLT_REMOTE_PASSWORD` | Push/pull auth password |

> **Note**: Dolt-specific configuration variables are documented in [DOLT.md](../../../docs/architecture/dolt.md) and are out of scope for OTEL design documentation.

---

## Event Types

### CLI Command Events

| Event | Trigger | Key Attributes |
|-------|---------|----------------|
| `issuegraph.command.<name>` | Each `issuegraph` subcommand execution | `issuegraph.command`, `issuegraph.version`, `issuegraph.args`, `issuegraph.actor` |

### Storage Events

| Event | Trigger | Key Attributes |
|-------|---------|----------------|
| `storage.CreateIssue` | Issue creation | `issuegraph.issue.id`, `issuegraph.issue.type`, `issuegraph.actor` |
| `storage.UpdateIssue` | Issue update | `issuegraph.issue.id`, `issuegraph.update.count`, `issuegraph.actor` |
| `storage.GetIssue` | Issue lookup | `issuegraph.issue.id` |
| `storage.SearchIssues` | Issue search | `issuegraph.query`, `issuegraph.result.count` |
| `storage.GetReadyWork` | Ready work query | `issuegraph.result.count` |
| `storage.GetBlockedIssues` | Blocked issues query | `issuegraph.result.count` |
| `storage.RunInTransaction` | Transaction execution | `db.commit_msg` |

### Dolt Events

| Event | Trigger | Key Attributes |
|-------|---------|----------------|
| `dolt.query` | Each SQL query (`queryContext`) | `db.operation`, `db.statement` |
| `dolt.exec` | Each SQL write (`execContext`) | `db.operation`, `db.statement` |
| `dolt.query_row` | Single-row queries (`queryRowContext`) | `db.operation`, `db.statement` |
| `dolt.commit` | DOLT_COMMIT operation | `commit_msg` |
| `dolt.push` | DOLT_PUSH operation | `dolt.branch` |
| `dolt.pull` | DOLT_PULL operation | `dolt.branch` |
| `dolt.merge` | DOLT_MERGE operation | `dolt.merge_branch` |
| `dolt.branch` | DOLT_BRANCH operation | `dolt.branch` |
| `dolt.checkout` | DOLT_CHECKOUT operation | `dolt.branch` |

### Hooks Events

| Event | Trigger | Key Attributes |
|-------|---------|----------------|
| `hook.exec` | Hook execution (span only — no metric counters) | `hook.event`, `hook.path`, `issuegraph.issue_id` |

---

## Monitoring Gaps

### Currently Monitored ✅

| Area | Coverage |
|-------|----------|
| Storage operations | Full (all CRUD, queries, transactions) |
| CLI command lifecycle | Full (all commands with arguments) |
| Dolt SQL queries | Full (all queries via queryContext/execContext wrappers) |
| Dolt retry counting | Full (retry counter incremented in withRetry) |
| Dolt version control | Full (commit, push, pull, merge, branch, checkout) |
| AI compaction | Full (issuegraph.ai.* metrics in compact/haiku.go) |

### Not Currently Monitored ❌

| Area | Notes | Operational Impact |
|-------|-------|-------------------|
| **Dolt lock wait time** | `issuegraph.db.lock_wait_ms` registered but `.Record()` not called | Lock contention invisible |
| **Dolt server lifecycle** | `internal/doltserver/` has no OTel imports | Server crashes are silent |
| **Hook execution time** | `hook.exec` span exists but no duration histogram | Cannot detect hook regressions |
| **versioned.go operations** | `versioned.go` has no OTel imports | History/AsOf/Diff invisible |
| **Dolt server metrics** | Dolt has internal metrics but not exposed to OTel | Cannot monitor server health, connection count, query load |
| **Working set size** | Uncommitted changes count unknown | Cannot detect batch mode accumulation |
| **Database size growth** | Dolt database size not tracked | Cannot plan capacity or detect bloat |
| **Branch proliferation** | Branch count not exposed | Cannot detect cleanup needed |
| **Remote sync bandwidth** | Bytes transferred not tracked | Cannot monitor network usage or cost |
| **Query execution plans** | EXPLAIN ANALYZE not captured | Cannot identify slow queries |
| **Connection pool utilization** | Active/idle counts not tracked | Cannot tune connection pool sizing |

---

## Queries

### Metrics (Any OTLP-compatible backend)

**Total counts by operation:**
```promql
sum(rate(issuegraph_storage_operations_total[5m])) by (db.operation)
sum(rate(issuegraph_db_retry_count_total[5m]))
```

**Latency distributions:**
```promql
histogram_quantile(0.50, issuegraph_storage_operation_duration_ms) by (db.operation)
histogram_quantile(0.95, issuegraph_storage_operation_duration_ms) by (db.operation)
histogram_quantile(0.99, issuegraph_storage_operation_duration_ms) by (db.operation)
```

**Issue counts by status:**
```promql
issuegraph_issue_count{status="open"}
issuegraph_issue_count{status="in_progress"}
issuegraph_issue_count{status="closed"}
issuegraph_issue_count{status="deferred"}
```

---

## Dolt Telemetry Capabilities

### Dolt Internal Metrics

**Important**: Dolt does not provide native OpenTelemetry export. The documentation search confirms there is no Dolt configuration variable or feature to enable OTLP export.

Dolt exposes internal metrics only via:
- `performance_schema` tables (MySQL standard, accessible via SQL queries)
- System tables (`dolt_log`, `dolt_status`, `dolt_diff`, `dolt_branches`, `dolt_conflicts`)

**IssueGraph implementation**:
IssueGraph currently queries Dolt metrics via direct SQL (see `modules/cli/doctor/perf_dolt.go`) rather than via OTLP. This is intentional — Dolt lacks native OTel support.

### Dolt System Tables for Telemetry

| Table | Purpose |
|--------|-----------|
| `dolt_log` | Commit history (queryable for audit) |
| `dolt_status` | Working set state (uncommitted changes) |
| `dolt_diff` | Cell-level diff between commits |
| `dolt_branches` | Branch metadata |
| `dolt_conflicts` | Merge conflicts (when present) |

### Sample Queries for Dolt Telemetry

**Commit frequency analysis:**
```sql
SELECT
    DATE_FORMAT(commit_date, '%Y-%m') as month,
    COUNT(*) as commits
FROM dolt_log
GROUP BY month
ORDER BY month DESC;
```

**Working set size tracking:**
```sql
SELECT
    COUNT(*) as staged_changes,
    SUM(CASE WHEN staged = 1 THEN 1 ELSE 0 END) as added,
    SUM(CASE WHEN staged = 0 THEN 1 ELSE 0 END) as removed
FROM dolt_status;
```

**Branch proliferation detection:**
```sql
SELECT
    COUNT(*) as branch_count,
    MIN(commit_date) as oldest,
    MAX(commit_date) as newest
FROM dolt_branches;
```

**Conflict analysis:**
```sql
SELECT
    COUNT(*) as conflict_count,
    COUNT(DISTINCT table_name) as tables_affected
FROM dolt_conflicts;
```

---

## Related Documentation

- [OTel Data Model](otel-data-model.md) — Complete event schema
- [OBSERVABILITY.md](../../../docs/reference/observability.md) — Quick reference for metrics
- [Dolt Backend](../../../docs/architecture/dolt.md) — Dolt configuration and usage
- [Dolt Concurrency](../dolt-concurrency.md) — Concurrency model and transactions

## Backends Compatible with OTLP

| Backend | Notes |
|---------|-------|
| **VictoriaMetrics** | Default for metrics (localhost:8428) — open source. Override with `ISSUEGRAPH_OTEL_METRICS_URL` |
| **VictoriaLogs** | Reserved for future log export. Override with `ISSUEGRAPH_OTEL_LOGS_URL` |
| **Prometheus** | Supports OTLP via remote_write receiver — open source |
| **Grafana Mimir** | Supports OTLP via write endpoint — open source |
| **Loki** | Requires OTLP bridge (Loki uses different format) — open source |
| **OpenTelemetry Collector** | Universal forwarder to any backend (recommended for production) — open source |

**Production Recommendation**: For production deployments, consider using **OpenTelemetry Collector** as a sidecar. The Collector provides:
- Single agent for all telemetry
- Advanced processing and batching
- Support for multiple backends simultaneously
- Better resource efficiency than per-process exporters

---

## Appendix: Source Reference Audit

Audited against **`main` @ `371df32b`**. All line numbers below refer to that commit.

Every factual claim in this document is backed by a specific source location. This table exists to prevent documentation drift and to make it easy to re-verify after code changes.

### Initialization (`internal/telemetry/telemetry.go`, `modules/cli/main.go`)

| Claim | Source |
|-------|--------|
| `Init` signature — returns only `error` | `telemetry.go:64` |
| `Enabled()` — true when `ISSUEGRAPH_OTEL_METRICS_URL` set or `ISSUEGRAPH_OTEL_STDOUT=true` | `telemetry.go:53-55` |
| Traces: stdout only when `ISSUEGRAPH_OTEL_STDOUT=true` | `telemetry.go:84-93` |
| Metrics: HTTP OTLP when `ISSUEGRAPH_OTEL_METRICS_URL` set | `telemetry.go:131-139` |
| Resource: `service.name`, `service.version` | `telemetry.go:73-75` |
| Resource: `WithHost()`, `WithProcess()` | `telemetry.go:76-77` |
| `Shutdown(ctx)` signature | `telemetry.go:162` |
| `Init` called in `PersistentPreRun` | `main.go:256` |
| Command span started with `issuegraph.command`, `issuegraph.version`, `issuegraph.args` | `main.go:262-266` |
| `issuegraph.actor` set on span after actor resolution | `main.go:474` |
| `Shutdown` called in `PersistentPostRun` | `main.go:681` |

### Storage Instrumentation (`internal/telemetry/storage.go`)

| Claim | Source |
|-------|--------|
| `WrapStorage` returns original store when telemetry disabled | `storage.go:33-36` |
| Metric `issuegraph.storage.operations` (Counter) | `storage.go:38-40` |
| Metric `issuegraph.storage.operation.duration` (Histogram, ms) | `storage.go:41-44` |
| Metric `issuegraph.storage.errors` (Counter) | `storage.go:45-47` |
| Gauge `issuegraph.issue.count` | `storage.go:48-50` |
| `CreateIssue` — attrs: `issuegraph.actor`, `issuegraph.issue.type` | `storage.go:86` |
| `UpdateIssue` — attrs: `issuegraph.issue.id`, `issuegraph.update.count`, `issuegraph.actor` | `storage.go:131` |
| `GetIssue` — attr: `issuegraph.issue.id` | `storage.go:108` |
| `SearchIssues` — attrs: `issuegraph.query`, `issuegraph.result.count` | `storage.go:162` |
| `GetReadyWork` — attr: `issuegraph.result.count` | `storage.go:283` |
| `GetBlockedIssues` — attr: `issuegraph.result.count` | `storage.go:293` |
| `RunInTransaction` — attr: `db.commit_msg` | `storage.go:393` |
| `GetStatistics` emits gauge broken down by status | `storage.go:349` |
| `AddDependency`, `RemoveDependency`, `GetDependencies` instrumented | `storage.go:175, 187, 198` |
| `AddLabel`, `RemoveLabel`, `GetLabels` instrumented | `storage.go:243, 254, 265` |

### Dolt Backend (`internal/storage/dolt/store.go`)

| Claim | Source |
|-------|--------|
| `doltTracer` package-level var | `store.go:288` |
| Metric `issuegraph.db.retry_count` (Counter) registered | `store.go:302` |
| `retryCount.Add()` called when `attempts > 1` | `store.go:281` |
| Metric `issuegraph.db.lock_wait_ms` (Histogram) registered | `store.go:306` |
| `lockWaitMs.Record()` never called anywhere | grep `store.go` for `lockWaitMs\.Record` → zero matches |
| Metric `issuegraph.db.circuit_trips` (Counter) registered | `store.go:310` |
| `circuitTrips.Add()` called on circuit open | `store.go:265` |
| Metric `issuegraph.db.circuit_rejected` (Counter) registered | `store.go:314` |
| `circuitRejected.Add()` called on fail-fast | `store.go:250, 554` |
| `withRetry()` function | `store.go:247` |
| `execContext()` uses `doltTracer.Start()` + `withRetry()` | `store.go:359` |
| `queryContext()` uses `doltTracer.Start()` + `withRetry()` | `store.go:396` |
| `queryRowContext()` uses `doltTracer.Start()` + `withRetry()` | `store.go:425` |
| Span `dolt.commit` | `store.go:1086` |
| Span `dolt.push` | `store.go:1231, 1266` |
| Span `dolt.pull` | `store.go:1295` |
| Span `dolt.merge` | `store.go:1389` |
| Span `dolt.branch` | `store.go:1357` |
| Span `dolt.checkout` | `store.go:1372` |

### versioned.go — no OTel

| Claim | Source |
|-------|--------|
| `versioned.go` has no OTel imports | `versioned.go:1-9` — imports: `context`, `fmt`, `storage`, `types` only |

### doltserver — no OTel

| Claim | Source |
|-------|--------|
| `internal/doltserver/` has no OTel imports | grep `internal/doltserver/*.go` for `otel\|telemetry\|otlp` → zero matches |

### Hooks (`internal/hooks/`)

| Claim | Source |
|-------|--------|
| Span `hook.exec` created in `runHook` | `hooks_unix.go:31` |
| Span attrs: `hook.event`, `hook.path`, `issuegraph.issue_id` | `hooks_unix.go:33-35` |
| Stdout/stderr added as span events via `addHookOutputEvents` | `hooks_otel.go:14, 20` |
| `hook.stdout` / `hook.stderr` events carry `output`, `bytes` attrs | `hooks_otel.go:15-16, 21-22` |
| No metric counters or histograms for hooks | grep `internal/hooks/` for `Counter\|Histogram` → zero matches |

### AI (`internal/compact/haiku.go`, `modules/cli/find_duplicates.go`)

| Claim | Source |
|-------|--------|
| Metric `issuegraph.ai.input_tokens` (Counter) | `haiku.go:110` |
| Metric `issuegraph.ai.output_tokens` (Counter) | `haiku.go:114` |
| Metric `issuegraph.ai.request.duration` (Histogram, ms) | `haiku.go:118` |
| Metrics initialized lazily via `aiMetricsOnce` | `haiku.go:62, 106` |
| Span `anthropic.messages.new` | `haiku.go:126` |
| Span attrs: `issuegraph.ai.model`, `issuegraph.ai.operation` | `haiku.go:129-130` |
| Span attrs: `issuegraph.ai.input_tokens`, `issuegraph.ai.output_tokens`, `issuegraph.ai.attempts` | `haiku.go:165-167` |
| Retry on HTTP 429 and 5xx | `haiku.go:217` |
| `find_duplicates.go` — span attrs only, no `aiMetrics.*` calls | `find_duplicates.go:429-454` |
| `find_duplicates.go` — `issuegraph.ai.batch_size` attr | `find_duplicates.go:433` |
