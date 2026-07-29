import { ServiceDagsDialog } from '@/components/dashboard/service-dags-dialog'
import { RunHistory } from '@/components/dashboard/run-history'
import { StatusBadge } from '@/components/dashboard/status-badge'
import {
  EM_DASH,
  durationTrend,
  formatDuration,
  formatSuccessRate,
  formatTime,
  TREND_GLYPH,
} from '@/lib/format'
import { recentRuns } from '@/lib/status'
import { cn } from '@/lib/utils'
import type { Service } from '@/lib/types'

function lastRunLabel(service: Service): string {
  return service.metrics.last_run_at ? formatTime(service.metrics.last_run_at) : 'No data'
}

function nextRunLabel(service: Service): string {
  if (service.metrics.next_run_at) return formatTime(service.metrics.next_run_at)
  if (service.status === 'paused') return 'Paused'
  if (service.status === 'unknown') return 'No data'
  return `${EM_DASH} blocked`
}

export function ServiceCard({ service, className }: { service: Service; className?: string }) {
  const { metrics } = service
  const successRate = formatSuccessRate(metrics.success_rate)
  const trend = durationTrend(metrics.latest_duration_seconds, metrics.avg_duration_seconds)

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

      <p className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-black/55">
        <span>
          Last <span className="text-black/75">{lastRunLabel(service)}</span>
        </span>
        <span>
          Next <span className="text-black/75">{nextRunLabel(service)}</span>
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

      <RunHistory
        runs={recentRuns(service)}
        muted={service.status === 'paused' || service.status === 'unknown'}
      />

      <p className="text-sm text-black/55">
        Avg <span className="text-black/75">{formatDuration(metrics.avg_duration_seconds)}</span> ·
        Latest{' '}
        <span className="text-black/75">{formatDuration(metrics.latest_duration_seconds)}</span>{' '}
        <span
          className={cn(trend === 'slower' ? 'text-flame' : 'text-black/55')}
          aria-label={trend === 'none' ? undefined : `Latest run ${trend} than average`}
        >
          {TREND_GLYPH[trend]}
        </span>
      </p>

      <ServiceDagsDialog service={service} />
    </article>
  )
}
