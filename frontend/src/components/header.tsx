'use client'

import Image from 'next/image'
import Link from 'next/link'
import { usePathname } from 'next/navigation'

import { Nav } from '@/components/nav'
import { SearchInput } from '@/components/dashboard/search-input'
import { TimeRangeSelect } from '@/components/dashboard/time-range-select'
import { useDashboardFilters } from '@/components/dashboard/filter-context'

export function Header() {
  const pathname = usePathname()
  const { search, setSearch, range, setRange } = useDashboardFilters()

  // The filters only act on the dashboard grid, so they'd be inert elsewhere.
  const showFilters = pathname === '/'

  return (
    <header className="flex flex-wrap items-center gap-x-8 gap-y-3 border-b border-black/8 bg-white px-6 lg:px-10">
      <Link href="/" className="flex items-center gap-3 py-4">
        <Image
          src="/logo.png"
          alt=""
          width={28}
          height={28}
          className="size-7 object-contain"
          priority
        />
        <span className="font-display text-xl font-medium tracking-tight text-ink">Hexatron</span>
      </Link>

      <Nav />

      {showFilters && (
        <div className="ml-auto flex flex-wrap items-center gap-3 py-3">
          <SearchInput value={search} onChange={setSearch} />
          <TimeRangeSelect value={range} onChange={setRange} />
        </div>
      )}
    </header>
  )
}
