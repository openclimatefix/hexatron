# Hexatron API Contract

Base URL: `http://localhost:8080`

---

## Endpoints

### 1. List All Services

```
GET /services
```

Returns the health status of all configured business services.

#### Query Parameters

| Parameter  | Type   | Required | Description                                      |
|------------|--------|----------|--------------------------------------------------|
| `search`   | string | No       | Case-insensitive substring filter on service name |
| `category` | string | No       | Exact match on service category                  |

#### Request Payload

```go
// structures.GetServicesRequestPayload
{
  Search:   string  // ?search=
  Category: string  // ?category=
}
```

#### Response Payload

```go
// structures.GetServicesResponsePayload  ([]ServiceSummary)
```

```json
[
  {
    "id":     "site-forecast",
    "name":   "Site Forecast",
    "status": "healthy"
  },
  {
    "id":     "consumer",
    "name":   "Consumer",
    "status": "failed"
  },
  {
    "id":     "data-platform",
    "name":   "Data Platform",
    "status": "healthy"
  }
]
```

#### Status Codes

| Code | Meaning        |
|------|----------------|
| 200  | Success        |

#### Examples

```bash
# All services
curl http://localhost:8080/services

# Search by name
curl "http://localhost:8080/services?search=consumer"

# Filter by category
curl "http://localhost:8080/services?category=Forecast"
```

---

### 2. Get Service Detail

```
GET /services/{serviceId}
```

Returns a single service with the health status of each of its underlying DAGs.

#### Path Parameters

| Parameter   | Type   | Required | Description              |
|-------------|--------|----------|--------------------------|
| `serviceId` | string | Yes      | The ID of the service    |

#### Request Payload

```go
// structures.GetServiceDetailRequestPayload
{
  ServiceID: string  // path param
}
```

#### Response Payload

```go
// structures.GetServiceDetailResponsePayload
```

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

#### Status Codes

| Code | Meaning                     |
|------|-----------------------------|
| 200  | Success                     |
| 404  | Service not found           |

#### 404 Response

```json
{
  "error": "service not found: <serviceId>"
}
```

#### Examples

```bash
# Site Forecast
curl http://localhost:8080/services/site-forecast

# Consumer
curl http://localhost:8080/services/consumer

# Data Platform
curl http://localhost:8080/services/data-platform
```

---

## Status Values

| Value     | Meaning                                      |
|-----------|----------------------------------------------|
| `healthy` | All DAGs in the service ran successfully     |
| `failed`  | One or more DAGs in the service have failed  |
| `unknown` | DAG status could not be determined           |

> A service is `failed` if **any** of its DAGs are `failed` or `unknown`.

---

## Headers

All responses include:

```
Content-Type: application/json
```

---

## Payload Struct Reference

| Struct                            | File                                    | Used By                     |
|-----------------------------------|-----------------------------------------|-----------------------------|
| `GetServicesRequestPayload`       | `internal/structures/service_list.go`   | `GET /services` (request)   |
| `GetServicesResponsePayload`      | `internal/structures/service_list.go`   | `GET /services` (response)  |
| `ServiceSummary`                  | `internal/structures/service_list.go`   | Item in services list       |
| `GetServiceDetailRequestPayload`  | `internal/structures/service_detail.go` | `GET /services/{id}` (request)  |
| `GetServiceDetailResponsePayload` | `internal/structures/service_detail.go` | `GET /services/{id}` (response) |
| `DAGStatus`                       | `internal/structures/service_detail.go` | DAG entry in detail response |

---

## Mock Data (Current Phase)

Mock responses are defined in `internal/api/controller/services.go`.

| Service ID      | Category | DAGs                                                         | Status    |
|-----------------|----------|--------------------------------------------------------------|-----------|
| `site-forecast` | Forecast | `site_forecast` → healthy                                    | `healthy` |
| `consumer`      | Consumer | `ecmwf_consumer` → healthy, `metoffice_consumer` → **failed**, `pvlive_consumer` → healthy | `failed`  |
| `data-platform` | Platform | `save_to_dp` → healthy                                       | `healthy` |

> In a future phase, mock data will be replaced by live calls to the Airflow REST API via `internal/clients/airflow/client.go`.
