'use client'

import { useEffect, useState } from 'react'

import { formatDuration } from '@/lib/format'
import { cn } from '@/lib/utils'

/**
 * Live elapsed time for an in-flight run, ticking every second.
 *
 * Renders nothing until mounted: the value depends on the current clock, so
 * rendering it on the server would guarantee a hydration mismatch.
 */
export function ElapsedTime({ since, className }: { since: string; className?: string }) {
  const [seconds, setSeconds] = useState<number | null>(null)

  useEffect(() => {
    const started = Date.parse(since)
    if (Number.isNaN(started)) return

    const tick = () => setSeconds(Math.max(0, (Date.now() - started) / 1000))
    tick()
    const timer = setInterval(tick, 1000)
    return () => clearInterval(timer)
  }, [since])

  if (seconds === null) return null

  return (
    <span
      className={cn('tabular-nums', className)}
      aria-label={`Running for ${formatDuration(seconds)}`}
    >
      {formatDuration(seconds)}
    </span>
  )
}
