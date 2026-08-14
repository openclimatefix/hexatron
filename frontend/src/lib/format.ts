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

/**
 * Renders a timestamp as "4m ago" or "in 41m".
 *
 * Absolute clock times are what made Last/Next unreadable: the service's last
 * run and its soonest upcoming run come from different DAGs, and with no date
 * shown, a next run tomorrow reads as a time earlier in the day than the last
 * one. Relative phrasing makes the ordering self-evident.
 *
 * `now` is a parameter rather than a call to `Date.now()` so this stays pure —
 * the caller owns the clock, which is what keeps it testable and keeps server
 * and client renders from disagreeing.
 */
export function formatRelative(iso: string | null, now: number): string {
  if (!iso) return EM_DASH
  const then = Date.parse(iso)
  if (Number.isNaN(then)) return EM_DASH

  const deltaSeconds = Math.round((then - now) / 1000)
  const future = deltaSeconds > 0
  const magnitude = Math.abs(deltaSeconds)

  // Under a minute either way, the exact number is noise.
  if (magnitude < 60) return future ? 'in <1m' : 'just now'

  const minutes = Math.floor(magnitude / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  let span: string
  if (days > 0) span = `${days}d`
  else if (hours > 0) span = minutes % 60 === 0 ? `${hours}h` : `${hours}h ${minutes % 60}m`
  else span = `${minutes}m`

  return future ? `in ${span}` : `${span} ago`
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
