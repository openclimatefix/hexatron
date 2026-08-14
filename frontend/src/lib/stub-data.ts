import { aggregateMetrics } from '@/lib/normalise'
import { deriveServiceStatus } from '@/lib/status'
import type { Dag, DagRun, DagRunState, Service } from '@/lib/types'

/**
 * Stub data standing in for the Airflow-backed API until example DAG payloads
 * are available. Shapes match `@/lib/types`, so wiring up the real backend is a
 * swap inside `@/lib/api` and nothing here leaks into components.
 *
 * Timestamps are anchored to a fixed instant rather than `Date.now()` so
 * renders are deterministic across server and client.
 */
const NOW = Date.parse('2026-07-29T10:15:00Z')
const MINUTE = 60_000

function isoAt(offsetMinutes: number): string {
  return new Date(NOW + offsetMinutes * MINUTE).toISOString()
}

/** Deterministic jitter in [-1, 1], keyed by dag and index. */
function jitter(seed: string, index: number): number {
  let hash = 0
  const key = `${seed}:${index}`
  for (let i = 0; i < key.length; i += 1) {
    hash = (hash << 5) - hash + key.charCodeAt(i)
    hash |= 0
  }
  return ((hash % 1000) / 1000) * 2 - 1
}

const STATE_BY_CODE: Record<string, DagRunState> = {
  S: 'success',
  F: 'failed',
  R: 'running',
}

interface SeriesOptions {
  /** Minutes relative to NOW for the most recent run's start. Negative = past. */
  lastRunOffset: number
  /** Gap between consecutive runs, in minutes. */
  intervalMinutes: number
  /** Typical successful run duration, in seconds. */
  baseDurationSeconds: number
  /** Overrides the most recent run's duration, for the "latest vs avg" trend. */
  latestDurationSeconds?: number | null
}

/**
 * Builds a run history from a compact pattern, most recent first.
 * 'S' success, 'F' failed, 'R' running.
 */
function series(dagId: string, pattern: string, options: SeriesOptions): DagRun[] {
  const { lastRunOffset, intervalMinutes, baseDurationSeconds, latestDurationSeconds } = options

  return [...pattern].map((code, index) => {
    const state = STATE_BY_CODE[code] ?? 'success'
    const startOffset = lastRunOffset - index * intervalMinutes
    const start = isoAt(startOffset)

    let durationSeconds: number | null
    if (index === 0 && latestDurationSeconds !== undefined) {
      durationSeconds = latestDurationSeconds
    } else if (state === 'running') {
      durationSeconds = null
    } else if (state === 'failed') {
      // Failures usually bail out early.
      durationSeconds = Math.round(baseDurationSeconds * 0.4)
    } else {
      durationSeconds = Math.round(baseDurationSeconds * (1 + jitter(dagId, index) * 0.15))
    }

    return {
      dag_run_id: `${dagId}__${start}`,
      dag_id: dagId,
      logical_date: start,
      start_date: start,
      end_date:
        durationSeconds === null
          ? null
          : new Date(Date.parse(start) + durationSeconds * 1000).toISOString(),
      state,
      run_type: 'scheduled' as const,
    }
  })
}

interface DagOptions extends Omit<Partial<Dag>, 'status' | 'metrics'> {
  /**
   * Totals over the full metrics window. `runs` only holds the last ten for the
   * history strip, so a DAG whose window is healthy can still show a failure in
   * the strip — pass the window figures to keep the badge and numbers honest.
   */
  totalRuns?: number
  failedRuns?: number
  avgDurationSeconds?: number
  latestDurationSeconds?: number | null
  nextRunAt?: string | null
}

function dag(dagId: string, displayName: string, runs: DagRun[], options: DagOptions = {}): Dag {
  const {
    totalRuns,
    failedRuns,
    avgDurationSeconds,
    latestDurationSeconds,
    nextRunAt,
    ...overrides
  } = options

  // Default the window to the visible runs when no wider figures are given.
  const total = totalRuns ?? runs.length
  const failed = failedRuns ?? runs.filter((run) => run.state === 'failed').length
  const successRate = total > 0 ? (total - failed) / total : null

  const durations = runs
    .map((run) =>
      run.start_date && run.end_date
        ? (Date.parse(run.end_date) - Date.parse(run.start_date)) / 1000
        : null,
    )
    .filter((d): d is number => d !== null)
  const observedAvg =
    durations.length > 0 ? durations.reduce((a, b) => a + b, 0) / durations.length : null

  const isPaused = overrides.is_paused ?? false
  const nextRun = nextRunAt ?? overrides.next_dagrun_logical_date ?? null

  return {
    dag_id: dagId,
    dag_display_name: displayName,
    is_paused: false,
    timetable_summary: 'hourly',
    next_dagrun_logical_date: null,
    airflow_url: null,
    runs,
    ...overrides,
    // Derived rather than hand-set, so changing a run pattern can't leave the
    // badge disagreeing with the history strip beside it.
    status: deriveServiceStatus({
      isPaused,
      latestRunState: runs[0]?.state ?? null,
      successRate,
    }),
    metrics: {
      total_runs: total,
      failed_runs: failed,
      success_rate: successRate,
      avg_duration_seconds: avgDurationSeconds ?? observedAvg,
      latest_duration_seconds:
        latestDurationSeconds !== undefined ? latestDurationSeconds : (durations[0] ?? null),
      last_run_at: runs[0]?.start_date ?? null,
      next_run_at: nextRun,
    },
  }
}

