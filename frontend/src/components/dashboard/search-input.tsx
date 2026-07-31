'use client'

import { Search } from 'lucide-react'

import { Input } from '@/components/ui/input'

export function SearchInput({
  value,
  onChange,
}: {
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="relative w-full sm:w-72">
      <Search
        className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-black/35"
        aria-hidden
      />
      <Input
        type="search"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder="Search DAGs..."
        aria-label="Search services and DAGs"
        className="h-9 pl-9"
      />
    </div>
  )
}
