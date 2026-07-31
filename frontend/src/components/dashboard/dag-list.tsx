import { RunHistory } from '@/components/dashboard/run-history'
import { StatusBadge } from '@/components/dashboard/status-badge'
import { EM_DASH, formatDuration, formatSuccessRate } from '@/lib/format'
import type { Dag } from '@/lib/types'

/** Shared by the dashboard modal and the deep-linkable service route. */
export function DagList({ dags }: { dags: Dag[] }) {
  if (dags.length === 0) {
    return (
      <p className="rounded-xl border border-dashed border-black/12 bg-white/50 p-6 text-sm text-black/55">
        No DAGs are mapped to this service yet.
      </p>
    )
  }

  return (
    <ul className="flex flex-col gap-3">
      {dags.map((dag) => {
        const rate = formatSuccessRate(dag.metrics.success_rate)
        return (
          <li
            key={dag.dag_id}
            className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-black/8 bg-white p-4"
          >
            <div className="min-w-0">
              <p className="truncate font-medium text-ink">{dag.dag_display_name}</p>
              <p className="truncate text-xs text-black/45">{dag.dag_id}</p>
              <p className="mt-1 text-xs text-black/55">
                {rate === null ? (
                  <span className="text-black/40">No runs in window</span>
                ) : (
                  <>
                    <span className="text-black/75">{rate}% success</span>{' '}
                    <span className="text-black/40">
                      ({dag.metrics.failed_runs}{' '}
                      {dag.metrics.failed_runs === 1 ? 'error' : 'errors'} /{' '}
                      {dag.metrics.total_runs} runs)
                    </span>
                  </>
                )}
                <span className="text-black/30"> · </span>
                Avg{' '}
                <span className="text-black/75">
                  {formatDuration(dag.metrics.avg_duration_seconds)}
                </span>
                {dag.metrics.latest_duration_seconds !== null && (
                  <>
                    <span className="text-black/30"> · </span>
                    Latest{' '}
                    <span className="text-black/75">
                      {formatDuration(dag.metrics.latest_duration_seconds)}
                    </span>
                  </>
                )}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-4">
              <span className="text-xs text-black/50">
                {dag.timetable_summary ?? `${EM_DASH} no schedule`}
              </span>
              <RunHistory runs={dag.runs} />
              <StatusBadge status={dag.status} />
            </div>
          </li>
        )
      })}
    </ul>
  )
}
