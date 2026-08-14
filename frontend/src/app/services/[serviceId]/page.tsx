import Link from 'next/link'
import { notFound } from 'next/navigation'

import { DagList } from '@/components/dashboard/dag-list'
import { HeartbeatLine } from '@/components/dashboard/heartbeat-line'
import { StatusBadge } from '@/components/dashboard/status-badge'
import { ServiceLoadError } from '@/components/service-load-error'
import { getService } from '@/lib/api'
import { EM_DASH, formatDuration, formatSuccessRate, formatTime } from '@/lib/format'
import type { Service } from '@/lib/types'

/** Live data — see the note in app/page.tsx. */
export const dynamic = 'force-dynamic'

interface PageProps {
  params: Promise<{ serviceId: string }>
}

export async function generateMetadata({ params }: PageProps) {
  const { serviceId } = await params
  try {
    const service = await getService(serviceId)
    return { title: service?.name ?? 'Service not found' }
  } catch {
    // Metadata must not be the thing that takes the page down.
    return { title: 'Service' }
  }
}

export default async function ServiceDetailPage({ params }: PageProps) {
  const { serviceId } = await params

  let service: Service | null
  try {
    service = await getService(serviceId)
  } catch (error) {
    console.error(`Failed to load service ${serviceId}:`, error)
    return <ServiceLoadError detail={error instanceof Error ? error.message : undefined} />
  }

  // Kept outside the try: notFound() signals via a thrown control-flow error
  // that must reach Next, not be swallowed as a load failure.
  if (!service) notFound()

  const successRate = formatSuccessRate(service.metrics.success_rate)

  return (
    <div className="flex-1 bg-canvas px-6 py-10 lg:px-10">
      <div className="mx-auto max-w-4xl">
        <Link href="/" className="text-sm text-black/50 transition-colors hover:text-black/75">
          <span aria-hidden>←</span> Back to dashboard
        </Link>

        <div className="mt-4 flex flex-wrap items-center gap-3">
          <h1 className="font-display text-3xl font-medium tracking-tight text-ink">
            {service.name}
          </h1>
          <StatusBadge status={service.status} />
          <span className="text-sm text-black/45">{service.category}</span>
        </div>

        {service.note && <p className="mt-2 text-sm text-black/55">{service.note}</p>}

        <HeartbeatLine heartbeat={service.heartbeat} className="mt-2" />

        <dl className="mt-6 grid grid-cols-2 gap-x-6 gap-y-4 rounded-xl border border-black/8 bg-white p-5 sm:grid-cols-4">
          <Stat label="Success rate" value={successRate ? `${successRate}%` : EM_DASH} />
          <Stat
            label="Runs"
            value={service.metrics.total_runs > 0 ? `${service.metrics.total_runs}` : EM_DASH}
          />
          <Stat label="Avg duration" value={formatDuration(service.metrics.avg_duration_seconds)} />
          <Stat label="Last run" value={formatTime(service.metrics.last_run_at)} />
        </dl>

        <h2 className="mt-10 font-display text-lg font-medium text-ink">
          DAGs ({service.dags.length})
        </h2>

        <div className="mt-3">
          <DagList dags={service.dags} />
        </div>
      </div>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs text-black/45">{label}</dt>
      <dd className="mt-0.5 text-lg font-medium text-ink">{value}</dd>
    </div>
  )
}
