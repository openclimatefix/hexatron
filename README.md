# Hexatron

Service-centric operational dashboard for the Open Climate Fix platform.

Instead of exposing Airflow directly, Hexatron presents the health of business
services (Solar Forecast, Wind Forecast, Consumers, DP, API, UI) so engineers can
answer one question quickly: **"Is everything running as expected?"**

## Structure

```
backend/                        Go module
├── cmd/hexatron/               Entrypoint
├── internal/
│   ├── api/                    REST layer (GET /services, GET /services/{id})
│   ├── clients/
│   │   ├── airflow/            Airflow REST API client
│   │   └── cloudwatch/         CloudWatch client (future)
│   ├── services/               Service registry + health aggregation
│   └── models/                 Shared types
└── config/                     services.yaml — single source of truth

frontend/                       Dashboard UI
docs/                           Design and implementation notes
```

## Status

Backend: package scaffolding only. Airflow integration follows in Phase 2.

Frontend: dashboard PoC running on stub data (`frontend/src/lib/stub-data.ts`).
All six services, the dependency graph and the five health states render; no
live Airflow data yet.

## Getting started

```bash
cp .env.example .env

# Backend
cd backend && go run main.go        # http://localhost:8080

# Frontend
cd frontend && pnpm install && pnpm dev   # http://localhost:3000
```

The frontend serves stub data by default. Set `USE_STUB_DATA=false` (and
`API_BASE_URL` if not localhost:8080) to point it at the real backend.

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
  { "id": "solar-forecast", "name": "Solar Forecast", "status": "degraded" },
  { "id": "consumer",       "name": "Consumers",      "status": "healthy"  },
  { "id": "data-platform",  "name": "DP",             "status": "healthy"  }
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

```json
{
  "id":     "consumer",
  "name":   "Consumers",
  "status": "healthy",
  "dags": [
    { "dag_id": "ecmwf_consumer",     "status": "healthy"  },
    { "dag_id": "metoffice_consumer", "status": "degraded" },
    { "dag_id": "pvlive_consumer",    "status": "healthy"  }
  ]
}
```

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

> **Changed for the dashboard — not yet implemented in the backend.**
> The frontend models five states. `failed` is replaced by `degraded` and
> `down`, and `paused` is added, because collapsing them loses the distinction
> an operator most needs: whether something is broken, partly broken, or off on
> purpose.

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

### Proposed contract additions

Needed by the dashboard, not yet served by the backend. The frontend runs on
stub data shaped exactly like this — see `frontend/src/lib/types.ts`, which is
the authoritative shape, and flip `USE_STUB_DATA=false` to hit the real API.

**`depends_on`** — upstream service ids, already added to `services.yaml`. Drives
the dependency graph on the dashboard. Data flows:

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

**`dags[]`** — expanded beyond `{dag_id, status}` to carry the run-history strip:
`dag_display_name`, `is_paused`, `timetable_summary`,
`next_dagrun_logical_date`, and `runs[]` (recent `DagRun` objects with
`state`, `start_date`, `end_date`).

**`note`** — optional operator-facing string, e.g. `"Paused for maintenance"`.

Not yet designed: a time-window parameter for the `Last 24h` selector. The
control is currently presentational.

---

### Mock Data (Current Phase)

The registry lives in `backend/config/services.yaml`. The frontend's equivalent
stub — which also carries metrics and run history — is in
`frontend/src/lib/stub-data.ts`.

| Service ID       | Name           | Category    | DAGs                                                     | Status     |
|------------------|----------------|-------------|----------------------------------------------------------|------------|
| `solar-forecast` | Solar Forecast | Forecast    | `solar_forecast_national`, `solar_forecast_sites`         | `degraded` |
| `wind-forecast`  | Wind Forecast  | Forecast    | `wind_forecast_national`                                  | `down`     |
| `consumer`       | Consumers      | Consumer    | `ecmwf_consumer`, `metoffice_consumer`, `pvlive_consumer` | `healthy`  |
| `data-platform`  | DP             | Platform    | `save_to_dp`                                              | `healthy`  |
| `api`            | API            | Application | `api_healthcheck`                                         | `paused`   |
| `ui`             | UI             | Application | —                                                          | `unknown`  |

Statuses above are the stub's demo values, chosen to exercise all five states.
