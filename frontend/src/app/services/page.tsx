import Link from 'next/link'

import { StatusBadge } from '@/components/dashboard/status-badge'
import { ServiceLoadError } from '@/components/service-load-error'
import { getServices } from '@/lib/api'
import { EM_DASH, formatSuccessRate, formatTime } from '@/lib/format'
import type { Service } from '@/lib/types'

export const metadata = {
  title: 'Services',
}

/** Live data — see the note in app/page.tsx. */
export const dynamic = 'force-dynamic'

export default async function ServicesPage() {
  let services: Service[]
  try {
    services = await getServices()
  } catch (error) {
    console.error('Failed to load services:', error)
    return <ServiceLoadError detail={error instanceof Error ? error.message : undefined} />
  }

  return (
    <div className="flex-1 bg-canvas px-6 py-10 lg:px-10">
      <div className="mx-auto max-w-4xl">
        <h1 className="font-display text-3xl font-medium tracking-tight text-ink">Services</h1>
        <p className="mt-1 text-sm text-black/55">{services.length} services from the registry.</p>

        <ul className="mt-6 flex flex-col gap-3">
          {services.map((service) => {
            const successRate = formatSuccessRate(service.metrics.success_rate)
            return (
              <li key={service.id}>
                <Link
                  href={`/services/${service.id}`}
                  className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-black/8 bg-white p-4 transition-colors hover:border-black/20"
                >
                  <div className="min-w-0">
                    <p className="font-medium text-ink">{service.name}</p>
                    <p className="text-xs text-black/45">
                      {service.category} · {service.dags.length}{' '}
                      {service.dags.length === 1 ? 'DAG' : 'DAGs'}
                    </p>
                  </div>

                  <div className="flex flex-wrap items-center gap-5 text-sm text-black/55">
                    <span>{successRate ? `${successRate}%` : EM_DASH}</span>
                    <span>{formatTime(service.metrics.last_run_at)}</span>
                    <StatusBadge status={service.status} />
                  </div>
                </Link>
              </li>
            )
          })}
        </ul>
      </div>
    </div>
  )
}
