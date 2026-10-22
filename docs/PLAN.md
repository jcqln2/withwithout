# Optimum Independent Verifier: Engineering Plan

**Purpose:** Build an independent, reproducible measurement system and public dashboard that tests Optimum's published claims about mump2p (RLNC-based gossip for Ethereum), by comparing validators/nodes **with** the Optimum Gateway sidecar against those **without** it.

**How to use this with Cursor:** Section 12 is in `.cursor/rules/project.mdc`. Work one phase at a time (Section 8). Paste the phase's "Cursor prompt" and stop at the acceptance criteria. Don't ask for the whole system in one prompt.

The repo directory is `withwithout`. The layout below is the layout of this repo (the plan's `optimum-verifier/` name is the product, not the directory).

---

## 1. Claims under test (pre-registered hypotheses)

Written into `docs/hypotheses.md`. Committed before any treatment data. Do not change them afterward. See that file for the freeze rule and the H1–H6 table.

Decision rule: for each hypothesis, report effect size, 95% CI, and a plain verdict: **supported / not supported / inconclusive**. Inconclusive is a valid result.

## 2. Methodology: three evidence tracks

### Track A: Controlled A/B nodes (primary, strongest evidence)

Run beacon nodes that are identical except for the sidecar:

- **control-1**: CL client + EL client, Gossipsub only
- **control-2**: identical to control-1 (A/A calibration; measures the noise floor)
- **treatment**: same CL client + EL client **plus** Optimum Gateway sidecar

Rules:

1. Same CL client and version, same EL client, same region, same instance type, same peer-count limits.
2. **Preferred:** run all arms on the **same host** (or same VM family on one provider) so they share a clock. Then the per-block delta `t_control - t_treatment` is clock-independent. If separate hosts are needed, run chrony and log clock offset continuously.
3. **Paired comparison:** for each block (slot), compute `delta = t_arrival_control - t_arrival_treatment`. Pairing removes variance from proposer publish time, builder latency, and block size.
4. **A/A test first:** `control-1` vs `control-2` delta distribution must be centered near zero. If it isn't, the harness is biased; fix it before trusting any A/B result.
5. **Crossover:** after N days, swap which physical node runs the sidecar to rule out hardware/peer-set luck.
6. Start on **Hoodi testnet** to debug the harness cheaply, then repeat on **mainnet**. Do not publish testnet results as proof of mainnet behavior.

What to measure per arm (via the Beacon API, not by trusting Optimum's telemetry):

- Block arrival time: timestamp when each node's CL client first reports the block, minus slot start (`genesis_time + slot * SECONDS_PER_SLOT`). Prefer the `block_gossip` SSE topic if the client supports it, otherwise `block`. Verify per client and document which was used.
- Head update time (`head` event).
- Attestations: count of attestations seen per slot by a fixed deadline (e.g., 4 s into the slot) per arm. This is a stretch goal for Phase 2.
- Bandwidth: bytes in/out from node exporter / cAdvisor per container.
- Peer count, CPU, memory (for overhead claims).

### Track B: Passive cohort analysis from public chain data (needs no Optimum access)

Compare validators operated by Optimum partners against matched non-partners, and each partner against **itself before vs after** adoption.

- **Cohort definition:** map operator → validator indices from public sources (Optimum Hub partner list, Lido Node Operator Registry on-chain, Rated/beaconcha.in entity labels). Store the source and retrieval date for every mapping.
- **Rollout dates:** keep a table `operator_rollouts(operator, mainnet_start, source_url, confidence)`. Fill from public announcements; mark unknown dates as unknown. Run change-point detection as a sanity check, never as the sole source.
- **Design:** difference-in-differences. Treated = partner validators after rollout. Controls = matched non-partner operators (match on operator size, CL client if known, geography if known). Report the parallel-trends check.
- **Metrics (per validator-epoch):**
  - head / target / source vote correctness (from `/eth/v1/beacon/rewards/attestations/{epoch}` where head reward = 0 signals a missed/incorrect head)
  - missed attestations
  - missed proposals (proposer duties vs. blocks actually produced)
  - attestation inclusion delay
  - consensus-layer reward per epoch
- **Known confounders (must be stated on the dashboard):** partners are large, sophisticated operators who likely tune other things at the same time; Track B shows correlation, Track A shows causation. Always display the caveat next to Track B charts.

### Track C: Third-party network observation (optional cross-check)

Compare against independent public propagation datasets (e.g., ethPandaOps Xatu, ProbeLab/Hermes-based data) for block arrival-time distributions before and after Optimum's mainnet launch. Verify what is currently publicly available before committing to this track; if not, drop it. Do not depend on Optimum's own Hub for anything except displaying their claims side by side.

## 3. Dependencies and open questions

Resolved as far as public docs allow in `docs/decisions.md`. Still open: written Gateway access, and written confirmation that an unstaked beacon node can join the mesh.

## 4. Architecture

```
                       +--------------------------+
 Beacon API (SSE/REST) |  collector (Go)          |   Prometheus metrics
 control-1, control-2, +--------------------------+------------------+
 treatment   -------->|  - slot clock             |                  |
                       |  - SSE listeners         |                  v
 Public chain data     |  - REST pollers          |          +--------------+
 (own node, beaconcha  |  - backfill workers      |          |  Prometheus  |
 .in/Rated adapters) ->+------------+-------------+          +------+-------+
                                    |                               |
                                    v                               v
                          +-------------------+            +--------------+
                          | ClickHouse or     |<-----------+   Grafana    |
                          | TimescaleDB       |            |  dashboards  |
                          | (raw events)      |            +------^-------+
                          +---------+---------+                   |
                                    |                             |
                                    v                             |
                          +-------------------+   results tables  |
                          | analysis (Python) +-------------------+
                          | pandas/statsmodels|
                          +-------------------+
```

**Stack:**

- Collector: **Go** (long-running SSE streams, strong concurrency). Expose Prometheus metrics for live panels.
- Storage: **TimescaleDB (Postgres)** by default for simplicity; ClickHouse only if Track B volume demands it.
- Analysis: **Python** (pandas, scipy, statsmodels) run as scheduled jobs; write to `results_*` tables.
- Dashboards: **Grafana** first. Optional later: a static public report site (Next.js) for the final write-up.
- Infra: Docker Compose for local; Terraform or plain scripts for the node VMs. Everything reproducible from the repo.

## 5. Repo layout

```
withwithout/
  README.md
  docs/
    PLAN.md
    hypotheses.md
    methodology.md
    independence.md
    decisions.md
    runbook.md
  infra/
    compose/
    terraform/
  collector/
    cmd/collector/
    internal/{beacon,slot,sse,store,metrics,backfill,adapters}/
  analysis/
    notebooks/
    verifier/{paired_delta.py,did.py,cohorts.py,stats.py}
    tests/
  dashboards/
    grafana/{provisioning,json}/
  data/
    cohorts/
    rollouts.csv
    snapshots/optimum_claims/
  .cursor/rules/
```

## 6. Data model (core tables)

Implemented in `infra/compose/timescaledb/init/001_schema.sql`.

```
block_arrivals(slot, block_root, node_id, arm, topic, seen_at_utc, delay_ms_from_slot_start, client, client_version, gateway_version)
head_events(slot, block_root, node_id, arm, seen_at_utc, delay_ms, client, client_version, gateway_version)
node_health(ts, node_id, arm, peers, cpu_pct, mem_bytes, net_in_bytes, net_out_bytes, clock_offset_ms, client, client_version, gateway_version)
slots(slot, slot_start_utc, proposer_index, block_root, missed bool, builder_type)
validator_epoch(epoch, validator_index, head_ok, target_ok, source_ok, attestation_missed, inclusion_delay, cl_reward_gwei)
validators(validator_index, pubkey, operator, source_url, retrieved_at)
operator_rollouts(operator, mainnet_start, start_epoch, source_url, confidence)
claims_snapshot(ts, source_url, metric, value, raw_text)
results_paired_delta(run_id, window_start, window_end, arm_pair, n, median_ms, p90_ms, p99_ms, mean_ms, ci_low, ci_high)
results_did(run_id, metric, estimate, ci_low, ci_high, n_treated, n_control, parallel_trends_p)
```

`gateway_version` and `p99_ms` are additions required by the statistical rules. `window` is split into `window_start` and `window_end` because `WINDOW` is reserved in PostgreSQL.

## 7. Statistical rules (non-negotiable)

- Report **median, p90, p99**, not only the mean. Propagation delays are heavy-tailed.
- Paired tests for Track A (Wilcoxon signed-rank or paired bootstrap). Bootstrap CIs everywhere.
- Exclude missed slots and reorged blocks consistently, and report how many were excluded.
- Never remove outliers without a logged, rule-based reason.
- Show the A/A result on the same page as every A/B result.
- Fix analysis windows in advance. No cherry-picking date ranges; publish all.
- Separate "faster" (H1/H2) from "earns more" (H3-H5). Faster propagation does not automatically imply higher rewards; test each link.
- Every chart shows `n` and the time window.

## 8. Phases

### Phase 0: Scaffolding and decisions (1-2 days)

Tasks: create repo/layout; write `hypotheses.md`, `methodology.md`, `independence.md`; resolve open questions in Section 3 (contact Optimum for Gateway access); set up Docker Compose skeleton with TimescaleDB, Prometheus, Grafana.

Acceptance: `docker compose up` brings up empty monitoring stack; docs committed.

**Cursor prompt:**

> Scaffold the repo per Section 5 of PLAN.md. Create a Docker Compose file with TimescaleDB, Prometheus, and Grafana (provisioned datasources). Add a Makefile with `up`, `down`, `lint`, `test`. No collector code yet.

### Phase 1: Slot clock + single-node arrival collector (3-4 days)

Tasks: Go collector connects to **one** Beacon API node; read genesis time and seconds-per-slot from `/eth/v1/beacon/genesis` and spec; subscribe to SSE `block`/`block_gossip` and `head`; compute delay from slot start; write to `block_arrivals`; expose Prometheus metrics; handle reconnects, duplicate events, and clock-offset logging.

Acceptance: on Hoodi, collector runs 24 h with no gaps; unit tests cover slot math and dedup; delay histogram appears in Grafana.

**Cursor prompt:**

> In `collector/`, implement a Go service that reads genesis_time and seconds_per_slot from a Beacon API URL, subscribes to the SSE `block` and `head` topics (with `block_gossip` as an optional topic flag), computes `delay_ms_from_slot_start`, deduplicates by block_root per node, and writes rows to the `block_arrivals` table. Add reconnect with backoff, Prometheus metrics (events seen, reconnects, delay histogram), and table-driven tests for slot math. Config via env vars. Do not add other features.

### Phase 2: A/B/A harness (1 week, includes node sync time)

Tasks: Compose files for control-1, control-2, treatment on the same host (or documented alternative); treatment adds the Optimum Gateway sidecar per its guide; collector monitors all three; add node_health collection (bandwidth, peers, CPU/mem, chrony offset); `analysis/verifier/paired_delta.py` computes per-block paired deltas.

Acceptance: A/A delta distribution median within ±5 ms (adjust threshold after seeing the data; document the choice); treatment confirmed receiving blocks via the mesh (check Gateway logs/metrics); 72 h of continuous data on Hoodi.

**Cursor prompt:**

> Extend the collector to monitor N named nodes (config list: node_id, arm, beacon_url). Create `analysis/verifier/paired_delta.py` that joins `block_arrivals` by block_root across arms and computes per-block deltas, median/p90/p99, paired bootstrap CI, and writes to `results_paired_delta`. Add pytest tests with synthetic data including missing arrivals and ties.

### Phase 3: Public-chain cohort pipeline (1-2 weeks)

Tasks: adapters for (a) own beacon node REST (rewards, duties, blocks) for forward ingest and (b) beaconcha.in/Rated-style backfill with rate limiting and caching; cohort builder that maps operators to validator indices with source URLs; load `operator_rollouts`; populate `validator_epoch` and `slots`.

Acceptance: reproducible rebuild of cohorts from sources; spot-check 20 random validators against an explorer; backfill resumable.

**Cursor prompt:**

> Add `collector/internal/adapters` with a `ChainDataSource` interface and two implementations: BeaconNodeSource and an explorer-API source with rate limiting, retries, and on-disk caching. Implement a job that, for a list of validator indices from `data/cohorts/*.csv`, fetches per-epoch attestation rewards and proposer duties and upserts into `validator_epoch` and `slots`. Make it idempotent and resumable.

### Phase 4: Difference-in-differences analysis (1 week)

Tasks: `did.py` with matched controls, event-study plot data (periods relative to rollout), parallel-trends test, bootstrap CIs; sensitivity run excluding each partner one at a time.

Acceptance: on synthetic data with a known injected effect, DiD recovers it; placebo test (fake rollout dates on non-partners) yields ~0 effect.

**Cursor prompt:**

> Implement `analysis/verifier/did.py`: build a validator-epoch panel from the DB, match controls on operator size bucket, estimate a two-way fixed-effects DiD for each metric, produce event-study coefficients, a parallel-trends test, and leave-one-operator-out sensitivity. Include a synthetic-data test with a known effect and a placebo test.

### Phase 5: Dashboards (1 week)

See Section 9. Provision via JSON in `dashboards/grafana/`.

Acceptance: dashboards load from provisioning on a clean `make up`; every panel shows n and time window; caveats visible.

**Cursor prompt:**

> Create Grafana dashboards as provisioned JSON per Section 9 of PLAN.md, reading from TimescaleDB and Prometheus. Use variables for window, arm pair, and operator. Add text panels for caveats.

### Phase 6: Mainnet run, review, and publication (2+ weeks of data)

Tasks: repeat Phases 2 on mainnet; crossover swap; run pre-registered analyses; write the report; give Optimum a factual-accuracy review window (no veto); publish the repo, raw data, and dashboards.

Acceptance: another engineer can reproduce all headline numbers from the repo and raw data.

## 9. Dashboard spec

**Dashboard 1: Head-to-head (Track A)**

- Block arrival delay distribution: control vs treatment (overlaid histograms, log-x option)
- Paired delta over time (median + p90 band), with the A/A line shown for reference
- % of blocks where treatment arrived first (vs. Optimum's ~90% claim, with their number displayed beside ours)
- Speedup factor (control median / treatment median) with CI vs. the "2-4x" claim
- Bandwidth in/out per arm; CPU/mem; peer counts
- Data quality strip: clock offset, missed events, nodes sync status

**Dashboard 2: Real-world cohort view (Track B)**

- Event study: head-vote correctness for partners relative to rollout epoch vs. matched controls
- Missed attestation and missed proposal rates, treated vs control, with CIs
- Reward per epoch delta (normalized), with the confounder warning
- Per-operator table: rollout date, source, n validators, effect estimate
- Placebo test result panel

**Dashboard 3: Claims scorecard**

- Table of H1-H6: Optimum's claim, our measured value, CI, verdict (supported/not supported/inconclusive), link to methodology and raw data
- Timestamped snapshots of Optimum Hub numbers vs. our measurements

**Global:** every panel has n, window, data source, and a "how to reproduce" link.

## 10. Independence and integrity checklist

Tracked in `docs/independence.md`.

- [ ] Pre-registered hypotheses committed before treatment data collected
- [ ] Optimum access terms (credentials, support) disclosed in `independence.md`; no payment from Optimum; no veto on findings
- [ ] Any funding or conflict of interest disclosed
- [ ] Raw data and code public; versions pinned (client versions, gateway image digest)
- [ ] A/A calibration published with the results
- [ ] Negative or null results published with equal prominence
- [ ] Optimum given a factual-error review window before publication, with all changes logged

## 11. Risks

| Risk | Mitigation |
|------|-----------|
| Gateway credentials delayed or refused | Track B and Track C deliver value without it; ask early in Phase 0 |
| Treatment node not representative of real validators | Say so; add a cooperating operator's data if possible |
| Clock error between hosts | Same-host arms; chrony + offset logging; paired deltas |
| Beacon API client differences in SSE semantics | Test per client; record client/version on every row |
| Partner rollout dates unknown | Track confidence; change-point check; leave-one-out sensitivity |
| Confounding in Track B | DiD + matching + placebo + clear caveats |
| Cost and disk growth (mainnet nodes) | Budget upfront; prune; document costs |
| Optimum changes its protocol/product mid-study | Record gateway version on every row; segment analysis by version |

## 12. Cursor rules

Saved as `.cursor/rules/project.mdc`.

## 13. Definition of done

1. A/A test passes and is published.
2. Mainnet A/B dataset of at least 14 days (suggest more) with crossover.
3. Cohort DiD results for every partner with a known rollout date.
4. Scorecard shows a verdict for H1-H6, each with CI and links to raw data.
5. A fresh clone can reproduce every headline number with `make reproduce`.
