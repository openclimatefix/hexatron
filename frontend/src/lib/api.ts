import { STUB_SERVICES } from '@/lib/stub-data'
import type { Service } from '@/lib/types'

/**
 * The seam between the dashboard and the Go backend. While `USE_STUB_DATA` is
 * on, everything is served from `@/lib/stub-data`; flip the env var once the
 * backend's /services endpoints are live and no component needs to change.
 */
const BASE_URL = process.env.API_BASE_URL ?? 'http://localhost:8080'
const USE_STUB_DATA = process.env.USE_STUB_DATA !== 'false'

/** Keeps a slow or down backend from hanging the dashboard render. */
const REVALIDATE_SECONDS = 15

export interface GetServicesOptions {
  search?: string
  category?: string
}

export async function getServices(options: GetServicesOptions = {}): Promise<Service[]> {
  if (USE_STUB_DATA) return filterStub(STUB_SERVICES, options)

  const params = new URLSearchParams()
  if (options.search) params.set('search', options.search)
  if (options.category) params.set('category', options.category)
  const query = params.size > 0 ? `?${params}` : ''

  const response = await fetch(`${BASE_URL}/services${query}`, {
    next: { revalidate: REVALIDATE_SECONDS },
  })
  if (!response.ok) {
    throw new Error(`Failed to load services: ${response.status}`)
  }
  return response.json()
}

export async function getService(serviceId: string): Promise<Service | null> {
  if (USE_STUB_DATA) {
    return STUB_SERVICES.find((service) => service.id === serviceId) ?? null
  }

  const response = await fetch(`${BASE_URL}/services/${serviceId}`, {
    next: { revalidate: REVALIDATE_SECONDS },
  })
  if (response.status === 404) return null
  if (!response.ok) {
    throw new Error(`Failed to load service ${serviceId}: ${response.status}`)
  }
  return response.json()
}

/**
 * `category` matches the documented contract exactly. `search` is deliberately
 * wider than the documented name-only behaviour — see the README note; the
 * backend needs the same widening before this can be delegated server-side.
 */
function filterStub(services: Service[], options: GetServicesOptions): Service[] {
  const search = options.search?.trim().toLowerCase()
  return services.filter((service) => {
    if (options.category && service.category !== options.category) return false
    if (!search) return true
    return (
      service.name.toLowerCase().includes(search) ||
      service.dags.some((dag) => dag.dag_id.toLowerCase().includes(search))
    )
  })
}
