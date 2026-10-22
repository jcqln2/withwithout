# withwithout

Independent check of Optimum's published claims about mump2p, by comparing Ethereum consensus nodes **with** the Optimum Gateway sidecar to nodes **without** it.

The hypotheses are frozen in [`docs/hypotheses.md`](docs/hypotheses.md). How the three evidence tracks work is in [`docs/methodology.md`](docs/methodology.md). The phased build plan is in [`docs/PLAN.md`](docs/PLAN.md). Disclosures are in [`docs/independence.md`](docs/independence.md).

This checkout is **Phase 0**: repo layout, those docs, and an empty TimescaleDB + Prometheus + Grafana stack. There is no collector and no treatment data.

```bash
make up      # Grafana on :3000, Prometheus on :9090, TimescaleDB on :5432
make down
make lint
make test
```

Local defaults and the credential variable names are in `.env.example`. Do not commit a real API key.

Next step, when you want it, is Phase 1 from `docs/PLAN.md`: a Go collector for one beacon node. Stop there until its acceptance criteria pass.
