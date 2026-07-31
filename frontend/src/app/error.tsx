'use client'

import { useEffect } from 'react'

import { Button } from '@/components/ui/button'
import { ServiceLoadError } from '@/components/service-load-error'

/**
 * Backstop for unexpected client-side and navigation errors. Expected failures
 * — chiefly an unreachable backend — are caught in the pages themselves, since
 * this boundary doesn't render during a failed initial server render.
 */
export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    console.error('Dashboard error:', error)
  }, [error])

  return (
    <ServiceLoadError
      title="Something went wrong"
      detail={error.digest ? `Ref ${error.digest}` : error.message}
      action={
        <Button size="lg" className="rounded-lg px-4" onClick={reset}>
          Try again
        </Button>
      }
    />
  )
}
