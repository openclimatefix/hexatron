'use client'

import { createContext, useContext, useMemo, useState, type ReactNode } from 'react'

import { LIVE } from '@/lib/scenarios'
import type { TimeRange } from '@/lib/types'

interface DashboardFilters {
  search: string
  setSearch: (value: string) => void
  range: TimeRange
  setRange: (value: TimeRange) => void
  /** `live` for the real API, otherwise a fixture id from `@/lib/scenarios`. */
  scenario: string
  setScenario: (value: string) => void
}

const FilterContext = createContext<DashboardFilters | null>(null)

/**
 * Shares filter state between the header controls and the dashboard body, which
 * sit in different parts of the tree. Kept in memory rather than the URL so
 * typing doesn't trigger a navigation per keystroke.
 */
export function DashboardFilterProvider({ children }: { children: ReactNode }) {
  const [search, setSearch] = useState('')
  const [range, setRange] = useState<TimeRange>('24h')
  const [scenario, setScenario] = useState<string>(LIVE)

  const value = useMemo(
    () => ({ search, setSearch, range, setRange, scenario, setScenario }),
    [search, range, scenario],
  )

  return <FilterContext.Provider value={value}>{children}</FilterContext.Provider>
}

export function useDashboardFilters(): DashboardFilters {
  const context = useContext(FilterContext)
  if (!context) {
    throw new Error('useDashboardFilters must be used within a DashboardFilterProvider')
  }
  return context
}
