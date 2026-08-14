import { HeartbeatLine } from '@/components/dashboard/heartbeat-line'
import { ServiceDagsDialog } from '@/components/dashboard/service-dags-dialog'
import { RunHistory } from '@/components/dashboard/run-history'
import { StatusBadge } from '@/components/dashboard/status-badge'
import { RelativeTime } from '@/components/dashboard/relative-time'
import { EM_DASH, formatSuccessRate } from '@/lib/format'
import { isAttentionStatus, recentRuns } from '@/lib/status'
import { cn } from '@/lib/utils'
import type { Service } from '@/lib/types'

/** Why a service has no upcoming run — only reached when next_run_at is null. */
function unscheduledLabel(service: Service): string {
  if (service.status === 'paused') return 'Paused'
  if (service.status === 'unknown') return 'No data'
  // Only a failing service has a *blocked* next run. Anything else without a
  // scheduled run is simply unscheduled — asset- or manually-triggered DAGs
  // legitimately have no next_run_at.
  if (service.status === 'down') return `${EM_DASH} blocked`
  return 'Not scheduled'
}

export function ServiceCard({ service, className }: { service: Service; className?: string }) {
  const { metrics } = service
  const successRate = formatSuccessRate(metrics.success_rate)
  // A service with a heartbeat but no DAGs — the UI, for one — has no run
  // history, schedule or success rate to show. Rendering those rows as dashes
  // implies missing data; the heartbeat is the whole story for these.
  const heartbeatOnly = service.heartbeat !== null && service.dags.length === 0
  // A planned service has no DAGs and never has run — there is no schedule or
  // run history to show, just the "coming soon" badge and reason.
  const noScheduleOrHistory = heartbeatOnly || service.status === 'planned'

  return (
    <article
      className={cn(
        'flex flex-col gap-3 rounded-xl border border-black/8 bg-white p-5 shadow-[0_1px_2px_rgba(0,0,0,0.04),0_8px_24px_-12px_rgba(0,0,0,0.12)]',
        className,
      )}
    >
      <header className="flex items-start justify-between gap-3">
        <h3 className="font-display text-xl leading-tight font-medium text-ink">{service.name}</h3>
        <StatusBadge status={service.status} />
      </header>

      {/* Sits directly under the badge it explains: a red card that doesn't say
          which DAG failed sends the operator digging through the modal. */}
      {service.status_reason && (
        <p
          className={cn(
            'text-sm',
            isAttentionStatus(service.status) ? 'text-flame-text' : 'text-black/55',
          )}
        >
          {service.status_reason}
        </p>
      )}

      {!noScheduleOrHistory && (
        <>
          <p className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-black/55">
            <span>
              Last{' '}
              <RelativeTime
                iso={metrics.last_run_at}
                fallback="No data"
                className="text-black/75"
              />
            </span>
            <span>
              Next{' '}
              <RelativeTime
                iso={metrics.next_run_at}
                fallback={unscheduledLabel(service)}
                className="text-black/75"
              />
            </span>
          </p>

          <p className="text-sm">
            {successRate === null ? (
              <span className="text-black/45">
                {EM_DASH} ({service.note ?? 'No monitoring data'})
              </span>
            ) : (
              <>
                <span className="font-medium text-ink">{successRate}% success</span>{' '}
                <span className="text-black/45">
                  ({metrics.failed_runs} {metrics.failed_runs === 1 ? 'error' : 'errors'} /{' '}
                  {metrics.total_runs} runs)
                </span>
              </>
            )}
          </p>
        </>
      )}

      <HeartbeatLine heartbeat={service.heartbeat} />

      {!noScheduleOrHistory && (
        <RunHistory
          runs={recentRuns(service)}
          muted={service.status === 'paused' || service.status === 'unknown'}
        />
      )}

      <ServiceDagsDialog service={service} />
    </article>
  )
}
