'use client'

import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import type { Heartbeat } from '@/lib/types'

const TRANSPORT_LABELS: Record<Heartbeat['type'], string> = {
  http: 'HTTP',
  grpc: 'gRPC',
}

const DOT_CLASSES: Record<Heartbeat['status'], string> = {
  // Not the `run-pulse` animation used for in-flight runs: a heartbeat is a
  // steady-state fact, and two different pulses on one card compete.
  healthy: 'bg-ink',
  down: 'bg-flame',
  unknown: 'bg-black/35',
}

function summary(heartbeat: Heartbeat): string {
  switch (heartbeat.status) {
    case 'healthy':
      return heartbeat.latency_ms === null
        ? 'Responding'
        : `Responding · ${Math.round(heartbeat.latency_ms)}ms`
    case 'down':
      return 'Not responding'
    default:
      // The probe never ran, which says nothing about the service — saying
      // "down" here would be a false alarm.
      return 'Not checked'
  }
}

/**
 * The liveness probe behind a service's status, for the ones Airflow cannot
 * speak for. Renders nothing when no health_check is configured, so it can be
 * dropped into every card unconditionally.
 */
export function HeartbeatLine({
  heartbeat,
  className,
}: {
  heartbeat: Heartbeat | null
  className?: string
}) {
  if (!heartbeat) return null

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <p className={cn('flex items-center gap-1.5 text-sm text-black/55', className)}>
          <span
            className={cn('size-1.5 shrink-0 rounded-full', DOT_CLASSES[heartbeat.status])}
            aria-hidden
          />
          <span className={heartbeat.status === 'down' ? 'text-flame-text' : undefined}>
            Heartbeat <span className="text-black/75">{summary(heartbeat)}</span>
          </span>
        </p>
      </TooltipTrigger>
      <TooltipContent side="top" sideOffset={6} className="pointer-events-none flex-col items-start gap-0.5">
        <span className="font-medium">
          {TRANSPORT_LABELS[heartbeat.type]} probe — {summary(heartbeat)}
        </span>
        {/* The target is the single most useful thing when a heartbeat is wrong:
            a probe pointed at a placeholder looks identical to a real outage. */}
        <span className="text-background/60">{heartbeat.target}</span>
        {heartbeat.detail && <span className="text-background/60">{heartbeat.detail}</span>}
      </TooltipContent>
    </Tooltip>
  )
}
