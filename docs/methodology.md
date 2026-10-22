# Methodology

This file describes how measurements will be made. Phase 0 does not collect them. When a later phase changes a definition, update this file in the same commit.

The hypotheses are frozen in `docs/hypotheses.md`. The build phases are in `docs/PLAN.md`. Resolved and still-open setup questions are in `docs/decisions.md`.

## Tracks

**Track A** is the causal test. Three beacon nodes, identical except for the sidecar:

- `control-1`: consensus client + execution client, Gossipsub only
- `control-2`: the same as `control-1` (A/A calibration)
- `treatment`: the same clients plus the Optimum Gateway sidecar

Same client builds, same region, same instance type, same peer-count limits. Preferred layout is one host so the paired delta does not depend on clock sync. If the arms are on different hosts, chrony runs on each and `node_health.clock_offset_ms` is logged continuously.

For each block, `delta = t_arrival_control - t_arrival_treatment`. Slot start is `genesis_time + slot * seconds_per_slot` from that node's beacon API (`/eth/v1/beacon/genesis` and the spec), not from Optimum. Arrival time is when that node's consensus client first reports the block. The SSE topic used is recorded on the row. See `docs/decisions.md` for which topics were checked on the public Hoodi endpoint, and which client is still unchecked.

The A/A distribution (`control-1` minus `control-2`) is published on the same view as every A/B result. If it is not centered near zero, the harness is biased and A/B results are not used. The ±5 ms median bound in the Phase 2 acceptance text is a starting threshold; the value actually used is written here after the first A/A run, not before.

After the first window, the sidecar moves to a different physical node (crossover). Hoodi is for debugging the harness. Mainnet is the only network whose results can be stated as mainnet results.

**Track B** is observational. Partner validators versus matched non-partners, and each partner versus itself around a public rollout date. Design is difference-in-differences with a parallel-trends check, a placebo on fake rollout dates, and leave-one-operator-out. Cohort rows store `source_url` and `retrieved_at`. Unknown rollout dates stay unknown. Change-point detection is a sanity check, never the only date source.

Track B charts always carry this caveat: partners are large operators who may change other things at the same time. Track B is correlation. Track A is the causal comparison.

**Track C** is optional. It compares our series to a public propagation dataset such as Xatu. It is not started until the dataset's current public access is checked. Optimum's Hub is a claims archive, not an input to our estimators.

## What we will not treat as our measurement

Optimum's Hub "× faster" number is a fleet propagation-spread ratio (p95 minus first arrival), compared with Xatu sentries. H1 is a paired delay from slot start on our own consensus clients. H2 is whether our treatment node sees the block before our control nodes. Those are defined side by side in `docs/decisions.md`. Publishing a Hub number next to ours is required. Calling them the same quantity is not.

Some Hub per-slot charts are a fixture generated from a target speed-up. The snapshot note in `data/snapshots/optimum_claims/` records that. Those points are not observations.

## Statistics

- Report n, median, p90, and p99. The mean may be shown with them. A mean alone is not a result.
- Track A uses a paired test (Wilcoxon signed-rank or a paired bootstrap) and bootstrap confidence intervals.
- Missed slots and reorged blocks are excluded with one rule, and the count excluded is reported.
- Outliers are not removed unless a rule was written down before the run and the removal is logged.
- Analysis windows are fixed before the run. Every window that was collected is published.
- Faster propagation (H1, H2) and higher rewards (H3–H5) are separate tests.
- Every chart shows n, the time window, the data source, and how to reproduce it.

## Storage

TimescaleDB holds the tables in `infra/compose/timescaledb/init/001_schema.sql`. Timestamps are `timestamptz` (UTC). `gateway_version` is on the measurement tables and is null when that arm has no sidecar. The plan's result-window column is split into `window_start` and `window_end` because `WINDOW` is reserved in PostgreSQL. `p99_ms` is on `results_paired_delta` because the rule above requires it.

Prometheus is for live process metrics. Grafana is provisioned from this repo. Phase 1 adds one delay dashboard. The Track A and Track B dashboards are still later.

`head_events.block_root` is stored so a repeated head for the same root can be deduplicated. The plan's original column list did not include it.

## Phase 1 collector

The collector subscribes to one Beacon API. The running deployment uses the public Hoodi endpoint `https://beacon.hoodi.ethpandaops.io` and labels every row `node_id=hoodi-public-1`, `arm=observer`. That arm is not `control-1`, `control-2`, or `treatment`. Rows from it are pipeline data. They are not evidence for H1–H6.

`seen_at_utc` is the collector's receipt time, UTC, millisecond precision from `time.Now()`. It is not a timestamp inside the SSE payload. On this public endpoint the receipt time includes the path from ethPandaOps to the collector. A Track A delay has to be taken by a collector on the same host as the consensus client. This deployment does not do that.

Slot start is `genesis_time + slot * SECONDS_PER_SLOT` from `/eth/v1/beacon/genesis` and `/eth/v1/config/spec`. Delay is `seen_at_utc - slot_start` in milliseconds, including when it is negative. Negative delays are stored and logged. They are not dropped.

`block` and `block_gossip` share one dedup key: `(node_id, block_root)`. The first topic is the stored arrival. A later event for that root is counted as a duplicate and is not inserted. `head` uses the same key shape in `head_events` and is not discarded because a block event for that root was already stored. On 2026-10-22 the public Hoodi stream emitted `block_gossip`, then `head`, then `block` for one slot, so the stored arrival topic on that stream is `block_gossip` when the flag is on.

`BLOCK_GOSSIP` defaults off in the process. The Compose file turns it on for this endpoint because that stream was observed emitting the topic. Client name and version come from the SSE response headers `X-Dugtrio-Endpoint-Type` and `X-Dugtrio-Endpoint-Version` when those headers are present, otherwise from `/eth/v1/node/version`. The endpoint is load-balanced, so the client can change between connections. A change is logged, and later rows use the new version. `gateway_version` is null.

Clock offset is collector time minus the HTTP `Date` header, logged and exported as `withwithout_clock_offset_milliseconds`. It is not subtracted from delay. `Date` is one-second resolution and may belong to the load balancer rather than the consensus client's clock.

Duplicates, invalid payloads, and store errors are logged and counted on `withwithout_events_total`. A failed insert forgets the dedup key so a later event for that root can be stored. The beacon does not replay the failed event itself.

## What is still not running

No owned beacon nodes, no Gateway sidecar, no treatment data, no analysis jobs, no verdicts. The 24-hour Hoodi soak in the Phase 1 acceptance text starts when this process stays up. It is not claimed here.
