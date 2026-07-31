'use client'

import { useEffect, useMemo, useState } from 'react'

import { HealthSummary } from '@/components/dashboard/health-summary'
import { ServiceGraph } from '@/components/dashboard/service-graph'
import { ScenarioBanner } from '@/components/dashboard/scenario-banner'
import { useDashboardFilters } from '@/components/dashboard/filter-context'
import { orderForGraph } from '@/lib/layout'
import { normaliseServices } from '@/lib/normalise'
import { LIVE, scenarioUrl } from '@/lib/scenarios'
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
  const { search, scenario } = useDashboardFilters()
  // Keyed by scenario so state is derived rather than cleared on switch, and
  // re-selecting a scenario is instant.
  const [loaded, setLoaded] = useState<Record<string, Service[]>>({})
  const [failures, setFailures] = useState<Record<string, string>>({})

  const fixture = scenario === LIVE ? null : (loaded[scenario] ?? null)
  const error = scenario === LIVE ? null : (failures[scenario] ?? null)
  const loading = scenario !== LIVE && fixture === null && error === null

  useEffect(() => {
    if (!loading) return

    // Guards against an earlier fixture landing after a later selection.
    let active = true

    fetch(scenarioUrl(scenario))
      .then((response) => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`)
        return response.json()
      })
      .then((data) => {
        if (active) setLoaded((prev) => ({ ...prev, [scenario]: normaliseServices(data) }))
      })
      .catch((cause: unknown) => {
        if (!active) return
        const message = cause instanceof Error ? cause.message : 'Failed to load scenario'
        setFailures((prev) => ({ ...prev, [scenario]: message }))
      })

    return () => {
      active = false
    }
  }, [scenario, loading])

  const source = fixture ?? services

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    const matched = !term
      ? source
      : source.filter(
          (service) =>
            service.name.toLowerCase().includes(term) ||
            service.category.toLowerCase().includes(term) ||
            service.dags.some((dag) => dag.dag_id.toLowerCase().includes(term)),
        )
    // Ordered here rather than in the API layer: this is a layout concern, and
    // it must survive whatever order the backend returns services in.
    return orderForGraph(matched)
  }, [search, source])

  // Live data stays on screen while a fixture loads, so the graph doesn't flash.
  const showing = loading && fixture === null ? services : filtered

  return (
    <>
      {scenario !== LIVE && <ScenarioBanner scenario={scenario} error={error} loading={loading} />}

      <HealthSummary services={showing} />

      <div className="flex-1 bg-canvas px-6 py-14 lg:px-10">
        {showing.length === 0 ? (
          <p className="py-20 text-center text-sm text-black/55">No services match “{search}”.</p>
        ) : (
          <div className="mx-auto max-w-6xl">
            <ServiceGraph services={showing} />
          </div>
        )}
      </div>
    </>
  )
}
