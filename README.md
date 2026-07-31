# Hexatron

Service-centric operational dashboard for the Open Climate Fix platform.

Instead of exposing Airflow directly, Hexatron presents the health of business
services (Site Forecast, Consumer, Data Platform, Quartz) so engineers can answer
one question quickly: **"Is everything running as expected?"**

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

Serving live Airflow data. Service health is aggregated from the latest DAG run
of every DAG listed in `data/services.yaml`.

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

### Configuration

| Variable                 | Default              | Purpose                     |
|--------------------------|----------------------|-----------------------------|
| `AIRFLOW_BASE_URL`       | `http://127.0.0.1:38000` | Airflow root URL        |
| `AIRFLOW_SESSION_COOKIE` | —                    | Session cookie (required)   |
| `PORT`                   | `:8080`              | Listen address              |
| `SERVICES_CONFIG_PATH`   | `data/services.yaml` | Service definitions         |

At startup Hexatron logs any drift between `services.yaml` and Airflow — DAGs
configured but missing, and DAGs Airflow runs that no service claims. Drift is a
warning, not a fatal error; a missing DAG reports as `unknown`.

---

## API Contract

Base URL: `http://localhost:8080`

### `GET /services`

Returns the health status of all configured business services.

**Query Parameters**

| Parameter  | Type   | Required | Description                                       |
|------------|--------|----------|---------------------------------------------------|
| `search`   | string | No       | Case-insensitive substring filter on service name |
| `category` | string | No       | Exact match on service category                   |

**Response**

```json
[
  { "id": "site-forecast", "name": "Site Forecast", "category": "Forecast", "status": "running" },
  { "id": "cloudcasting",  "name": "Cloudcasting",  "category": "Forecast", "status": "failed"  },
  { "id": "consumer",      "name": "Consumer",      "category": "Consumer", "status": "failed"  },
  { "id": "data-platform", "name": "Data Platform", "category": "Platform", "status": "healthy" }
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
  "id":       "cloudcasting",
  "name":     "Cloudcasting",
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

`last_run` is absent for a DAG that has never run.

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
curl http://localhost:8080/services/site-forecast
curl http://localhost:8080/services/consumer
curl http://localhost:8080/services/data-platform
```

---

### Status Values

Derived from the **latest run** of each DAG.

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

All responses include `Content-Type: application/json`.

---

### Configured Services

Defined in [`backend/data/services.yaml`](backend/data/services.yaml), which
groups the 21 DAGs in the OCF Airflow deployment into six services.

| Service ID          | Category | DAGs |
|---------------------|----------|------|
| `site-forecast`     | Forecast | `uk-forecast-site`, `nl-forecast` |
| `national-forecast` | Forecast | `uk-forecast-gsp` |
| `cloudcasting`      | Forecast | `uk-forecast-clouds`, `uk-analysis-clouds` |
| `consumer`          | Consumer | `uk-consume-neso`, `uk-consume-nwp`, `uk-consume-pv`, `uk-consume-pvlive-dayafter`, `uk-consume-pvlive-intraday`, `uk-consume-sat`, `uk-consume-sat-v1`, `nl-consume-ned-nl`, `nl-consume-ned-nl-forecast`, `nl-consume-nwp` |
| `quartz`            | API      | `uk-api-quartz-national-gsp-check`, `uk-api-site-check`, `nl-api-check` |
| `data-platform`     | Platform | `uk-manage-clean-up-logs`, `uk-manage-elb`, `uk-manage-sitedb-cleanup` |

The grouping is a first pass over the DAG naming convention — adjust it in the
YAML, which is the only place service membership is defined.

### Known limitations

- **No caching.** Each request re-queries Airflow: one `/dags` call plus one call
  per DAG, fanned out 8 at a time. Fine locally, worth caching (Phase 7) before
  a dashboard auto-refreshes against shared Airflow.
- **Latest run only.** A DAG that failed repeatedly and is now retrying reports
  `running`, not `failed`. Reading back to the last *completed* run would fix
  this.
