import { PagePlaceholder } from '@/components/page-placeholder'

export const metadata = {
  title: 'Alerts',
}

export default function AlertsPage() {
  return (
    <PagePlaceholder title="Alerts">
      Alerting isn’t wired up yet. Once the backend exposes DAG run history, this is where failures
      and SLA breaches will surface.
    </PagePlaceholder>
  )
}
