import type { DagRun, Service, ServiceStatus } from '@/lib/types'

export const SERVICE_STATUSES: ServiceStatus[] = [
  'healthy',
  'degraded',
  'down',
  'running',
  'paused',
  'unknown',
]

export const STATUS_LABELS: Record<ServiceStatus, string> = {
  healthy: 'Healthy',
  degraded: 'Degraded',
  down: 'Down',
  running: 'Running',
  paused: 'Paused',
  unknown: 'Unknown',
}

/** Statuses an operator should act on — drives the alert bar's tone. */
export function isAttentionStatus(status: ServiceStatus): boolean {
  return status === 'degraded' || status === 'down'
}

export function countByStatus(services: Service[]): Record<ServiceStatus, number> {
  const counts: Record<ServiceStatus, number> = {
    healthy: 0,
    degraded: 0,
    down: 0,
    running: 0,
    paused: 0,
    unknown: 0,
  }
  for (const service of services) counts[service.status] += 1
  return counts
}

/** Success rate at or above this counts as healthy. */
export const DEGRADED_THRESHOLD = 0.95

/**
 * Derives service health. The backend will own this once the aggregator lands;
 * it lives here so stub data and any client-side recomputation stay consistent.
 *
 * Deliberately rate-based rather than "any failure degrades it" — a single
 * failure in 180 runs is noise, not a degradation.
 */
export function deriveServiceStatus({
  isPaused = false,
  latestRunState = null,
  successRate = null,
}: {
  isPaused?: boolean
  latestRunState?: DagRun['state'] | null
  successRate?: number | null
}): ServiceStatus {
  if (isPaused) return 'paused'
  if (latestRunState === null || successRate === null) return 'unknown'
  if (latestRunState === 'failed') return 'down'
  if (successRate < DEGRADED_THRESHOLD) return 'degraded'
  return 'healthy'
}

/** Most recent runs across every DAG in a service, newest first. */
export function recentRuns(service: Service, limit = 10): DagRun[] {
  return service.dags
    .flatMap((dag) => dag.runs)
    .sort((a, b) => Date.parse(b.logical_date) - Date.parse(a.logical_date))
    .slice(0, limit)
}
