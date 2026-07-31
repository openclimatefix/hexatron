# Hexatron Architecture

## Overview

Hexatron is a service-centric operational dashboard for the Open Climate Fix platform. It translates Airflow DAG states into business-oriented service health.

## Request Flow

```
Client
  │
  ▼
middleware/cors.go        – CORS headers
middleware/request_id.go  – attach X-Request-ID
middleware/logging.go     – structured request log
middleware/recovery.go    – panic → 500
  │
  ▼
routes/routes.go          – mux pattern matching
  │
  ├── routes/airflow_routes.go   →  controllers/airflow_controller.go
  │                                    │
  │                                    ▼
  │                               services/airflow_service.go  (mock data → Phase 2: Airflow client)
  │
  └── routes/health_routes.go    →  controllers/health_controller.go
                                        │
                                        ▼
                                   services/health_service.go
```

## Package Responsibilities

| Package        | Responsibility                                              |
|----------------|-------------------------------------------------------------|
| `cmd/server`   | Entry point – wires config and router                       |
| `config`       | Load env vars; logger factory                               |
| `constants`    | All string literals – status, headers, endpoints, messages  |
| `routes`       | Register routes; apply middleware stack                     |
| `middleware`   | CORS, logging, recovery, request ID                         |
| `controllers`  | Parse request → call service → write response               |
| `services`     | Business logic; mock data (Phase 1); Airflow calls (Phase 2)|
| `clients`      | Only layer aware of the Airflow REST API                    |
| `models`       | Domain types (Service, DAG, DagRun, TaskInstance)           |
| `structures`   | Request & response payload structs per endpoint             |
| `interfaces`   | Service contracts (enables mocking in tests)                |
| `utils`        | Shared helpers: WriteJSON, WriteError, GenerateRequestID    |

## Phases

| Phase | Deliverable                                         |
|-------|-----------------------------------------------------|
| 1     | Project structure, mock responses, REST API working |
| 2     | Real Airflow integration via `clients/`             |
| 3     | Caching, error handling, logging improvements       |
| 4     | CloudWatch integration (future)                     |
