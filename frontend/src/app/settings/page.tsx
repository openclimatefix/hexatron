import { PagePlaceholder } from '@/components/page-placeholder'

export const metadata = {
  title: 'Settings',
}

export default function SettingsPage() {
  return (
    <PagePlaceholder title="Settings">
      Service definitions currently live in{' '}
      <code className="rounded bg-black/6 px-1 py-0.5 text-xs">backend/config/services.yaml</code>.
      Editing them from here depends on the backend exposing a write path.
    </PagePlaceholder>
  )
}
