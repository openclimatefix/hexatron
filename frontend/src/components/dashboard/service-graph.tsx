'use client'

import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'

import { ServiceCard } from '@/components/dashboard/service-card'
import { isAttentionStatus } from '@/lib/status'
import type { Service } from '@/lib/types'

interface Box {
  left: number
  top: number
  right: number
  bottom: number
  centerX: number
  centerY: number
}

interface Edge {
  id: string
  path: string
  attention: boolean
}

/** Below this the grid collapses to a single column and edges stop being meaningful. */
const GRAPH_MEDIA_QUERY = '(min-width: 768px)'
/** Centres within this many px count as aligned, absorbing sub-pixel layout drift. */
const ALIGNMENT_TOLERANCE = 8
const ARROW_GAP = 7

function toBox(element: HTMLElement, origin: DOMRect): Box {
  const rect = element.getBoundingClientRect()
  const left = rect.left - origin.left
  const top = rect.top - origin.top
  return {
    left,
    top,
    right: left + rect.width,
    bottom: top + rect.height,
    centerX: left + rect.width / 2,
    centerY: top + rect.height / 2,
  }
}

type Side = 'top' | 'bottom' | 'left' | 'right'

/** Which face of the target card an incoming edge should terminate on. */
function entrySide(from: Box, to: Box): Side {
  if (Math.abs(from.centerY - to.centerY) < ALIGNMENT_TOLERANCE) {
    return to.centerX > from.centerX ? 'left' : 'right'
  }
  return to.centerY > from.centerY ? 'top' : 'bottom'
}

/**
 * Spreads n arrival points across the middle half of a card's face, so several
 * upstream services converging on one card don't stack on the same pixel.
 */
function fanOut(start: number, end: number, count: number, index: number): number {
  const center = (start + end) / 2
  if (count <= 1) return center
  const band = (end - start) * 0.5
  return center - band / 2 + (band * index) / (count - 1)
}

/**
 * Orthogonal route from a source card's face to a given point on the target's.
 * Collapses to a straight line when source and entry already line up.
 */
function routeEdge(from: Box, to: Box, side: Side, entry: number): string {
  if (side === 'top' || side === 'bottom') {
    const down = side === 'top'
    const startY = down ? from.bottom : from.top
    const rawEndY = down ? to.top : to.bottom
    const endY = rawEndY + (down ? -ARROW_GAP : ARROW_GAP)
    const midY = (startY + rawEndY) / 2

    return [
      `M ${from.centerX} ${startY}`,
      `L ${from.centerX} ${midY}`,
      `L ${entry} ${midY}`,
      `L ${entry} ${endY}`,
    ].join(' ')
  }

  const right = side === 'left'
  const startX = right ? from.right : from.left
  const rawEndX = right ? to.left : to.right
  const endX = rawEndX + (right ? -ARROW_GAP : ARROW_GAP)
  const midX = (startX + rawEndX) / 2

  return [
    `M ${startX} ${from.centerY}`,
    `L ${midX} ${from.centerY}`,
    `L ${midX} ${entry}`,
    `L ${endX} ${entry}`,
  ].join(' ')
}

