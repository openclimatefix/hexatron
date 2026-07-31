/**
 * Timestamps are formatted in UTC so server and client renders agree. Switching
 * to the viewer's local zone would need a client-only boundary to avoid
 * hydration mismatches.
 */
const timeFormatter = new Intl.DateTimeFormat('en-US', {
  hour: '2-digit',
  minute: '2-digit',
  hour12: true,
  timeZone: 'UTC',
})

export const EM_DASH = '—'

export function formatTime(iso: string | null): string {
  if (!iso) return EM_DASH
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return EM_DASH
  return timeFormatter.format(date)
}

export function formatDuration(seconds: number | null): string {
  if (seconds === null || !Number.isFinite(seconds)) return EM_DASH

  const total = Math.round(seconds)
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const secs = total % 60

  if (hours > 0) return `${hours}h ${minutes}m`
  if (minutes > 0) return `${minutes}m ${secs}s`
  return `${secs}s`
}

/** Renders 0.995 as "99.5", 0.92 as "92" — trailing ".0" is noise on a card. */
export function formatSuccessRate(rate: number | null): string | null {
  if (rate === null || !Number.isFinite(rate)) return null
  const pct = rate * 100
  return Number.isInteger(pct) ? String(pct) : pct.toFixed(1)
}

export type DurationTrend = 'slower' | 'faster' | 'steady' | 'none'

/**
 * Compares the latest run against the average. The 10% deadband keeps normal
 * run-to-run jitter from showing as a trend.
 */
export function durationTrend(latest: number | null, average: number | null): DurationTrend {
  if (latest === null || average === null || average <= 0) return 'none'
  const delta = (latest - average) / average
  if (delta > 0.1) return 'slower'
  if (delta < -0.1) return 'faster'
  return 'steady'
}

export const TREND_GLYPH: Record<DurationTrend, string> = {
  slower: '▲',
  faster: '▼',
  steady: EM_DASH,
  none: EM_DASH,
}
