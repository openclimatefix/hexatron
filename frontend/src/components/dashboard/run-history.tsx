import { cn } from '@/lib/utils'
import type { DagRun } from '@/lib/types'

const SLOTS = 10

const STATE_CLASSES: Record<DagRun['state'], string> = {
  success: 'bg-ink',
  failed: 'bg-flame',
  running: 'bg-ink/35',
  queued: 'bg-black/12',
}

const STATE_LABELS: Record<DagRun['state'], string> = {
  success: 'Succeeded',
  failed: 'Failed',
  running: 'Running',
  queued: 'Queued',
}

/**
 * Recent run outcomes, oldest to newest left-to-right. Pads with empty slots so
 * every card's strip is the same width regardless of how much history exists.
 */
export function RunHistory({
  runs,
  muted = false,
  className,
}: {
  runs: DagRun[]
  /** Renders every slot empty — for paused or unmonitored services, where the
   * stored history no longer reflects what's happening. */
  muted?: boolean
  className?: string
}) {
  const recent = muted ? [] : runs.slice(0, SLOTS).reverse()
  const padding = Math.max(0, SLOTS - recent.length)

  return (
    <div
      className={cn('flex items-center gap-1', className)}
      role="img"
      aria-label={
        recent.length === 0
          ? muted
            ? 'No current run data'
            : 'No recent runs'
          : `Last ${recent.length} runs: ${recent
              .map((run) => STATE_LABELS[run.state].toLowerCase())
              .join(', ')}`
      }
    >
      {Array.from({ length: padding }, (_, index) => (
        <span key={`empty-${index}`} className="size-2.5 rounded-[2px] bg-black/8" />
      ))}
      {recent.map((run) => (
        <span
          key={run.dag_run_id}
          className={cn('size-2.5 rounded-[2px]', STATE_CLASSES[run.state])}
        />
      ))}
    </div>
  )
}
