'use client'

import { ElapsedTime } from '@/components/dashboard/elapsed-time'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatDuration, formatTime } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { DagRun } from '@/lib/types'

const SLOTS = 10

const STATE_CLASSES: Record<DagRun['state'], string> = {
  success: 'bg-ink',
  failed: 'bg-flame',
  // Solid `active` blue with a ping, matching the running badge — a hollow grey
  // square reads as "old run" rather than "happening right now".
  running: 'bg-active motion-safe:animate-run-pulse',
  queued: 'bg-black/12',
}

const STATE_LABELS: Record<DagRun['state'], string> = {
  success: 'Succeeded',
  failed: 'Failed',
  running: 'Running',
  queued: 'Queued',
}

function runDurationSeconds(run: DagRun): number | null {
  if (!run.start_date || !run.end_date) return null
  const seconds = (Date.parse(run.end_date) - Date.parse(run.start_date)) / 1000
  return Number.isFinite(seconds) && seconds >= 0 ? seconds : null
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
      // The squares themselves aren't focusable — ten tab stops per strip would
      // swamp keyboard navigation — so the whole strip is described here and the
      // tooltips are a pointer affordance on top.
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
      {recent.map((run) => {
        const seconds = runDurationSeconds(run)
        return (
          // dag_run_id is derived from the schedule, so it's only unique within
          // a DAG — two DAGs on the same cron collide once runs are merged here.
          <Tooltip key={`${run.dag_id}__${run.dag_run_id}`}>
            <TooltipTrigger asChild>
              {/* The hit area is padded out by half the gap and pulled back with
                  a negative margin, so neighbouring targets touch without
                  changing the layout. Without it the 4px gap is a dead zone that
                  closes and reopens the tooltip on the way past. */}
              <span className="-mx-0.5 -my-1 flex shrink-0 items-center px-0.5 py-1">
                <span className={cn('size-2.5 rounded-[2px]', STATE_CLASSES[run.state])} />
              </span>
            </TooltipTrigger>
            <TooltipContent
              side="top"
              sideOffset={6}
              // Purely informational, so it must never take the pointer — that
              // is what makes moving along the strip feel sticky.
              className="pointer-events-none flex-col items-start gap-0.5"
            >
              <span className={run.state === 'failed' ? 'font-medium text-flame' : 'font-medium'}>
                {STATE_LABELS[run.state]}
              </span>
              <span className="text-background/60">
                {run.start_date ? formatTime(run.start_date) : 'time unknown'}
                {seconds !== null && ` · took ${formatDuration(seconds)}`}
                {/* Still going: count up from the start rather than showing a
                    duration it doesn't have yet. */}
                {run.state === 'running' && run.start_date && (
                  <>
                    {' · running '}
                    <ElapsedTime since={run.start_date} className="text-background" />
                  </>
                )}
                {run.state === 'running' && !run.start_date && ' · running'}
              </span>
              <span className="text-background/40">{run.dag_id}</span>
            </TooltipContent>
          </Tooltip>
        )
      })}
    </div>
  )
}
