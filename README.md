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

Port-forward Airflow, then grab a `session` cookie from your browser's dev tools
after logging into the Airflow UI:

```bash
cp .env.example .env          # fill in AIRFLOW_SESSION_COOKIE
set -a && . ./.env && set +a  # Go does not read .env itself
cd backend && go run ./cmd/hexatron
```

This currently prints every DAG Airflow knows about:

```
STATE   DAG ID                      SCHEDULE             DESCRIPTION
active  nl-api-check                0 * * * *            General checks on NL API.
paused  uk-consume-pv               */5 * * * *          Dag to download PV generation data.
...
21 DAGs, 20 active
```

The cookie expires; a stale one fails with `401 Unauthorized (session cookie is
missing or expired)`.
