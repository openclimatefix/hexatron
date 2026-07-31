/**
 * Pre-loaded fixtures for demoing states that are hard to reproduce on demand.
 *
 * Each was built from a snapshot of the live backend, so DAG ids, schedules and
 * run timestamps are real — only run states, durations and notes are rewritten.
 * They live in `public/scenarios/` and are fetched on selection.
 */
export const LIVE = 'live'

export interface ScenarioOption {
  id: string
  label: string
  /** One line on what the scenario shows — used as the dropdown hint. */
  summary: string
}

export const SCENARIOS: ScenarioOption[] = [
  {
    id: LIVE,
    label: 'Live',
    summary: 'Current state from the Airflow-backed API',
  },
  {
    id: 'upstream-outage',
    label: 'Upstream outage',
    summary: 'ECMWF NWP feed down 14h — cascade from Consumers',
  },
  {
    id: 'silent-degradation',
    label: 'Silent degradation',
    summary: 'Nothing failing, but runtimes creeping toward their windows',
  },
  {
    id: 'maintenance-vs-failure',
    label: 'Maintenance vs failure',
    summary: 'Paused work alongside one genuine failure',
  },
]

export function isScenario(value: string): boolean {
  return value !== LIVE && SCENARIOS.some((scenario) => scenario.id === value)
}

export function scenarioLabel(id: string): string {
  return SCENARIOS.find((scenario) => scenario.id === id)?.label ?? id
}

export function scenarioUrl(id: string): string {
  return `/scenarios/${id}.json`
}
