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

Phase 1 — project scaffolding. Packages are stubs; implementation follows in
later phases.

## Getting started

```bash
cp .env.example .env
cd backend && go build ./...
```
