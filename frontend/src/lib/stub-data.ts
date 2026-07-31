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

interface DagOptions extends Omit<Partial<Dag>, 'status'> {
  /**
   * Success rate over the full metrics window. `runs` only holds the last ten
   * for the history strip, so a DAG whose window is healthy can still show a
   * failure in the strip — pass the real rate to keep the badge honest.
   */
  successRate?: number
}

function dag(dagId: string, displayName: string, runs: DagRun[], options: DagOptions = {}): Dag {
  const { successRate, ...overrides } = options
  const successes = runs.filter((run) => run.state === 'success').length
  const observedRate = runs.length > 0 ? successes / runs.length : null

  return {
    dag_id: dagId,
    dag_display_name: displayName,
    is_paused: false,
    timetable_summary: 'hourly',
    next_dagrun_logical_date: null,
    runs,
    ...overrides,
    // Derived rather than hand-set, so changing a run pattern can't leave the
    // badge disagreeing with the history strip beside it.
    status: deriveServiceStatus({
      isPaused: overrides.is_paused ?? false,
      latestRunState: runs[0]?.state ?? null,
      successRate: successRate ?? observedRate,
    }),
  }
}

export const STUB_SERVICES: Service[] = [
  // ---- Top row -----------------------------------------------------------
  {
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
        { next_dagrun_logical_date: isoAt(45), successRate: 0.92 },
      ),
      dag(
        'solar_forecast_sites',
        'Solar Forecast (Sites)',
        series('solar_forecast_sites', 'SSSSFSSSSS', {
          lastRunOffset: -12,
          intervalMinutes: 60,
          baseDurationSeconds: 240,
        }),
        { next_dagrun_logical_date: isoAt(45), successRate: 0.92 },
      ),
    ],
    metrics: {
      total_runs: 75,
      failed_runs: 6,
      success_rate: 0.92,
      avg_duration_seconds: 270,
      latest_duration_seconds: 430,
      last_run_at: isoAt(-10),
      next_run_at: isoAt(45),
    },
  },
  {
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
        { next_dagrun_logical_date: isoAt(15), timetable_summary: 'every 3h' },
      ),
      dag(
        'metoffice_consumer',
        'Met Office Consumer',
        series('metoffice_consumer', 'SSSSFSSSSS', {
          lastRunOffset: -22,
          intervalMinutes: 180,
          baseDurationSeconds: 130,
        }),
        { next_dagrun_logical_date: isoAt(18), timetable_summary: 'every 3h', successRate: 0.99 },
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
        { next_dagrun_logical_date: isoAt(15), timetable_summary: 'every 30m' },
      ),
    ],
    metrics: {
      total_runs: 180,
      failed_runs: 1,
      success_rate: 0.99,
      avg_duration_seconds: 130,
      latest_duration_seconds: 115,
      last_run_at: isoAt(-17),
      next_run_at: isoAt(15),
    },
  },
  {
    id: 'ui',
    name: 'UI',
    category: 'Application',
    status: 'unknown',
    depends_on: ['api'],
    note: 'No monitoring data',
    dags: [],
    metrics: {
      total_runs: 0,
      failed_runs: 0,
      success_rate: null,
      avg_duration_seconds: null,
      latest_duration_seconds: null,
      last_run_at: null,
      next_run_at: null,
    },
  },

  // ---- Bottom row --------------------------------------------------------
  {
    id: 'wind-forecast',
    name: 'Wind Forecast',
    category: 'Forecast',
    status: 'down',
    depends_on: [],
    note: 'Upstream feed unavailable — next run blocked',
    dags: [
      dag(
        'wind_forecast_national',
        'Wind Forecast (National)',
        series('wind_forecast_national', 'FFSFFFFSSS', {
          lastRunOffset: -95,
          intervalMinutes: 60,
          baseDurationSeconds: 230,
          latestDurationSeconds: null,
        }),
        { next_dagrun_logical_date: null, successRate: 0.61 },
      ),
    ],
    metrics: {
      total_runs: 46,
      failed_runs: 18,
      success_rate: 0.61,
      avg_duration_seconds: 230,
      latest_duration_seconds: null,
      last_run_at: isoAt(-95),
      next_run_at: null,
    },
  },
  {
    id: 'data-platform',
    name: 'DP',
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
          timetable_summary: 'every 15m',
          successRate: 0.995,
        },
      ),
    ],
    metrics: {
      total_runs: 210,
      failed_runs: 1,
      success_rate: 0.995,
      avg_duration_seconds: 90,
      latest_duration_seconds: 88,
      last_run_at: isoAt(-3),
      next_run_at: isoAt(12),
    },
  },
  {
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
          timetable_summary: 'every 30m',
        },
      ),
    ],
    metrics: {
      total_runs: 0,
      failed_runs: 0,
      success_rate: null,
      avg_duration_seconds: null,
      latest_duration_seconds: null,
      last_run_at: isoAt(-75),
      next_run_at: null,
    },
  },
]
