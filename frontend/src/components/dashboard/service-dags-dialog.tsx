'use client'

import Link from 'next/link'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { DagList } from '@/components/dashboard/dag-list'
import { StatusBadge } from '@/components/dashboard/status-badge'
import { EM_DASH, formatDuration, formatSuccessRate, formatTime } from '@/lib/format'
import type { Service } from '@/lib/types'

/**
 * Opens the service's DAGs in place so the dashboard keeps its filter state and
 * the graph doesn't have to re-measure on the way back. The full page at
 * /services/[id] stays available for deep links and sharing.
 */
export function ServiceDagsDialog({ service }: { service: Service }) {
  const successRate = formatSuccessRate(service.metrics.success_rate)

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button size="lg" className="mt-1 w-fit rounded-lg px-4">
          View DAGs <span aria-hidden>→</span>
          <span className="sr-only">for {service.name}</span>
        </Button>
      </DialogTrigger>

      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex flex-wrap items-center gap-3">
            <span className="font-display text-2xl font-medium tracking-tight">{service.name}</span>
            <StatusBadge status={service.status} />
          </DialogTitle>
          <DialogDescription>
            {service.note ?? `${service.category} · ${service.dags.length} DAGs`}
          </DialogDescription>
        </DialogHeader>

        <dl className="grid grid-cols-2 gap-x-6 gap-y-4 rounded-xl border border-black/8 bg-canvas p-4 sm:grid-cols-4">
          <Stat label="Success rate" value={successRate ? `${successRate}%` : EM_DASH} />
          <Stat
            label="Runs"
            value={service.metrics.total_runs > 0 ? `${service.metrics.total_runs}` : EM_DASH}
          />
          <Stat label="Avg duration" value={formatDuration(service.metrics.avg_duration_seconds)} />
          <Stat label="Last run" value={formatTime(service.metrics.last_run_at)} />
        </dl>

        <DagList dags={service.dags} />

        <Link
          href={`/services/${service.id}`}
          className="text-sm text-black/50 underline-offset-4 transition-colors hover:text-black/75 hover:underline"
        >
          Open full page <span aria-hidden>→</span>
        </Link>
      </DialogContent>
    </Dialog>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs text-black/45">{label}</dt>
      <dd className="mt-0.5 text-base font-medium text-ink">{value}</dd>
    </div>
  )
}