/**
 * Service metrics roll up from the DAGs, exactly as they do for live data.
 * `heartbeat` is optional here because most services have no health_check;
 * omitting it is the common case, not an oversight.
 */
function service(
  base: Omit<Service, 'metrics' | 'heartbeat' | 'status_reason'> &
    Partial<Pick<Service, 'heartbeat' | 'status_reason'>>,
): Service {
  return { heartbeat: null, status_reason: null, ...base, metrics: aggregateMetrics(base.dags) }
}

export const STUB_SERVICES: Service[] = [
  // ---- Top row -----------------------------------------------------------
  service({
    id: 'solar-forecast',
    name: 'Solar Forecast',
    category: 'Forecast',
    status: 'degraded',
    depends_on: [],
    note: null,
    dags: [
      dag(
        'solar_forecast_national',
        'Solar Forecast (National)',
        series('solar_forecast_national', 'SSFSSFSSSS', {
          lastRunOffset: -10,
          intervalMinutes: 60,
          baseDurationSeconds: 270,
          latestDurationSeconds: 430,
        }),
        {
          next_dagrun_logical_date: isoAt(45),
          nextRunAt: isoAt(45),
          totalRuns: 40,
          failedRuns: 4,
          avgDurationSeconds: 270,
          latestDurationSeconds: 430,
        },
      ),
      dag(
        'solar_forecast_sites',
        'Solar Forecast (Sites)',
        series('solar_forecast_sites', 'SSSSFSSSSS', {
          lastRunOffset: -12,
          intervalMinutes: 60,
          baseDurationSeconds: 240,
        }),
        {
          next_dagrun_logical_date: isoAt(45),
          nextRunAt: isoAt(45),
          totalRuns: 35,
          failedRuns: 2,
          avgDurationSeconds: 240,
        },
      ),
    ],
  }),
  service({
    id: 'consumer',
    name: 'Consumers',
    category: 'Consumer',
    status: 'healthy',
    depends_on: [],
    note: null,
    dags: [
      dag(
        'ecmwf_consumer',
        'ECMWF Consumer',
        series('ecmwf_consumer', 'SSSSSSSSSS', {
          lastRunOffset: -17,
          intervalMinutes: 180,
          baseDurationSeconds: 150,
        }),
        {
          next_dagrun_logical_date: isoAt(15),
          nextRunAt: isoAt(15),
          timetable_summary: 'every 3h',
          totalRuns: 60,
          failedRuns: 0,
          avgDurationSeconds: 150,
          latestDurationSeconds: 115,
        },
      ),
      dag(
        'metoffice_consumer',
        'Met Office Consumer',
        series('metoffice_consumer', 'SSSSFSSSSS', {
          lastRunOffset: -22,
          intervalMinutes: 180,
          baseDurationSeconds: 130,
        }),
        {
          next_dagrun_logical_date: isoAt(18),
          nextRunAt: isoAt(18),
          timetable_summary: 'every 3h',
          totalRuns: 60,
          failedRuns: 1,
          avgDurationSeconds: 130,
        },
      ),
      dag(
        'pvlive_consumer',
        'PVLive Consumer',
        series('pvlive_consumer', 'SSSSSSSSSS', {
          lastRunOffset: -27,
          intervalMinutes: 30,
          baseDurationSeconds: 95,
          latestDurationSeconds: 115,
        }),
        {
          next_dagrun_logical_date: isoAt(15),
          nextRunAt: isoAt(15),
          timetable_summary: 'every 30m',
          totalRuns: 60,
          failedRuns: 0,
          avgDurationSeconds: 95,
        },
      ),
    ],
  }),
  service({
    id: 'ui',
    name: 'UI',
    category: 'Application',
    status: 'unknown',
    depends_on: ['api'],
    note: 'No monitoring data',
    dags: [],
  }),

  // ---- Bottom row --------------------------------------------------------
  service({
    id: 'wind-forecast',
    name: 'Wind Forecast',
    category: 'Forecast',
    status: 'planned',
    depends_on: [],
    note: null,
    dags: [],
  }),
  service({
    id: 'data-platform',
    name: 'Data Platform',
    category: 'Platform',
    status: 'healthy',
    depends_on: ['solar-forecast', 'wind-forecast', 'consumer'],
    note: null,
    dags: [
      dag(
        'save_to_dp',
        'Save to Data Platform',
        series('save_to_dp', 'SSSSSFSSSS', {
          lastRunOffset: -3,
          intervalMinutes: 15,
          baseDurationSeconds: 90,
          latestDurationSeconds: 88,
        }),
        {
          next_dagrun_logical_date: isoAt(12),
          nextRunAt: isoAt(12),
          timetable_summary: 'every 15m',
          totalRuns: 210,
          failedRuns: 1,
          avgDurationSeconds: 90,
          latestDurationSeconds: 88,
        },
      ),
    ],
  }),
  service({
    id: 'api',
    name: 'API',
    category: 'Application',
    status: 'paused',
    depends_on: ['data-platform'],
    note: 'Paused for maintenance',
    dags: [
      dag(
        'api_healthcheck',
        'API Healthcheck',
        series('api_healthcheck', 'SSSSSSSSSS', {
          lastRunOffset: -75,
          intervalMinutes: 30,
          baseDurationSeconds: 40,
        }),
        {
          is_paused: true,
          next_dagrun_logical_date: null,
          nextRunAt: null,
          timetable_summary: 'every 30m',
          // Paused: no window to measure, so the card shows the note instead.
          totalRuns: 0,
          failedRuns: 0,
          avgDurationSeconds: undefined,
          latestDurationSeconds: null,
        },
      ),
    ],
  }),
]
