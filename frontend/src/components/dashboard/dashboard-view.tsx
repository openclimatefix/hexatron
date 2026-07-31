'use client'

import { useMemo } from 'react'

import { HealthSummary } from '@/components/dashboard/health-summary'
import { ServiceGraph } from '@/components/dashboard/service-graph'
import { useDashboardFilters } from '@/components/dashboard/filter-context'
import type { Service } from '@/lib/types'

/**
 * Filters the fleet from the header's search box. Matches service name,
 * category and DAG id, because the box is labelled "Search DAGs…".
 *
 * Note the backend's documented `search` param is name-only; widening it is
 * flagged in the README. Until that lands, this must stay client-side or
 * DAG-id queries will come back empty.
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
