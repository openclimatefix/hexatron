import type { Metadata } from 'next'
import localFont from 'next/font/local'

import { Header } from '@/components/header'
import { DashboardFilterProvider } from '@/components/dashboard/filter-context'
import './globals.css'

const display = localFont({
  src: [
    { path: './fonts/MatterXHLight.otf', weight: '300', style: 'normal' },
    { path: './fonts/MatterXHRegular.otf', weight: '400', style: 'normal' },
    { path: './fonts/MatterXHMedium.ttf', weight: '500', style: 'normal' },
  ],
  variable: '--font-display',
})

const body = localFont({
  src: [
    { path: './fonts/MatterSemiMonoRegular.otf', weight: '400', style: 'normal' },
    { path: './fonts/MatterSemiMonoMedium.otf', weight: '500', style: 'normal' },
  ],
  variable: '--font-sans',
})

export const metadata: Metadata = {
  title: {
    default: 'Hexatron',
    template: '%s — Hexatron',
  },
  description: 'Service-centric operational dashboard for the Open Climate Fix platform.',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en" className={`${display.variable} ${body.variable} h-full antialiased`}>
      <body className="flex min-h-full flex-col bg-canvas text-ink">
        <DashboardFilterProvider>
          <Header />
          <main className="flex flex-1 flex-col">{children}</main>
        </DashboardFilterProvider>
      </body>
    </html>
  )
}