export function ServiceGraph({ services }: { services: Service[] }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const cardRefs = useRef(new Map<string, HTMLDivElement>())
  const [edges, setEdges] = useState<Edge[]>([])
  const [size, setSize] = useState({ width: 0, height: 0 })
  const [showGraph, setShowGraph] = useState(false)

  const registerCard = useCallback((id: string, node: HTMLDivElement | null) => {
    if (node) cardRefs.current.set(id, node)
    else cardRefs.current.delete(id)
  }, [])

  useEffect(() => {
    const query = window.matchMedia(GRAPH_MEDIA_QUERY)
    const sync = () => setShowGraph(query.matches)
    sync()
    query.addEventListener('change', sync)
    return () => query.removeEventListener('change', sync)
  }, [])

  const measure = useCallback(() => {
    const container = containerRef.current
    if (!container) return

    const origin = container.getBoundingClientRect()
    setSize({ width: origin.width, height: origin.height })

    const visible = new Set(services.map((service) => service.id))
    const statusById = new Map(services.map((s) => [s.id, s.status]))
    const next: Edge[] = []

    for (const service of services) {
      const toNode = cardRefs.current.get(service.id)
      if (!toNode) continue
      const to = toBox(toNode, origin)

      // Skip edges from cards filtered out of view — a dangling arrow is worse
      // than no arrow.
      const incoming = service.depends_on
        .filter((id) => visible.has(id) && cardRefs.current.has(id))
        .map((id) => {
          const from = toBox(cardRefs.current.get(id)!, origin)
          return { id, from, side: entrySide(from, to) }
        })

      // Arrivals are spread per face, so edges reaching the same side of a card
      // land on distinct points.
      const bySide = new Map<Side, typeof incoming>()
      for (const edge of incoming) {
        const group = bySide.get(edge.side) ?? []
        group.push(edge)
        bySide.set(edge.side, group)
      }

      for (const [side, group] of bySide) {
        // Keep arrivals in visual order so edges don't cross unnecessarily.
        const ordered = [...group].sort((a, b) =>
          side === 'top' || side === 'bottom'
            ? a.from.centerX - b.from.centerX
            : a.from.centerY - b.from.centerY,
        )

        ordered.forEach((edge, index) => {
          const entry =
            side === 'top' || side === 'bottom'
              ? fanOut(to.left, to.right, ordered.length, index)
              : fanOut(to.top, to.bottom, ordered.length, index)

          const upstreamStatus = statusById.get(edge.id)
          next.push({
            id: `${edge.id}->${service.id}`,
            path: routeEdge(edge.from, to, side, entry),
            attention: upstreamStatus ? isAttentionStatus(upstreamStatus) : false,
          })
        })
      }
    }

    setEdges(next)
  }, [services])

  useLayoutEffect(() => {
    // Stale edges are harmless while narrow — rendering is gated on showGraph,
    // and they get re-measured on the way back to a wide viewport.
    if (!showGraph) return

    measure()

    const container = containerRef.current
    if (!container) return

    const observer = new ResizeObserver(measure)
    observer.observe(container)
    for (const node of cardRefs.current.values()) observer.observe(node)

    window.addEventListener('resize', measure)
    return () => {
      observer.disconnect()
      window.removeEventListener('resize', measure)
    }
  }, [measure, showGraph])

  return (
    <div ref={containerRef} className="relative">
      {showGraph && edges.length > 0 && (
        <svg
          className="pointer-events-none absolute inset-0 z-0 overflow-visible"
          width={size.width}
          height={size.height}
          aria-hidden
        >
          <defs>
            <marker
              id="hexatron-arrow"
              viewBox="0 0 8 8"
              refX="6"
              refY="4"
              markerWidth="6"
              markerHeight="6"
              orient="auto-start-reverse"
            >
              <path d="M 0 1 L 7 4 L 0 7 z" className="fill-black/30" />
            </marker>
            <marker
              id="hexatron-arrow-attention"
              viewBox="0 0 8 8"
              refX="6"
              refY="4"
              markerWidth="6"
              markerHeight="6"
              orient="auto-start-reverse"
            >
              <path d="M 0 1 L 7 4 L 0 7 z" className="fill-flame" />
            </marker>
          </defs>

          {edges.map((edge) => (
            <path
              key={edge.id}
              d={edge.path}
              fill="none"
              strokeWidth={1.5}
              strokeDasharray="5 5"
              className={edge.attention ? 'stroke-flame' : 'stroke-black/30'}
              markerEnd={`url(#${edge.attention ? 'hexatron-arrow-attention' : 'hexatron-arrow'})`}
            />
          ))}
        </svg>
      )}

      <div className="relative z-10 grid gap-x-16 gap-y-24 md:grid-cols-2 lg:grid-cols-3">
        {services.map((service) => (
          <div
            key={service.id}
            ref={(node) => {
              registerCard(service.id, node)
            }}
          >
            <ServiceCard service={service} className="h-full" />
          </div>
        ))}
      </div>

      <DependencySummary services={services} />
    </div>
  )
}

/** The SVG edges are decorative; this carries the same relationships to assistive tech. */
function DependencySummary({ services }: { services: Service[] }) {
  const withDependencies = services.filter((service) => service.depends_on.length > 0)
  if (withDependencies.length === 0) return null

  const nameById = new Map(services.map((service) => [service.id, service.name]))

  return (
    <div className="sr-only">
      <h3>Service dependencies</h3>
      <ul>
        {withDependencies.map((service) => (
          <li key={service.id}>
            {service.name} depends on{' '}
            {service.depends_on.map((id) => nameById.get(id) ?? id).join(', ')}
          </li>
        ))}
      </ul>
    </div>
  )
}
