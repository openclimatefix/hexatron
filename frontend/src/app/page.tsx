import { DashboardView } from '@/components/dashboard/dashboard-view'
import { getServices } from '@/lib/api'

export const metadata = {
  title: 'Dashboard',
}

export default async function DashboardPage() {
  const services = await getServices()

  return <DashboardView services={services} />
}
