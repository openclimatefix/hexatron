#!/usr/bin/env node
/**
 * Regenerates the demo fixtures in public/scenarios/ from a live backend.
 *
 *   node scripts/build-scenarios.mjs [baseUrl]
 *   API_BASE_URL=https://… node scripts/build-scenarios.mjs
 *
 * It snapshots /services and each /services/{id}, then rewrites run states,
 * durations and notes per scenario. DAG ids, schedules and run timestamps are
 * left as they came off the real Airflow, which is what keeps the fixtures
 * looking plausible.
 *
 * Re-run this whenever the service registry changes; the committed JSON is the
 * artefact the app actually loads.
 */
import { mkdir, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const BASE_URL = process.argv[2] ?? process.env.API_BASE_URL ?? 'http://localhost:8080'
const OUT_DIR = join(dirname(fileURLToPath(import.meta.url)), '..', 'public', 'scenarios')

async function getJson(path) {
  const response = await fetch(`${BASE_URL}${path}`)
  if (!response.ok) throw new Error(`GET ${path} → HTTP ${response.status}`)
  return response.json()
}

const clone = (value) => JSON.parse(JSON.stringify(value))
const findDag = (service, id) => service.dags.find((dag) => dag.dag_id === id)
const byId = (services, id) => services.find((service) => service.id === id)

/** Rewrites a DAG's runs from a pattern, keeping the snapshot's real timestamps. */
function setRuns(dag, pattern, { durationSeconds } = {}) {
  const codes = [...pattern]
  dag.runs = (dag.runs ?? []).slice(0, codes.length).map((run, i) => {
    const state = codes[i] === 'F' ? 'failed' : codes[i] === 'R' ? 'running' : 'success'
    const start = run.start_date ?? run.logical_date
    let end = run.end_date
    if (state === 'running') end = null
    else if (durationSeconds) end = new Date(Date.parse(start) + durationSeconds * 1000).toISOString()
    else if (!end) end = new Date(Date.parse(start) + 120_000).toISOString()
    return { ...run, state, start_date: start, end_date: end }
  })
  return dag
}

const durationOf = (run) =>
  run.start_date && run.end_date
    ? (Date.parse(run.end_date) - Date.parse(run.start_date)) / 1000
    : null

/** Window totals for a DAG. runs[] only holds the visible last-N. */
function setMetrics(dag, { totalRuns, failedRuns, avgSeconds, latestSeconds }) {
  const durations = dag.runs.map(durationOf).filter((d) => d !== null)
  const observedAvg = durations.length
    ? durations.reduce((a, b) => a + b, 0) / durations.length
    : null
  dag.metrics = {
    total_runs: totalRuns,
    failed_runs: failedRuns,
    success_rate: totalRuns > 0 ? (totalRuns - failedRuns) / totalRuns : null,
    avg_duration_seconds: avgSeconds ?? observedAvg,
    latest_duration_seconds: latestSeconds !== undefined ? latestSeconds : (durations[0] ?? null),
    last_run_at: dag.runs[0]?.start_date ?? null,
    next_run_at: dag.next_dagrun_logical_date ?? null,
  }
  return dag
}

const EMPTY_METRICS = {
  total_runs: 0,
  failed_runs: 0,
  success_rate: null,
  avg_duration_seconds: null,
  latest_duration_seconds: null,
  last_run_at: null,
  next_run_at: null,
}

/** Mirrors aggregateMetrics in src/lib/normalise.ts. */
function rollUp(service) {
  const dags = service.dags ?? []
  if (dags.length === 0) {
    service.metrics = EMPTY_METRICS
    return service
  }
  const ms = dags.map((dag) => dag.metrics)
  const total = ms.reduce((n, m) => n + m.total_runs, 0)
  const failed = ms.reduce((n, m) => n + m.failed_runs, 0)
  const weighted = ms.filter((m) => m.avg_duration_seconds !== null && m.total_runs > 0)
  const weight = weighted.reduce((n, m) => n + m.total_runs, 0)
  const lastTimes = ms.map((m) => m.last_run_at).filter(Boolean)
  const last = lastTimes.length
    ? lastTimes.reduce((a, b) => (Date.parse(b) > Date.parse(a) ? b : a))
    : null
  const nextTimes = ms.map((m) => m.next_run_at).filter(Boolean)

  service.metrics = {
    total_runs: total,
    failed_runs: failed,
    success_rate: total > 0 ? (total - failed) / total : null,
    avg_duration_seconds: weight
      ? weighted.reduce((n, m) => n + m.avg_duration_seconds * m.total_runs, 0) / weight
      : null,
    latest_duration_seconds: ms.find((m) => m.last_run_at === last)?.latest_duration_seconds ?? null,
    last_run_at: last,
    next_run_at: nextTimes.length
      ? nextTimes.reduce((a, b) => (Date.parse(b) < Date.parse(a) ? b : a))
      : null,
  }
  return service
}

/** Drops the transitional duplicates the backend still sends alongside the real fields. */
function tidy(service) {
  for (const dag of service.dags ?? []) {
    delete dag.schedule
    delete dag.last_run
  }
  return service
}

/** Every DAG green — the baseline each scenario diverges from. */
function baseline(snapshot) {
  return snapshot.map((raw) => {
    const service = tidy(clone(raw))
    service.note = null
    for (const dag of service.dags ?? []) {
      dag.is_paused = false
      dag.status = 'healthy'
      setRuns(dag, 'SSSSSSSSSS')
      setMetrics(dag, { totalRuns: 100, failedRuns: 0 })
    }
    const hasDags = (service.dags ?? []).length > 0
    service.status = hasDags ? 'healthy' : 'unknown'
    if (!hasDags) service.note = 'No monitoring data'
    return rollUp(service)
  })
}

// ---------------------------------------------------------------------------

/** An upstream feed dies; the damage is upstream but the symptoms look downstream. */
function upstreamOutage(snapshot) {
  const out = baseline(snapshot)

  const consumer = byId(out, 'consumer')
  if (consumer) {
    for (const id of ['uk-consume-nwp', 'nl-consume-nwp']) {
      const dag = findDag(consumer, id)
      if (!dag) continue
      setRuns(dag, 'FFFFFFFFFF', { durationSeconds: 38 })
      setMetrics(dag, { totalRuns: 96, failedRuns: 84, avgSeconds: 41, latestSeconds: 38 })
      dag.status = 'down'
    }
    consumer.status = 'down'
    consumer.note = 'ECMWF NWP feed unreachable since 17:40 yesterday — 14h of failures'
    rollUp(consumer)
  }

  const solar = byId(out, 'solar-forecast')
  if (solar) {
    for (const id of ['uk-forecast-gsp', 'uk-forecast-site']) {
      const dag = findDag(solar, id)
      if (!dag) continue
      setRuns(dag, 'FSFFSFSSFS')
      setMetrics(dag, { totalRuns: 100, failedRuns: 38 })
      dag.status = 'degraded'
    }
    solar.status = 'degraded'
    solar.note = 'Running on stale NWP — no fresh inputs since 17:40'
    rollUp(solar)
  }

  const api = byId(out, 'api')
  if (api) api.note = 'Healthy, but serving last-good forecasts — inputs are 14h stale'

  return out
}

/** Nothing has failed yet; runtimes are creeping toward their windows. */
function silentDegradation(snapshot) {
  const out = baseline(snapshot)

  const consumer = byId(out, 'consumer')
  if (consumer) {
    const slow = findDag(consumer, 'uk-consume-pvlive-intraday')
    if (slow) {
      setRuns(slow, 'SSSSSSSSSS', { durationSeconds: 738 })
      setMetrics(slow, { totalRuns: 100, failedRuns: 0, avgSeconds: 512, latestSeconds: 738 })
      slow.status = 'degraded'
    }
    const pv = findDag(consumer, 'uk-consume-pv')
    if (pv) {
      setRuns(pv, 'SSSSSSSSSS', { durationSeconds: 268 })
      setMetrics(pv, { totalRuns: 100, failedRuns: 0, avgSeconds: 191, latestSeconds: 268 })
    }
    consumer.status = 'degraded'
    consumer.note = 'uk-consume-pvlive-intraday runtime up 3x — now overruns its 15m window'
    rollUp(consumer)
  }

  const solar = byId(out, 'solar-forecast')
  if (solar) {
    const site = findDag(solar, 'uk-forecast-site')
    if (site) {
      setRuns(site, 'SSSSSSSSSS', { durationSeconds: 611 })
      setMetrics(site, { totalRuns: 100, failedRuns: 0, avgSeconds: 430, latestSeconds: 611 })
    }
    rollUp(solar)
  }

  return out
}

/** Deliberately-off work sitting alongside one genuine failure. */
function maintenanceVsFailure(snapshot) {
  const out = baseline(snapshot)

  const consumer = byId(out, 'consumer')
  if (consumer) {
    for (const id of ['uk-consume-sat', 'uk-consume-sat-v1']) {
      const dag = findDag(consumer, id)
      if (!dag) continue
      dag.is_paused = true
      dag.status = 'paused'
      dag.next_dagrun_logical_date = null
      setRuns(dag, 'SSSSSSSSSS')
      setMetrics(dag, { totalRuns: 0, failedRuns: 0, avgSeconds: null, latestSeconds: null })
      dag.metrics.next_run_at = null
    }
    consumer.note = 'Satellite ingestion paused for GOES-19 migration — resumes 14:00'
    rollUp(consumer)
  }

  const solar = byId(out, 'solar-forecast')
  if (solar) {
    const broken = findDag(solar, 'uk-analysis-clouds')
    if (broken) {
      setRuns(broken, 'FFFSSSSSSS', { durationSeconds: 96 })
      setMetrics(broken, { totalRuns: 100, failedRuns: 22, avgSeconds: 310, latestSeconds: 96 })
      broken.status = 'down'
    }
    solar.status = 'down'
    solar.note = 'uk-analysis-clouds failing since 06:00 — this one is real'
    rollUp(solar)
  }

  const api = byId(out, 'api')
  if (api) {
    for (const dag of api.dags ?? []) {
      dag.is_paused = true
      dag.status = 'paused'
      dag.next_dagrun_logical_date = null
      setMetrics(dag, { totalRuns: 0, failedRuns: 0, avgSeconds: null, latestSeconds: null })
      dag.metrics.next_run_at = null
    }
    api.status = 'paused'
    api.note = 'Paused for deploy — v2.4.1 rollout'
    rollUp(api)
  }

  return out
}

const SCENARIOS = {
  'upstream-outage': upstreamOutage,
  'silent-degradation': silentDegradation,
  'maintenance-vs-failure': maintenanceVsFailure,
}

async function main() {
  console.log(`Snapshotting ${BASE_URL}`)
  const summaries = await getJson('/services')
  const snapshot = await Promise.all(summaries.map((s) => getJson(`/services/${s.id}`)))
  console.log(
    `  ${snapshot.length} services, ` +
      `${snapshot.reduce((n, s) => n + (s.dags ?? []).length, 0)} DAGs`,
  )

  await mkdir(OUT_DIR, { recursive: true })
  for (const [name, build] of Object.entries(SCENARIOS)) {
    const data = build(snapshot)
    await writeFile(join(OUT_DIR, `${name}.json`), JSON.stringify(data, null, 2) + '\n')
    const counts = {}
    for (const service of data) counts[service.status] = (counts[service.status] ?? 0) + 1
    const summary = Object.entries(counts)
      .map(([status, n]) => `${n} ${status}`)
      .join(' · ')
    console.log(`  ${name.padEnd(24)} ${summary}`)
  }
}

main().catch((error) => {
  console.error(`\nFailed: ${error.message}`)
  console.error('Pass a base URL as the first argument, or set API_BASE_URL.')
  process.exitCode = 1
})
