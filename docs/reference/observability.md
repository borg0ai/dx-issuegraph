---
title: Observability (OpenTelemetry)
description: Exporting issuegraph metrics and traces over OpenTelemetry (OTLP), with a local VictoriaMetrics and Grafana stack, env vars, and a metric reference.
---

IssueGraph exports metrics via OTLP HTTP. Telemetry is **disabled by default** — zero overhead when no variable is set.

## Recommended local stack

| Service | Port | Role |
|---------|------|------|
| VictoriaMetrics | 8428 | OTLP metrics storage |
| VictoriaLogs | 9428 | Reserved for future OTLP log storage |
| Grafana | 9429 | Dashboards |

```bash
# From your personal stack's opentelemetry/ folder
docker compose up -d
```

## Configuration

Telemetry is **explicit opt-in**. Set `ISSUEGRAPH_OTEL_ENABLED=true` and configure the
exporter via standard OpenTelemetry SDK environment variables — both go in
your shell profile or workspace `.env`:

```bash
export ISSUEGRAPH_OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=http://localhost:8428/opentelemetry/api/v1/push
```

A standard `OTEL_*` variable alone will not turn issuegraph telemetry on — `issuegraph` won't
auto-activate from a machine-global `OTEL_*` setting that was set for some
other instrumented tool.

Log export is not implemented yet. `ISSUEGRAPH_OTEL_LOGS_URL` is reserved for a future
VictoriaLogs exporter and does not activate telemetry today.

### Shell profile (recommended)

```bash
# ~/.zshrc or ~/.bashrc
export ISSUEGRAPH_OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=http://localhost:8428/opentelemetry/api/v1/push
```

### Environment variables

| Variable | Example | Description |
|----------|---------|-------------|
| `ISSUEGRAPH_OTEL_ENABLED` | `true` | **Master switch.** Activates telemetry. Without it, the variables below are ignored. |
| `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` | `http://localhost:8428/opentelemetry/api/v1/push` | Push metrics to an OTLP HTTP receiver. |
| `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` | `http://localhost:9428/insert/opentelemetry/v1/logs` | Push logs to an OTLP HTTP receiver (reserved for future log export). |
| `OTEL_TRACES_EXPORTER` | `console` | Write spans to stderr (dev/debug). |
| `OTEL_METRICS_EXPORTER` | `console` | Metric exporter selection, comma-separated (`otlp`, `console`, `none`). Unset defaults to `otlp`; `console` writes to stderr (dev/debug); a selection without `otlp` never exports to an OTLP endpoint, even if one is configured machine-globally. |
| `OTEL_SERVICE_NAME` | `issuegraph` | Override the `service.name` resource attribute. |
| `OTEL_RESOURCE_ATTRIBUTES` | `deployment.environment=workstation,team=infra` | Extend or override resource attributes (comma-separated `key=value`). |
| `OTEL_SDK_DISABLED` | `true` | Force telemetry off even when `ISSUEGRAPH_OTEL_ENABLED=true` is set. |

### Resource attributes

Every metric and span carries the OTel resource describing the issuegraph process:

| Attribute | Value | Notes |
|-----------|-------|-------|
| `service.name` | `issuegraph` | Override with `OTEL_SERVICE_NAME`. |
| `service.version` | issuegraph version | |

Add anything else via `OTEL_RESOURCE_ATTRIBUTES`.

### Local debug mode

```bash
ISSUEGRAPH_OTEL_ENABLED=true OTEL_TRACES_EXPORTER=console OTEL_METRICS_EXPORTER=console issuegraph list
```

### Legacy environment variables (deprecated)

The earlier `ISSUEGRAPH_OTEL_*` data variables are honored for backwards
compatibility. Setting any of them activates telemetry on its own (no
`ISSUEGRAPH_OTEL_ENABLED=true` required) and translates to the standard OTLP equivalent
— a legacy value wins over a pre-existing `OTEL_*` value so a machine-global
`OTEL_*` setting cannot silently redirect issuegraph telemetry. Each `issuegraph` invocation
that sees one logs a one-line deprecation warning to stderr:

| Legacy | Standard equivalent |
|--------|---------------------|
| `ISSUEGRAPH_OTEL_METRICS_URL` | `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` |
| `ISSUEGRAPH_OTEL_LOGS_URL` | `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` |
| `ISSUEGRAPH_OTEL_STDOUT=true` | `OTEL_TRACES_EXPORTER=console` + `OTEL_METRICS_EXPORTER=console` |

## Verification

```bash
issuegraph list   # triggers metrics → visible in VictoriaMetrics
```

Verification queries in Grafana (VictoriaMetrics datasource):

```promql
issuegraph_storage_operations_total                              # raw counter
sum(rate(issuegraph_storage_operations_total[5m]))               # operation rate
```

### Confirm storage instrumentation locally

To verify the storage decorator chain is wired up without standing up a
collector, run `issuegraph` with stdout exporters and look for `issuegraph.storage.*`
records on stderr:

