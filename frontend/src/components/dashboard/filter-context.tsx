'use client'

import { createContext, useContext, useMemo, useState, type ReactNode } from 'react'

import type { TimeRange } from '@/lib/types'

interface DashboardFilters {
  search: string
  setSearch: (value: string) => void
  range: TimeRange
  setRange: (value: TimeRange) => void
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

  const value = useMemo(() => ({ search, setSearch, range, setRange }), [search, range])

  return <FilterContext.Provider value={value}>{children}</FilterContext.Provider>
}

export function useDashboardFilters(): DashboardFilters {
  const context = useContext(FilterContext)
  if (!context) {
    throw new Error('useDashboardFilters must be used within a DashboardFilterProvider')
  }
  return context
}
