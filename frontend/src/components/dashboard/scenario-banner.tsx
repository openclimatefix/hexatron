import { SCENARIOS } from '@/lib/scenarios'

/**
 * Makes it unmistakable that the dashboard isn't showing live data. Worth being
 * loud about — a scenario is indistinguishable from a real incident otherwise.
 */
export function ScenarioBanner({
  scenario,
  error,
  loading,
}: {
  scenario: string
  error: string | null
  loading: boolean
}) {
  const option = SCENARIOS.find((s) => s.id === scenario)

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-black/10 bg-ink px-6 py-2 text-sm text-white lg:px-10">
      <span className="rounded-full bg-white/15 px-2 py-0.5 text-xs font-medium tracking-wide uppercase">
        Scenario
      </span>
      <span className="font-medium">{option?.label ?? scenario}</span>
      <span className="text-white/60">
        {error
          ? `Couldn’t load this scenario — ${error}`
          : loading
            ? 'Loading…'
            : (option?.summary ?? 'Simulated data, not live')}
      </span>
    </div>
  )
}
