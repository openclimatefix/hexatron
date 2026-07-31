import { DashboardView } from '@/components/dashboard/dashboard-view'
import { ServiceLoadError } from '@/components/service-load-error'
import { getServices } from '@/lib/api'
import type { Service } from '@/lib/types'

export const metadata = {
  title: 'Dashboard',
}

/**
 * Service health is live operational data: it must never be baked in at build
 * time, and a backend that's down during a deploy must not fail the build.
 */
export const dynamic = 'force-dynamic'

export default async function DashboardPage() {
  let services: Service[]
  try {
    services = await getServices()
  } catch (error) {
    // A missing backend is an expected state right now, so it renders as a
    // status message rather than escaping as an unhandled error.
    console.error('Failed to load services:', error)
    return <ServiceLoadError detail={error instanceof Error ? error.message : undefined} />
  }

  return <DashboardView services={services} />
}
