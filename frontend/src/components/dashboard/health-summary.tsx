import { StatusDot } from '@/components/dashboard/status-badge'
import { SERVICE_STATUSES, STATUS_LABELS, countByStatus, isAttentionStatus } from '@/lib/status'
import { cn } from '@/lib/utils'
import type { Service } from '@/lib/types'

/**
 * Single-line fleet roll-up. Tinted only when something actually needs
 * attention, so the bar stays quiet on a good day.
 */
export function HealthSummary({ services }: { services: Service[] }) {
  const counts = countByStatus(services)
  const present = SERVICE_STATUSES.filter((status) => counts[status] > 0)
  const needsAttention = services.some((service) => isAttentionStatus(service.status))

  return (
    <div
      className={cn(
        'border-b px-6 py-3 lg:px-10',
        needsAttention ? 'border-flame/20 bg-flame-soft' : 'border-black/8 bg-white',
      )}
    >
      {present.length === 0 ? (
        <p className="text-sm text-black/55">No services match the current filters.</p>
      ) : (
        <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
          <StatusDot status={needsAttention ? 'down' : 'healthy'} className="shrink-0" />
          {present.map((status, index) => (
            <span key={status} className="flex items-center gap-2">
              {index > 0 && <span className="text-black/30">·</span>}
              <span className={cn(isAttentionStatus(status) ? 'text-flame-text' : 'text-black/65')}>
                {counts[status]} {STATUS_LABELS[status].toLowerCase()}
              </span>
            </span>
          ))}
        </p>
      )}
    </div>
  )
}
