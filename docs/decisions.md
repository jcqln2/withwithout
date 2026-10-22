# Phase 0 decisions and open items

Retrieved 2026-10-22 from public docs. Nothing here is a measurement of mump2p. Items marked **unconfirmed** are assumptions Phase 2 must not treat as facts.

## 1. Gateway access

The current Gateway guide says the sidecar ships as Docker, authenticates with `OPT_API_KEY` (`ogw_…`) or `OPT_JOIN_KEY` (`ojk_…`), and requires a `gateway_cluster_id` assigned during onboarding. Chain and validator scope come from the credential, not from YAML. Auth is `https://auth.getoptimum.io`.

Sources: [Optimum Gateway user guide](https://getoptimum.github.io/optimum-gateway/), [Quick start](https://getoptimum.github.io/optimum-gateway/versions/latest/quick-start), [Configuration](https://getoptimum.github.io/optimum-gateway/versions/latest/configuration).

**Unconfirmed:** no credential has been requested, and no written no-veto agreement exists. See `docs/independence.md`. Track B and the public-claims archive do not need a credential.

`remote_push_enable` sends gateway logs and metrics to Optimum. This project will leave it off unless `docs/independence.md` is updated first. Our results come from the beacon API and host metrics, not from Optimum telemetry.

## 2. Does the treatment arm have to be a staked validator?

Public docs describe the Gateway as a sidecar to a consensus client. It does not take validator keys, and it re-forwards inbound beacon blocks from mump2p to the local CL in partner mode. Outbound attestation forwarding is limited to the validator list on the API key.

**Working assumption, unconfirmed by Optimum:** a synced beacon node that is not proposing or attesting is enough for H1, H2, and H6, if Optimum will issue a credential for that node. H3–H5 are Track B, or they need owned validators or a cooperating operator. Do not start Phase 2 node provisioning on this assumption until Optimum confirms it in writing.

## 3. Consensus client

The Gateway guide's "current stable" list, retrieved 2026-10-22 from [Connecting your CL client](https://getoptimum.github.io/optimum-gateway/versions/latest/cl-clients):

| Client | Version named in that guide |
|--------|-----------------------------|
| Prysm | v7.2.1 |
| Lighthouse | v8.2.3 |
| Teku | 26.9.1 |
| Nimbus | v26.10.0 |
| Lodestar | v1.49.0 |

**Provisional choice for Phase 2: Lighthouse v8.2.3 on every arm.** The guide documents its peering flags (`--boot-nodes` plus `--trusted-peers`) and says Lighthouse does not re-dial a restarted gateway, so `direct_cl_peers` is required. Re-read that page before pinning the image. All three arms use the same CL build.

The execution client is not chosen. All arms must use the same EL client and version. Pick it when the Phase 2 machine is budgeted.

**Unverified assumption:** which Beacon API SSE topic is the first observation of a block. The guide does not specify `block` versus `block_gossip` for Lighthouse. Phase 1 must record the topic list the pinned client actually emits, and must not guess.

## 4. Hardware budget, not a purchase

Sources retrieved 2026-10-22:

- [Lighthouse installation](https://lighthouse-book.sigmaprime.io/installation.html): one beacon node plus an execution client, modest validator set: 4 cores, 32 GB RAM, 2 TB SSD, 100 Mb/s down / 20 Mb/s up. Lighthouse says 16 GB is tight.
- [Nethermind system requirements](https://docs.nethermind.io/get-started/system-requirements/): mainnet 16 GB and 4 cores for the EL alone; at least 2 TB disk as of their October 2024 note, NVMe, ≥10k IOPS. Their page's disk figure is older than the launchpad note below.
- [Ethereum launchpad checklist](https://launchpad.ethereum.org/en/checklist): as of the page's February 2025 note, 2 TB SSD minimum and 4 TB recommended; 32 GB RAM minimum and 64 GB recommended.

Three full nodes do not share a chain database. A same-host mainnet A/B/A rig is on the order of three execution databases plus three beacon databases. Budget at least 64 GB RAM and 6 TB NVMe before buying, then re-check current client docs. Hoodi is the debug network and is smaller; still size the disk from a current Hoodi sync report before Phase 2. Same-host remains the preferred clock design. If the disk does not fit, separate hosts require chrony and continuous offset logging.

## 5. What "first" and "2–4×" mean in Optimum's own materials

These are different statistics. H1 and H2 keep the definitions in `docs/hypotheses.md`. A Hub number is not a result for those hypotheses unless the definitions match. They do not.

| Quantity | Whose definition | What it measures |
|----------|------------------|------------------|
| H1 primary metric | This project | Per node, CL event time minus slot start (`genesis_time + slot * seconds_per_slot`). Paired delta across arms. Median, p90, p99. |
| H2 primary metric | This project | Share of blocks where the treatment CL reports the block before the control CLs. |
| Propagation spread | Hub / console report | Per slot, p95 arrival minus first arrival, then averaged. Proposer build time is excluded. This is the figure they multiply into "× faster". |
| Slot arrival | Hub / console report | Per slot, p95 of arrivals measured from slot start, then averaged. Includes proposer build time, so the ratio is much closer to 1. |
| Slots accelerated | Hub / console report | Share of slots where mump2p was first versus an Xatu sentry reference, across their gateway fleet. Not one treatment node versus two matched controls. |
| Gateway `t_mum_seen_ms` / `t_eth_seen_ms` | [Gateway metrics methodology](https://getoptimum.github.io/optimum-gateway/versions/v1.3.2/metrics-methodology) | Timestamps inside the gateway, not the CL beacon API. Their own doc says raw libp2p-versus-mump2p deltas are debug signals, not validator-outcome KPIs. |

The live console payload also reports a mean per-slot lead and a CI (`lead_ms`, `lead_ci_ms`). That is their fleet-versus-Xatu lead, not our paired CL delta.

## 6. Hub data access

`https://hub.getoptimum.xyz/` is a dashboard UI. A fetch of the HTML on 2026-10-22 returned a shell with no measurements and no JSON API on that host.

The dashboard's script loads fleet performance from:

`https://console.getoptimum.xyz/api/public/network-performance`

That URL returned HTTP 200 `application/json` at 2026-10-22T19:12:18Z. The body is stored at `data/snapshots/optimum_claims/2026-10-22T1912Z-network-performance.json`. `generated_at` inside the body is `2026-10-22T19:12:17.849Z`.

The same Hub script, read the same day and not vendored here, also contains:

- `STATIC_PROPAGATION`, a compiled fallback used before the API responds and if it fails. It is a 7-day mainnet run labeled Aug 27–Sep 2, 2026 (spread 136.9 ms versus 540.0 ms, factor 3.95, slots accelerated 90.8%, 315 gateways, 173 Xatu sentries). A non-executing fetch of the Hub can show these numbers even when the API is serving a newer window.
- A per-slot `TELEMETRY` fixture whose comment says the mump2p arm is derived from a target multiple (7.5×) and "nothing here is measured." That fixture feeds the Stats page and the fallback per-slot card. Do not archive those chart points as Optimum measurements.

Track C stays optional. Xatu is the baseline Optimum already cites. Using it as an independent cross-check is reasonable only with a separately obtained Xatu extract, not by republishing their comparison.

## Numbers in the 2026-10-22 API snapshot

Live window in that file: mainnet, 2026-10-20T06:00:33Z through 2026-10-22T18:00:00Z (60 hours). These are Optimum's published figures, not ours.

| Metric in the payload | Value |
|-----------------------|-------|
| Observers | 300 gateways, 165 Xatu sentries |
| Slots measured / spread slots compared | 17,966 / 17,962 |
| Spread (mump2p vs gossipsub), ms | 134.3 vs 426.7 |
| Spread factor | 3.18 |
| Arrival from slot start, p95 mean, ms | 1,977.4 vs 2,249.9 |
| Slots accelerated | 87.9% |
| Lead | 272 ms, CI ± 5.5 ms |
| Percentile lead p50 / p75 / p80 / p95 | 59 / 114 / 124 / 272 ms, each over 17,962 slots |

The headline object in this payload has no "slots over 4s" fields. The static fallback does. Do not mix the two windows.
