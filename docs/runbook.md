# Runbook

`make up` starts TimescaleDB, Prometheus, Grafana, and the Phase 1 collector. The collector follows the public Hoodi beacon API. It does not run a consensus client, and it does not join mump2p.

## Start and stop

From the repo root:

```bash
make up
make down
```

`make lint` checks that Compose parses and that the collector is gofmt-clean and `go vet` clean. `make test` runs the scaffold check and `go test ./...` in `collector/`.

| Service | URL | Local default login |
|---------|-----|---------------------|
| Grafana | http://localhost:3000 | `admin` / `admin` |
| Prometheus | http://localhost:9090 | none |
| TimescaleDB | `localhost:5432`, database `verifier` | `verifier` / `verifier` |
| Collector metrics | http://localhost:8080/metrics | none |

Override ports and passwords with a gitignored `.env`. The variable names are in `.env.example`. Grafana's Postgres datasource reads `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB` from the container environment.

The database schema is applied only when the Timescale volume is created. After a schema change during this empty phase, reset it with `docker compose -f infra/compose/docker-compose.yml down -v` and `make up`. Do not do that once real rows exist.

## Phase status

| Phase | State |
|-------|--------|
| 0 Scaffold, hypotheses, monitoring stack | This repo |
| 1 Slot clock and one-node collector | Running against the public Hoodi beacon when `make up` is up. Not yet a 24 h gap-free soak. |
| 2 A/B/A harness | Not started |
| 3 Cohort ingest | Not started |
| 4 Difference-in-differences | Not started |
| 5 Dashboards | Not started |
| 6 Mainnet run and publication | Not started |

Work one phase at a time. The prompt for each phase is in `docs/PLAN.md` section 8.

## Before any treatment data

1. `docs/hypotheses.md` is already committed. Do not edit the table.
2. Request a Hoodi Gateway credential and the written terms in `docs/independence.md`.
3. Confirm whether an unstaked beacon node is enough (see `docs/decisions.md`).
4. Re-check the Lighthouse and Gateway versions, and size disk for three nodes, before buying hardware.

`data/rollouts.csv` and `data/cohorts/` stay empty until Phase 3. Every cohort row needs a source URL and a retrieval time. `data/snapshots/optimum_claims/` is the archive of Optimum's published numbers, not an input to the estimators.
