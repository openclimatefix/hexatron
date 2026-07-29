import type { ReactNode } from 'react'

/** Marks a route that exists for navigation but has no backend behind it yet. */
export function PagePlaceholder({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex-1 bg-canvas px-6 py-10 lg:px-10">
      <div className="mx-auto max-w-4xl">
        <h1 className="font-display text-3xl font-medium tracking-tight text-ink">{title}</h1>
        <div className="mt-4 rounded-xl border border-dashed border-black/12 bg-white/50 p-6 text-sm leading-relaxed text-black/55">
          {children}
        </div>
      </div>
    </div>
  )
}
