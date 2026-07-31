import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/lib/utils'
import { STATUS_LABELS } from '@/lib/status'
import type { ServiceStatus } from '@/lib/types'

const badgeVariants = cva(
  'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium whitespace-nowrap',
  {
    variants: {
      status: {
        healthy: 'border-black/12 bg-white text-ink',
        degraded: 'border-flame/50 bg-white text-flame-text',
        down: 'border-transparent bg-flame text-white',
        running: 'border-black/12 bg-white text-black/65',
        paused: 'border-transparent bg-black/6 text-black/55',
        unknown: 'border-transparent bg-black/6 text-black/55',
      },
    },
    defaultVariants: { status: 'unknown' },
  },
)

/** Single source of dot colour, shared by the badge and the standalone dot. */
const DOT_COLORS: Record<ServiceStatus, string> = {
  healthy: 'bg-ink',
  degraded: 'bg-flame',
  down: 'bg-flame',
  // Hollow ring: in-flight, not yet an outcome.
  running: 'bg-transparent ring-2 ring-inset ring-black/45',
  paused: 'bg-black/35',
  unknown: 'bg-black/35',
}

interface StatusBadgeProps extends VariantProps<typeof badgeVariants> {
  status: ServiceStatus
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  return (
    <span className={cn(badgeVariants({ status }), className)}>
      {/* The `down` badge is a solid fill, so a dot on top would be redundant. */}
      {status !== 'down' && <StatusDot status={status} />}
      {STATUS_LABELS[status]}
    </span>
  )
}

export function StatusDot({ status, className }: { status: ServiceStatus; className?: string }) {
  return <span className={cn('size-1.5 rounded-full', DOT_COLORS[status], className)} aria-hidden />
}
