'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

import { cn } from '@/lib/utils'

const LINKS = [
  { href: '/', label: 'Dashboard' },
  { href: '/services', label: 'Services' },
  { href: '/alerts', label: 'Alerts' },
  { href: '/settings', label: 'Settings' },
]

function isActive(pathname: string, href: string): boolean {
  if (href === '/') return pathname === '/'
  return pathname === href || pathname.startsWith(`${href}/`)
}

export function Nav() {
  const pathname = usePathname()

  return (
    <nav aria-label="Primary">
      <ul className="flex items-center gap-6">
        {LINKS.map((link) => {
          const active = isActive(pathname, link.href)
          return (
            <li key={link.href}>
              <Link
                href={link.href}
                aria-current={active ? 'page' : undefined}
                className={cn(
                  'relative block py-5 text-sm transition-colors',
                  active ? 'font-medium text-ink' : 'text-black/50 hover:text-black/75',
                )}
              >
                {link.label}
                {active && (
                  <span
                    className="absolute inset-x-0 bottom-0 h-0.5 rounded-full bg-flame"
                    aria-hidden
                  />
                )}
              </Link>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
