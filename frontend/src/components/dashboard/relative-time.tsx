'use client'

import { useEffect, useState } from 'react'

import { EM_DASH, formatRelative, formatTime } from '@/lib/format'

/** Re-rendered on this cadence so "4m ago" doesn't quietly go stale. */
const TICK_MS = 30_000

/**
 * A timestamp shown relative to now — "4m ago", "in 41m".
 *
 * Renders the absolute UTC time first and swaps to relative once mounted. That
 * ordering is deliberate: the relative value depends on the current clock, so
 * emitting it during SSR would guarantee a hydration mismatch. The first client
 * render still matches the server's, and the effect upgrades it afterwards.
 */
export function RelativeTime({
  iso,
  fallback = EM_DASH,
  className,
}: {
  iso: string | null
  /** Shown when there's no timestamp at all, e.g. "Not scheduled". */
  fallback?: string
  className?: string
}) {
  const [now, setNow] = useState<number | null>(null)

  useEffect(() => {
    if (!iso) return

    const tick = () => setNow(Date.now())
    tick()
    const timer = setInterval(tick, TICK_MS)
    return () => clearInterval(timer)
  }, [iso])

  if (!iso) return <span className={className}>{fallback}</span>

  return (
    // The absolute time stays reachable on hover — "in 41m" is the better
    // default, but an operator comparing against Airflow wants the real clock.
    <span className={className} title={formatTime(iso)}>
      {now === null ? formatTime(iso) : formatRelative(iso, now)}
    </span>
  )
}
