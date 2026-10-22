# withwithout

Independent check of Optimum's published claims about mump2p, by comparing Ethereum consensus nodes **with** the Optimum Gateway sidecar to nodes **without** it.

The hypotheses are frozen in [`docs/hypotheses.md`](docs/hypotheses.md). How the three evidence tracks work is in [`docs/methodology.md`](docs/methodology.md). The phased build plan is in [`docs/PLAN.md`](docs/PLAN.md). Disclosures are in [`docs/independence.md`](docs/independence.md).

This checkout is **Phase 1**. `make up` runs TimescaleDB, Prometheus, Grafana, and a Go collector against the public Hoodi beacon at `https://beacon.hoodi.ethpandaops.io`. Rows are labeled `arm=observer`. They are not a with/without comparison and they are not a verdict on H1–H6.

```bash
make up      # Grafana on :3000, Prometheus on :9090, TimescaleDB on :5432
make down
make lint
make test
```

Local defaults and the credential variable names are in `.env.example`. Do not commit a real API key.

Grafana's "Phase 1 block delay" dashboard is the live histogram. The 24-hour no-gap soak is still open. Phase 2, the three-node harness, waits until you ask for it.
