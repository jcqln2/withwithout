# Methodology

This file describes how measurements will be made. Phase 0 does not collect them. When a later phase changes a definition, update this file in the same commit.

The hypotheses are frozen in `docs/hypotheses.md`. The build phases are in `docs/PLAN.md`. Resolved and still-open setup questions are in `docs/decisions.md`.

## Tracks

**Track A** is the causal test. Three beacon nodes, identical except for the sidecar:

- `control-1`: consensus client + execution client, Gossipsub only
- `control-2`: the same as `control-1` (A/A calibration)
- `treatment`: the same clients plus the Optimum Gateway sidecar

Same client builds, same region, same instance type, same peer-count limits. Preferred layout is one host so the paired delta does not depend on clock sync. If the arms are on different hosts, chrony runs on each and `node_health.clock_offset_ms` is logged continuously.

For each block, `delta = t_arrival_control - t_arrival_treatment`. Slot start is `genesis_time + slot * seconds_per_slot` from that node's beacon API (`/eth/v1/beacon/genesis` and the spec), not from Optimum. Arrival time is when that node's consensus client first reports the block. The SSE topic used is recorded on the row. Phase 1 has to confirm the topic on the pinned client before any delay is interpreted. See the unverified assumption in `docs/decisions.md`.

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

Prometheus is for live process metrics. Grafana is provisioned from this repo. No dashboard panels exist in Phase 0.

## What Phase 0 does not do

No collector, no beacon nodes, no treatment data, no analysis jobs, no verdicts.
