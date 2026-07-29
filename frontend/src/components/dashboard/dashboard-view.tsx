'use client'

import { useMemo } from 'react'

import { HealthSummary } from '@/components/dashboard/health-summary'
import { ServiceGraph } from '@/components/dashboard/service-graph'
import { useDashboardFilters } from '@/components/dashboard/filter-context'
import type { Service } from '@/lib/types'

/**
 * Filters the fleet from the header's search box. Matching on service name,
 * category or DAG id mirrors the backend's `search` param, so behaviour won't
 * shift when filtering moves server-side.
 */
export function DashboardView({ services }: { services: Service[] }) {
  const { search } = useDashboardFilters()

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    if (!term) return services
    return services.filter(
      (service) =>
        service.name.toLowerCase().includes(term) ||
        service.category.toLowerCase().includes(term) ||
        service.dags.some((dag) => dag.dag_id.toLowerCase().includes(term)),
    )
  }, [search, services])

  return (
    <>
      <HealthSummary services={filtered} />

      <div className="flex-1 bg-canvas px-6 py-14 lg:px-10">
        {filtered.length === 0 ? (
          <p className="py-20 text-center text-sm text-black/55">No services match “{search}”.</p>
        ) : (
          <div className="mx-auto max-w-6xl">
            <ServiceGraph services={filtered} />
          </div>
        )}
      </div>
    </>
  )
}
