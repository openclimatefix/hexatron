import type { Service } from '@/lib/types'

/**
 * Card order for the dashboard grid. The arrangement is deliberate rather than
 * alphabetical or topological: at three columns it puts each service adjacent
 * to the ones it feeds, so the dependency edges run short and mostly straight.
 *
 *   Solar Forecast   Consumers   UI
 *   Wind Forecast    DP          API
 *
 * Sources sit above or beside DP, and API sits directly under UI. Ordering by
 * anything else leaves DP and its three upstreams on opposite rows, which
 * routes every edge the long way round.
 */
const DISPLAY_ORDER = ['solar-forecast', 'consumer', 'ui', 'wind-forecast', 'data-platform', 'api']

/**
 * Sorts services into the layout above. Anything not named — a new service, or
 * a differently-configured registry — keeps its API order and follows on after,
 * so the grid never drops a card it doesn't recognise.
 */
export function orderForGraph(services: Service[]): Service[] {
  const rank = new Map(DISPLAY_ORDER.map((id, index) => [id, index]))
  const fallback = DISPLAY_ORDER.length

  return [...services]
    .map((service, index) => ({ service, index }))
    .sort((a, b) => {
      const rankA = rank.get(a.service.id) ?? fallback + a.index
      const rankB = rank.get(b.service.id) ?? fallback + b.index
      return rankA - rankB
    })
    .map(({ service }) => service)
}
