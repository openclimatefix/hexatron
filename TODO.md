NOW
- [x] UI heartbeat
- [x] API heartbeat
- [x] Data Platform health check (gRPC — same mechanism as the two above)
- [x] "Down" on forecast not correct — cloudcasting is now `non_critical_dag_patterns`
      in services.yaml, so it degrades rather than downs; the failing DAG is named
      on the card via `status_reason`
- [x] Check Last 01:20 PM Next 11:00 AM wiring — values were right, the absolute
      UTC formatting was the bug; now relative ("4m ago" / "in 41m")
- [x] Running grey squares should be obvious e.g. flash (already done — solid
      `bg-active` + run-pulse; only `queued` is grey)
- [x] Links to Airflow DAGs (already done — c690799)
- [x] Hover states on squares (already done — Radix tooltips)
- [x] Running time ticker (from started at) (already done — elapsed-time.tsx)
- [ ] Real URLs for the three health_check targets — currently PLACEHOLDERS in
      backend/data/services.yaml (`*.example.invalid`, `localhost:50051`).
      Unresolvable targets deliberately report "Not checked", not "Down".
- [ ] Avg / Latest on Service doesn't make sense — still on the service detail
      page ([serviceId]/page.tsx, "Avg duration"); blending durations across
      heterogeneous DAGs is meaningless
- [ ] Favicon (src/app/favicon.ico is still the Next.js default)
- [ ] Wind Service show as not implemented/coming soon (has no DAGs, so reports
      "unknown"; wants a config flag rather than a hardcoded id check)
- [ ] DP -> Data Platform (service name is already right; the DAG display names
      underneath need checking against live data)
- [ ] Grid per DAG? Could be too busy, but could be good quick glance
- [ ] long-running DAGs mark as unhealthy if abnormal
- [ ] Split services by country? e.g. the squares-grid

PRE-EXISTING TEST FAILURES (not from this work — confirmed against a clean tree)
- TestListServicesEndpoint — asserts DAGs are omitted from /services, but
  c690799 deliberately started retaining them. Stale assertion.
- TestDAGURLHostReplacement — expects host 128.0.0.1, gets 127.0.0.1.

LATER
