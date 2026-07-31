# Hexatron

Service-centric operational dashboard for the Open Climate Fix platform.

Instead of exposing Airflow directly, Hexatron presents the health of business
services (Solar Forecast, Wind Forecast, Consumers, DP, API, UI) so engineers can
answer one question quickly: **"Is everything running as expected?"**

## Structure

```
backend/                        Go module
├── cmd/server/                 Entrypoint
├── internal/
│   ├── routes/                 Route registration
│   ├── controllers/            HTTP handlers, one per endpoint
│   ├── services/               Business logic: registry + health aggregation
│   ├── clients/                Airflow + CloudWatch API clients
│   ├── models/                 Domain types (models/airflow: API shapes)
│   ├── structures/             Config, request and response payloads
│   ├── constants/              Shared constants
│   ├── middleware/             CORS, logging, recovery, request ID
│   └── utils/                  Shared helpers
├── data/                       services.yaml — single source of truth
└── tests/                      Test suite

frontend/                       Dashboard UI
docs/                           Design and implementation notes
```

## Status

Backend: serving live Airflow data. Service health is aggregated from the latest
DAG run of every DAG listed in `data/services.yaml`.

Frontend: dashboard PoC running on stub data (`frontend/src/lib/stub-data.ts`).
All six services, the dependency graph and the five health states render. Set
`USE_STUB_DATA=false` to point it at the live backend — the fields the stub
carries that the API does not yet serve are listed under
[Proposed contract additions](#proposed-contract-additions).

## Getting started

Port-forward Airflow, then grab a `session` cookie from your browser's dev tools
after logging into the Airflow UI:

```bash
cp .env.example .env          # fill in AIRFLOW_SESSION_COOKIE
set -a && . ./.env && set +a  # Go does not read .env itself
cd backend && make run        # or: go run ./cmd/server
# Server starts on http://localhost:8080
```

The cookie expires; a stale one makes requests fail with `502` and
`{"error": "airflow session cookie is missing or expired"}`.

Run the tests with `make test`. They use a fake Airflow server, so no live
Airflow or cookie is needed.

Then the dashboard:

```bash
cd frontend && pnpm install && pnpm dev   # http://localhost:3000
```

The frontend serves stub data by default. Set `USE_STUB_DATA=false` (and
`API_BASE_URL` if the backend is not on localhost:8080) to point it at the
backend above.

### Configuration

Backend:

| Variable                 | Default              | Purpose                     |
|--------------------------|----------------------|-----------------------------|
| `AIRFLOW_BASE_URL`       | `http://127.0.0.1:38000` | Airflow root URL        |
| `AIRFLOW_SESSION_COOKIE` | —                    | Session cookie (required)   |
| `PORT`                   | `:8080`              | Listen address              |
| `SERVICES_CONFIG_PATH`   | `data/services.yaml` | Service definitions         |

Frontend:

| Variable         | Default                 | Purpose                          |
|------------------|-------------------------|----------------------------------|
| `USE_STUB_DATA`  | `true`                  | Serve `stub-data.ts` instead of the API |
| `API_BASE_URL`   | `http://localhost:8080` | Backend root URL                 |

At startup Hexatron logs any drift between `services.yaml` and Airflow — DAG
patterns that match nothing, and DAGs Airflow runs that no service claims. Drift
is a warning, not a fatal error; a service left with no DAGs reports `unknown`.

---

## API Contract

Base URL: `http://localhost:8080`

### `GET /services`

Returns the health status of all configured business services.

**Query Parameters**

| Parameter  | Type   | Required | Description                                                 |
|------------|--------|----------|-------------------------------------------------------------|
| `search`   | string | No       | Case-insensitive substring on service name, category or DAG id |
| `category` | string | No       | Exact match on service category                             |

> **`search` scope widened — not yet implemented in the backend.** The dashboard's
> box is labelled "Search DAGs…", so it must match DAG ids (`pvlive` should find
> Consumers) as well as service names. The frontend filters this way today; a
> name-only backend would silently return nothing for those queries once
> `USE_STUB_DATA=false`.

**Response**

```json
[
  { "id": "solar-forecast", "name": "Solar Forecast", "category": "Forecast",    "status": "running" },
  { "id": "wind-forecast",  "name": "Wind Forecast",  "category": "Forecast",    "status": "unknown" },
  { "id": "consumer",       "name": "Consumers",      "category": "Consumer",    "status": "failed"  },
  { "id": "data-platform",  "name": "DP",             "category": "Platform",    "status": "healthy" },
  { "id": "api",            "name": "API",            "category": "Application", "status": "healthy" },
  { "id": "ui",             "name": "UI",             "category": "Application", "status": "unknown" }
]
```

**Examples**

```bash
curl http://localhost:8080/services
curl "http://localhost:8080/services?search=consumer"
curl "http://localhost:8080/services?category=Forecast"
```

---

### `GET /services/{serviceId}`

Returns a single service with the health status of each underlying DAG.

**Path Parameters**

| Parameter   | Type   | Required | Description           |
|-------------|--------|----------|-----------------------|
| `serviceId` | string | Yes      | The ID of the service |

**Response**

Each DAG entry carries the context needed to act on a failure without a second
request: the schedule, whether the DAG is paused, its last run, and a deep link
into Airflow.

```json
{
  "id":       "solar-forecast",
  "name":     "Solar Forecast",
  "category": "Forecast",
  "status":   "failed",
  "dags": [
    {
      "dag_id":      "uk-forecast-clouds",
      "name":        "uk-forecast-clouds",
      "status":      "healthy",
      "is_paused":   false,
      "schedule":    "12,42 * * * *",
      "last_run":    { "run_id": "scheduled__…", "state": "success",
                       "start_date": "…", "end_date": "…" },
      "airflow_url": "http://localhost:38000/dags/uk-forecast-clouds/grid"
    },
    {
      "dag_id":      "uk-analysis-clouds",
      "name":        "uk-analysis-clouds",
      "status":      "failed",
      "is_paused":   false,
      "schedule":    "0 6 * * *",
      "last_run":    { "run_id": "scheduled__…", "state": "failed",
                       "start_date": "…", "end_date": "…" },
      "airflow_url": "http://localhost:38000/dags/uk-analysis-clouds/grid"
    }
  ]
}
```

`last_run` is absent for a DAG that has never run. `dags` is empty for a service
whose patterns match nothing in Airflow — `wind-forecast` and `ui` today — and
its status is `unknown`.

The dashboard needs more per-DAG fields than this — see
[Proposed contract additions](#proposed-contract-additions).

**Status Codes**

| Code | Meaning           |
|------|-------------------|
| 200  | Success           |
| 404  | Service not found |

**404 Response**

```json
{ "error": "service not found: <serviceId>" }
```

**Examples**

```bash
curl http://localhost:8080/services/solar-forecast
curl http://localhost:8080/services/consumer
curl http://localhost:8080/services/data-platform
```

---

### Status Values

Served by the backend today, derived from the **latest run** of each DAG.

| Value     | DAG meaning                          | Service meaning              |
|-----------|--------------------------------------|------------------------------|
| `healthy` | Latest run succeeded                 | All DAGs healthy             |
| `failed`  | Latest run failed or upstream-failed | Any DAG failed               |
| `running` | Latest run is in progress            | Any DAG running, none failed |
| `queued`  | Latest run is queued                 | Any DAG queued, none above   |
| `unknown` | Never run, or DAG missing            | Any DAG unknown, none above  |

Precedence when aggregating a service: `failed` → `running` → `queued` →
`unknown` → `healthy`. Failure outranks everything, so a broken DAG is never
masked by a busy one.

> **Changed for the dashboard — not yet implemented in the backend.**
> The frontend models five states. `failed` is replaced by `degraded` and
> `down`, and `paused` is added, because collapsing them loses the distinction
> an operator most needs: whether something is broken, partly broken, or off on
> purpose. The backend already serves the per-DAG `is_paused` flag these need.

| Value      | Meaning                                                    |
|------------|------------------------------------------------------------|
| `healthy`  | Latest run succeeded and success rate is at or above 95%    |
| `degraded` | Latest run succeeded but success rate is below 95%          |
| `down`     | The latest run failed                                       |
| `paused`   | The DAGs are paused — no runs expected                       |
| `unknown`  | No monitoring data available                                 |

Deliberately rate-based rather than "any failure degrades the service": one
failure in 180 runs is noise, not a degradation. Threshold lives in
`frontend/src/lib/status.ts` (`DEGRADED_THRESHOLD`) until the backend owns it.

All responses include `Content-Type: application/json`.

---

### Configured Services

The six services are defined in
[`backend/data/services.yaml`](backend/data/services.yaml). No DAG is named
there: each service carries `dag_patterns`, globs matched against the `dag_id`s
Airflow reports on every request. Airflow is the only place DAGs are listed, so
a DAG renamed or added there needs no edit here.

| Service ID       | Name           | Category    | `dag_patterns` | Matches today |
|------------------|----------------|-------------|----------------|---------------|
| `solar-forecast` | Solar Forecast | Forecast    | `uk-forecast-*`, `uk-analysis-*`, `nl-forecast` | `uk-forecast-site`, `uk-forecast-gsp`, `uk-forecast-clouds`, `uk-analysis-clouds`, `nl-forecast` |
| `wind-forecast`  | Wind Forecast  | Forecast    | —              | none yet |
| `consumer`       | Consumers      | Consumer    | `*-consume-*`  | the 10 `uk-consume-*` and `nl-consume-*` DAGs |
| `data-platform`  | DP             | Platform    | `*-manage-*`   | `uk-manage-clean-up-logs`, `uk-manage-elb`, `uk-manage-sitedb-cleanup` |
| `api`            | API            | Application | `*-api-*`      | `uk-api-quartz-national-gsp-check`, `uk-api-site-check`, `nl-api-check` |
| `ui`             | UI             | Application | —              | none yet |

That accounts for all 21 DAGs in the OCF deployment, each claimed by exactly one
service. Cloudcasting predicts the cloud cover the solar forecast runs on, which
is why the `*-clouds` DAGs count against `solar-forecast`.

A service with no patterns is not a misconfiguration — `wind-forecast` and `ui`
have nothing in Airflow to watch yet, and report `unknown` until they do. A
pattern that matches *nothing*, on the other hand, is drift and is warned about
at startup: it usually means a DAG was renamed.

Patterns are globs, not substrings, and `*` does not span the whole id. That
anchoring is load-bearing: `nl-forecast` claims only the DAG of that name and
not `nl-consume-ned-nl-forecast`, which belongs to `consumer`.

### Known limitations

- **No caching.** Each request re-queries Airflow: one `/dags` call plus one call
  per claimed DAG, fanned out 8 at a time. Fine locally, worth caching (Phase 7)
  before a dashboard auto-refreshes against shared Airflow.
- **Latest run only.** A DAG that failed repeatedly and is now retrying reports
  `running`, not `failed`. Reading back to the last *completed* run would fix
  this.
- **Stub DAG ids are invented.** The six services line up with
  `frontend/src/lib/stub-data.ts`, but its DAG names (`solar_forecast_national`
  and so on) are demo values, not the ids Airflow actually serves.

---

### Proposed contract additions

Needed by the dashboard, not yet served by the backend. The frontend runs on
stub data shaped like this — see `frontend/src/lib/types.ts`, which is the
authoritative shape, and flip `USE_STUB_DATA=false` to hit the real API.

**`depends_on`** — upstream service ids, already in `services.yaml` but not yet
in the response. Drives the dependency graph on the dashboard. Data flows:

```
solar-forecast ┐
wind-forecast  ├─▶ data-platform ─▶ api ─▶ ui
consumer       ┘
```

**`metrics`** — per-service aggregates over the selected window. Field names
follow Airflow's own schema where they map, so the client can pass DagRun data
through with minimal transformation:

```json
{
  "metrics": {
    "total_runs": 75,
    "failed_runs": 6,
    "success_rate": 0.92,
    "avg_duration_seconds": 270,
    "latest_duration_seconds": 430,
    "last_run_at": "2026-07-29T10:05:00Z",
    "next_run_at": "2026-07-29T11:00:00Z"
  }
}
```

**`dags[]`** — the run-history strip needs more than the current entry carries.
`dag_display_name` is served as `name` and `is_paused` as-is; `schedule` holds
the cron string the frontend calls `timetable_summary`. Still missing:
`next_dagrun_logical_date` (the Airflow client already reads it, the response
does not expose it) and `runs[]`, recent `DagRun` objects with `state`,
`start_date` and `end_date`. Only the latest run is fetched today.

**`note`** — optional operator-facing string, e.g. `"Paused for maintenance"`.

Not yet designed: a time-window parameter for the `Last 24h` selector. The
control is currently presentational.

---

### Frontend stub data

`frontend/src/lib/stub-data.ts` backs the dashboard while `USE_STUB_DATA=true`.
It carries the same six services as the registry, plus the metrics and run
history the API does not serve yet. Its statuses are demo values chosen to
exercise all five frontend states.
