import { normaliseService, normaliseServices } from '@/lib/normalise'
import { STUB_SERVICES } from '@/lib/stub-data'
import type { Service } from '@/lib/types'

/**
 * The seam between the dashboard and the Go backend. While `USE_STUB_DATA` is
 * on, everything is served from `@/lib/stub-data`; flip the env var once the
 * backend's /services endpoints are live and no component needs to change.
 */
const BASE_URL = process.env.API_BASE_URL ?? 'http://localhost:8080'
const USE_STUB_DATA = process.env.USE_STUB_DATA !== 'false'

const REVALIDATE_SECONDS = 15
/** A tunnelled dev backend can hang; don't let it hold the render open. */
const TIMEOUT_MS = 8000

export interface GetServicesOptions {
  search?: string
  category?: string
}

/**
 * The backend returns `{"error": "..."}` on failure — surfacing that beats a
 * bare status code when the real cause is something like "could not reach
 * airflow".
 */
async function describeFailure(response: Response): Promise<string> {
  try {
    const body = await response.json()
    const message = (body as { error?: unknown })?.error
    if (typeof message === 'string' && message.length > 0) {
      return `${message} (HTTP ${response.status})`
    }
  } catch {
    // Non-JSON body — the status code is all we have.
  }
  return `HTTP ${response.status}`
}

/**
 * A tunnelled dev backend can return a 200 with a truncated or double-encoded
 * body. Fail with something diagnosable rather than a raw SyntaxError.
 */
async function parseJson(response: Response, path: string): Promise<unknown> {
  const text = await response.text()
  try {
    return JSON.parse(text)
  } catch {
    const preview = text.trim().slice(0, 80)
    throw new Error(`${path} returned a non-JSON body: ${preview || '(empty)'}`)
  }
}

async function request(path: string): Promise<Response> {
  try {
    return await fetch(`${BASE_URL}${path}`, {
      next: { revalidate: REVALIDATE_SECONDS },
      signal: AbortSignal.timeout(TIMEOUT_MS),
    })
  } catch (error) {
    const reason = error instanceof Error ? error.message : String(error)
    throw new Error(`Could not reach ${BASE_URL} — ${reason}`)
  }
}

export async function getServices(options: GetServicesOptions = {}): Promise<Service[]> {
  if (USE_STUB_DATA) return filterStub(STUB_SERVICES, options)

  const params = new URLSearchParams()
  if (options.search) params.set('search', options.search)
  if (options.category) params.set('category', options.category)
  const query = params.size > 0 ? `?${params}` : ''

  const response = await request(`/services${query}`)
  if (!response.ok) {
    throw new Error(await describeFailure(response))
  }
  return normaliseServices(await parseJson(response, `/services${query}`))
}

export async function getService(serviceId: string): Promise<Service | null> {
  if (USE_STUB_DATA) {
    return STUB_SERVICES.find((service) => service.id === serviceId) ?? null
  }

  const response = await request(`/services/${serviceId}`)
  if (response.status === 404) return null
  if (!response.ok) {
    throw new Error(await describeFailure(response))
  }
  return normaliseService(await parseJson(response, `/services/${serviceId}`))
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
