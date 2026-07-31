import { RunHistory } from '@/components/dashboard/run-history'
import { StatusBadge } from '@/components/dashboard/status-badge'
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
      {dags.map((dag) => (
        <li
          key={dag.dag_id}
          className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-black/8 bg-white p-4"
        >
          <div className="min-w-0">
            <p className="truncate font-medium text-ink">{dag.dag_display_name}</p>
            <p className="truncate text-xs text-black/45">{dag.dag_id}</p>
          </div>

          <div className="flex flex-wrap items-center gap-4">
            <span className="text-xs text-black/50">{dag.timetable_summary ?? 'No schedule'}</span>
            <RunHistory runs={dag.runs} />
            <StatusBadge status={dag.status} />
          </div>
        </li>
      ))}
    </ul>
  )
}
