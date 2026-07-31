/**
 * Types mirror the Airflow REST API where possible so the Go backend can pass
 * DAG/DagRun payloads through with minimal transformation. Fields sourced from
 * Airflow keep Airflow's snake_case names; Hexatron-derived fields are marked.
 */

/** Airflow DagRun.state. */
export type DagRunState = 'queued' | 'running' | 'success' | 'failed'

/** Airflow DagRun.run_type. */
export type DagRunType = 'scheduled' | 'manual' | 'backfill' | 'asset_triggered'

/**
 * Hexatron-derived service health. Wider than the three states in the README
 * contract: `paused` (intentional) and `degraded` (partial failure) are
 * distinct operational conditions that `failed` alone collapses.
 */
export type ServiceStatus = 'healthy' | 'degraded' | 'down' | 'running' | 'paused' | 'unknown'

/** Subset of Airflow's DagRun object that the dashboard renders. */
export interface DagRun {
  dag_run_id: string
  dag_id: string
  logical_date: string
  start_date: string | null
  end_date: string | null
  state: DagRunState
  run_type: DagRunType
}

/**
 * Run aggregates over the selected time window. Reported per DAG — that's the
 * level Airflow actually measures at. All nullable so unmonitored DAGs render
 * cleanly.
 */
export interface Metrics {
  total_runs: number
  failed_runs: number
  /** 0..1, or null when there are no runs to measure. */
  success_rate: number | null
  avg_duration_seconds: number | null
  latest_duration_seconds: number | null
  last_run_at: string | null
  next_run_at: string | null
}

/** Subset of Airflow's DAG object, plus the runs backing its history strip. */
export interface Dag {
  dag_id: string
  dag_display_name: string
  is_paused: boolean
  /** Airflow's human-readable schedule summary, e.g. "hourly". */
  timetable_summary: string | null
  next_dagrun_logical_date: string | null
  /** Hexatron-derived, from the DAG's recent run states. */
  status: ServiceStatus
  /** Most recent first. */
  runs: DagRun[]
  metrics: Metrics
}

export interface Service {
  id: string
  name: string
  category: string
  /** Hexatron-derived, aggregated from `dags`. */
  status: ServiceStatus
  /** Upstream service ids. Drives the dependency graph edges. */
  depends_on: string[]
  dags: Dag[]
  /**
   * Rolled up from `dags` rather than read from the payload — the service-level
   * block is being retired, and summing the DAGs keeps the card consistent with
   * the per-DAG numbers behind it.
   */
  metrics: Metrics
  /** Operator-facing explanation, e.g. "Paused for maintenance". */
  note: string | null
}

export const TIME_RANGES = ['1h', '24h', '7d', '30d'] as const
export type TimeRange = (typeof TIME_RANGES)[number]

export const TIME_RANGE_LABELS: Record<TimeRange, string> = {
  '1h': 'Last 1h',
  '24h': 'Last 24h',
  '7d': 'Last 7d',
  '30d': 'Last 30d',
}
