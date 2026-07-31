import type { Dag, DagRun, DagRunState, Metrics, Service, ServiceStatus } from '@/lib/types'

/**
 * Defensive parsing of API responses. The backend is expected to send the
 * agreed shape; this only guards against fields being absent or unrecognised
 * mid-development, so one bad value degrades a single card instead of blanking
 * the dashboard. It does no field renaming — what arrives should already match
 * `@/lib/types`.
 */

const SERVICE_STATUSES: ServiceStatus[] = ['healthy', 'degraded', 'down', 'paused', 'unknown']
const RUN_STATES: DagRunState[] = ['queued', 'running', 'success', 'failed']

/** Anything unrecognised becomes `unknown` rather than breaking the badge. */
export function toServiceStatus(value: unknown): ServiceStatus {
  if (typeof value === 'string') {
    const lower = value.toLowerCase() as ServiceStatus
    if (SERVICE_STATUSES.includes(lower)) return lower
  }
  return 'unknown'
}

function toRunState(value: unknown): DagRunState {
  if (typeof value === 'string') {
    const lower = value.toLowerCase() as DagRunState
    if (RUN_STATES.includes(lower)) return lower
  }
  return 'queued'
}

function str(value: unknown): string | null {
  return typeof value === 'string' && value.length > 0 ? value : null
}

function num(value: unknown): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

function count(value: unknown): number {
  return num(value) ?? 0
}

function arr(value: unknown): unknown[] {
  return Array.isArray(value) ? value : []
}

function normaliseRun(raw: unknown, fallbackDagId: string): DagRun | null {
  if (typeof raw !== 'object' || raw === null) return null
  const r = raw as Record<string, unknown>
  const dagId = str(r.dag_id) ?? fallbackDagId
  const logical = str(r.logical_date)

  return {
    dag_run_id: str(r.dag_run_id) ?? `${dagId}__${logical ?? 'unknown'}`,
    dag_id: dagId,
    logical_date: logical ?? '',
    start_date: str(r.start_date),
    end_date: str(r.end_date),
    state: toRunState(r.state),
    run_type: (str(r.run_type) as DagRun['run_type']) ?? 'scheduled',
  }
}

function normaliseDag(raw: unknown): Dag | null {
  if (typeof raw !== 'object' || raw === null) return null
  const d = raw as Record<string, unknown>
  const dagId = str(d.dag_id)
  if (!dagId) return null

  return {
    dag_id: dagId,
    dag_display_name: str(d.dag_display_name) ?? dagId,
    is_paused: d.is_paused === true,
    timetable_summary: str(d.timetable_summary),
    next_dagrun_logical_date: str(d.next_dagrun_logical_date),
    status: toServiceStatus(d.status),
    runs: arr(d.runs)
      .map((run) => normaliseRun(run, dagId))
      .filter((run): run is DagRun => run !== null),
    metrics: normaliseMetrics(d.metrics),
  }
}

/** Missing metrics are normal for unmonitored DAGs — the UI renders dashes. */
function normaliseMetrics(raw: unknown): Metrics {
  const m = (typeof raw === 'object' && raw !== null ? raw : {}) as Record<string, unknown>
  return {
    total_runs: count(m.total_runs),
    failed_runs: count(m.failed_runs),
    success_rate: num(m.success_rate),
    avg_duration_seconds: num(m.avg_duration_seconds),
    latest_duration_seconds: num(m.latest_duration_seconds),
    last_run_at: str(m.last_run_at),
    next_run_at: str(m.next_run_at),
  }
}

const EMPTY_METRICS: Metrics = {
  total_runs: 0,
  failed_runs: 0,
  success_rate: null,
  avg_duration_seconds: null,
  latest_duration_seconds: null,
  last_run_at: null,
  next_run_at: null,
}

/** Earliest non-null, for picking the soonest upcoming run. */
function earliest(values: (string | null)[]): string | null {
  const times = values.filter((v): v is string => v !== null)
  if (times.length === 0) return null
  return times.reduce((a, b) => (Date.parse(b) < Date.parse(a) ? b : a))
}

/** Latest non-null, for picking the most recent run. */
function latest(values: (string | null)[]): string | null {
  const times = values.filter((v): v is string => v !== null)
  if (times.length === 0) return null
  return times.reduce((a, b) => (Date.parse(b) > Date.parse(a) ? b : a))
}

/**
 * Rolls DAG metrics up to the service. Run counts sum, so the success rate is
 * recomputed from the totals rather than averaging rates — otherwise a DAG with
 * three runs would weigh as heavily as one with three hundred. Duration is
 * likewise weighted by run count.
 */
export function aggregateMetrics(dags: Dag[]): Metrics {
  if (dags.length === 0) return EMPTY_METRICS

  const metrics = dags.map((dag) => dag.metrics)
  const totalRuns = metrics.reduce((sum, m) => sum + m.total_runs, 0)
  const failedRuns = metrics.reduce((sum, m) => sum + m.failed_runs, 0)

  const weighted = metrics.filter((m) => m.avg_duration_seconds !== null && m.total_runs > 0)
  const weightTotal = weighted.reduce((sum, m) => sum + m.total_runs, 0)
  const avgDuration =
    weightTotal > 0
      ? weighted.reduce((sum, m) => sum + m.avg_duration_seconds! * m.total_runs, 0) / weightTotal
      : null

  // "Latest" should describe the service's most recent run, so take it from
  // whichever DAG ran last rather than blending across them.
  const lastRunAt = latest(metrics.map((m) => m.last_run_at))
  const newest = metrics.find((m) => m.last_run_at !== null && m.last_run_at === lastRunAt)

  return {
    total_runs: totalRuns,
    failed_runs: failedRuns,
    success_rate: totalRuns > 0 ? (totalRuns - failedRuns) / totalRuns : null,
    avg_duration_seconds: avgDuration,
    latest_duration_seconds: newest?.latest_duration_seconds ?? null,
    last_run_at: lastRunAt,
    next_run_at: earliest(metrics.map((m) => m.next_run_at)),
  }
}

export function normaliseService(raw: unknown): Service | null {
  if (typeof raw !== 'object' || raw === null) return null
  const s = raw as Record<string, unknown>
  const id = str(s.id)
  if (!id) return null

  const dags = arr(s.dags)
    .map(normaliseDag)
    .filter((dag): dag is Dag => dag !== null)

  return {
    id,
    name: str(s.name) ?? id,
    category: str(s.category) ?? 'Uncategorised',
    status: toServiceStatus(s.status),
    depends_on: arr(s.depends_on).filter((d): d is string => typeof d === 'string'),
    dags,
    // Falls back to a service-level block only while one is still sent and the
    // service has no DAGs of its own to roll up.
    metrics: dags.length > 0 ? aggregateMetrics(dags) : normaliseMetrics(s.metrics),
    note: str(s.note),
  }
}

export function normaliseServices(raw: unknown): Service[] {
  return arr(raw)
    .map(normaliseService)
    .filter((service): service is Service => service !== null)
}
