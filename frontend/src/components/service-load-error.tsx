import type { ReactNode } from 'react'

/**
 * Shown when the services API can't be reached. The backend being unavailable
 * is an expected state while it's still being built, so pages catch the failure
 * and render this rather than letting it escape as an unhandled error.
 */
export function ServiceLoadError({
  title = 'Couldn’t reach the services API',
  detail,
  action,
}: {
  title?: string
  detail?: string
  action?: ReactNode
}) {
  return (
    <div className="flex flex-1 items-center justify-center bg-canvas px-6 py-20">
      <div className="max-w-md text-center">
        <h1 className="font-display text-2xl font-medium tracking-tight text-ink">{title}</h1>
        <p className="mt-2 text-sm leading-relaxed text-black/55">
          The dashboard couldn’t load service health. Check that the backend is running on the
          configured <code className="text-xs">API_BASE_URL</code>, or set{' '}
          <code className="text-xs">USE_STUB_DATA=true</code> to work against stub data.
        </p>
        {detail && <p className="mt-2 font-mono text-xs break-all text-black/35">{detail}</p>}
        {action && <div className="mt-6">{action}</div>}
      </div>
    </div>
  )
}
