# Hexatron

Service-centric operational dashboard for the Open Climate Fix platform.

Instead of exposing Airflow directly, Hexatron presents the health of business
services (Site Forecast, Consumer, Data Platform, Quartz) so engineers can answer
one question quickly: **"Is everything running as expected?"**

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

Controller logic implemented with mock responses. Airflow integration follows in Phase 2.

## Getting started

```bash
cp .env.example .env
cd backend && go run main.go
# Server starts on http://localhost:8080
```

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
  { "id": "site-forecast", "name": "Site Forecast", "status": "healthy" },
  { "id": "consumer",      "name": "Consumer",      "status": "failed"  },
  { "id": "data-platform", "name": "Data Platform", "status": "healthy" }
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
  "name":   "Consumer",
  "status": "failed",
  "dags": [
    { "dag_id": "ecmwf_consumer",     "status": "healthy" },
    { "dag_id": "metoffice_consumer", "status": "failed"  },
    { "dag_id": "pvlive_consumer",    "status": "healthy" }
  ]
}
```

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

| Value     | Meaning                                     |
|-----------|---------------------------------------------|
| `healthy` | All DAGs in the service ran successfully    |
| `failed`  | One or more DAGs in the service have failed |
| `unknown` | DAG status could not be determined          |

> A service is `failed` if **any** of its DAGs are `failed` or `unknown`.

All responses include `Content-Type: application/json`.

---

### Mock Data (Current Phase)

Defined in `backend/internal/api/controller/services.go`.

| Service ID      | Category | DAGs                                                                                                              | Status    |
|-----------------|----------|-------------------------------------------------------------------------------------------------------------------|-----------|
| `site-forecast` | Forecast | `site_forecast` → healthy                                                                                         | `healthy` |
| `consumer`      | Consumer | `ecmwf_consumer` → healthy, `metoffice_consumer` → **failed**, `pvlive_consumer` → healthy                       | `failed`  |
| `data-platform` | Platform | `save_to_dp` → healthy                                                                                            | `healthy` |
