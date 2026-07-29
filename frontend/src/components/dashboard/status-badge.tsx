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
        paused: 'border-transparent bg-black/6 text-black/55',
        unknown: 'border-transparent bg-black/6 text-black/55',
      },
    },
    defaultVariants: { status: 'unknown' },
  },
)

const dotVariants = cva('size-1.5 rounded-full', {
  variants: {
    status: {
      healthy: 'bg-ink',
      degraded: 'bg-flame',
      down: 'hidden',
      paused: 'bg-black/35',
      unknown: 'bg-black/35',
    },
  },
  defaultVariants: { status: 'unknown' },
})

interface StatusBadgeProps extends VariantProps<typeof badgeVariants> {
  status: ServiceStatus
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  return (
    <span className={cn(badgeVariants({ status }), className)}>
      <span className={dotVariants({ status })} aria-hidden />
      {STATUS_LABELS[status]}
    </span>
  )
}

export function StatusDot({ status, className }: { status: ServiceStatus; className?: string }) {
  return (
    <span
      className={cn('size-1.5 rounded-full', className, {
        'bg-ink': status === 'healthy',
        'bg-flame': status === 'degraded' || status === 'down',
        'bg-black/35': status === 'paused' || status === 'unknown',
      })}
      aria-hidden
    />
  )
}