```bash
ISSUEGRAPH_OTEL_STDOUT=true issuegraph list 2>&1 | grep -F issuegraph.storage.operations
```

Expect at least one line per storage call (`GetReadyWork`, `GetIssue`, …).
If `issuegraph.storage.*` and `issuegraph.issue.count` are absent but `issuegraph.db.pool_*` is
present, the storage decorator is not in the chain — check
`wireStorageDecorators` in `modules/cli/storage_chain.go`.

---

## Metrics

### Storage (`issuegraph_storage_*`)

| Metric | Type | Attributes | Description |
|--------|------|------------|-------------|
| `issuegraph_storage_operations_total` | Counter | `db.operation` | Storage operations executed |
| `issuegraph_storage_operation_duration_ms` | Histogram | `db.operation` | Operation duration (ms) |
| `issuegraph_storage_errors_total` | Counter | `db.operation` | Storage errors |

> These metrics are emitted by `InstrumentedStorage`, the issuegraph SDK wrapper.

### Dolt database (`issuegraph_db_*`)

| Metric | Type | Attributes | Description |
|--------|------|------------|-------------|
| `issuegraph_db_retry_count_total` | Counter | — | SQL retries in server mode |
| `issuegraph_db_lock_wait_ms` | Histogram | `dolt_lock_exclusive` | Wait time to acquire database locks |

### Issues (`issuegraph_issue_*`)

| Metric | Type | Attributes | Description |
|--------|------|------------|-------------|
| `issuegraph_issue_count` | Gauge | `status` | Number of issues by status |

`status` values: `open`, `in_progress`, `closed`, `deferred`.

### AI (`issuegraph_ai_*`)

| Metric | Type | Attributes | Description |
|--------|------|------------|-------------|
| `issuegraph_ai_input_tokens_total` | Counter | `issuegraph_ai_model` | Anthropic input tokens |
| `issuegraph_ai_output_tokens_total` | Counter | `issuegraph_ai_model` | Anthropic output tokens |
| `issuegraph_ai_request_duration_ms` | Histogram | `issuegraph_ai_model` | API call latency |

---

## Traces (spans)

Spans are only exported when `OTEL_TRACES_EXPORTER=console` — there is no trace backend in the recommended local stack.

| Span | Source | Description |
|------|--------|-------------|
| `issuegraph.command.<name>` | CLI | Total duration of the command |
| `dolt.exec` / `dolt.query` / `dolt.query_row` | SQL | Each SQL operation |
| `dolt.commit` / `dolt.push` / `dolt.pull` / `dolt.merge` | Dolt VC | Version control procedures |
| `ephemeral.count` / `ephemeral.nuke` | SQLite | Ephemeral store operations |
| `hook.exec` | Hooks | Hook execution (root span, fire-and-forget) |
| `tracker.sync` / `tracker.pull` / `tracker.push` | Sync | Tracker sync phases |
| `anthropic.messages.new` | AI | Claude API calls |

### Notable attributes

**`issuegraph.command.<name>`**

| Attribute | Description |
|-----------|-------------|
| `issuegraph.command` | Subcommand name (`list`, `create`, ...) |
| `issuegraph.version` | issuegraph version |
| `issuegraph.args` | Raw arguments passed to the command (e.g. "create 'title' -p 2") |
| `issuegraph.actor` | Actor (resolved from git config / env) |

**`hook.exec`**

| Attribute / Event | Description |
|-------------------|-------------|
| `hook.event` | Event type (`create`, `update`, `close`) |
| `hook.path` | Absolute path to the script |
| `issuegraph.issue_id` | ID of the triggering issue |
| event `hook.stdout` | Script standard output (truncated to 1 024 bytes) |
| event `hook.stderr` | Script error output (truncated to 1 024 bytes) |

The `hook.stdout` / `hook.stderr` events carry two attributes: `output` (the text) and `bytes` (original size before truncation).

---

## Architecture

```
modules/cli/main.go
  └─ telemetry.Init()
      ├─ OTEL_TRACES_EXPORTER=console        → TracerProvider stdout
      ├─ OTEL_METRICS_EXPORTER=console       → MeterProvider stdout
      └─ OTEL_EXPORTER_OTLP_METRICS_ENDPOINT → MeterProvider HTTP → VictoriaMetrics

internal/storage/dolt/        → issuegraph_db_* metrics + dolt.* spans
internal/storage/ephemeral/   → ephemeral.* spans
internal/hooks/               → hook.exec span
internal/tracker/             → tracker.* spans
internal/compact/             → issuegraph_ai_* metrics + anthropic.* spans
internal/telemetry/storage.go → issuegraph_storage_* metrics (SDK wrapper)
```

When no OpenTelemetry SDK environment variable selects an exporter,
`telemetry.Init()` installs **no-op** providers: hot paths execute only no-op
calls with no memory allocation.
