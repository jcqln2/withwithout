#!/usr/bin/env bash
# Phase 0 scaffold check. Fails if a required file is missing or Compose will not parse.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

required=(
  docs/PLAN.md
  docs/hypotheses.md
  docs/methodology.md
  docs/independence.md
  docs/decisions.md
  docs/runbook.md
  infra/compose/docker-compose.yml
  infra/compose/prometheus/prometheus.yml
  infra/compose/timescaledb/init/001_schema.sql
  dashboards/grafana/provisioning/datasources/datasources.yml
  data/rollouts.csv
  .env.example
  .cursor/rules/project.mdc
)

for path in "${required[@]}"; do
  if [[ ! -f "$path" ]]; then
    echo "missing $path" >&2
    exit 1
  fi
done

for id in H1 H2 H3 H4 H5 H6; do
  if ! grep -q "$id" docs/hypotheses.md; then
    echo "hypotheses.md is missing $id" >&2
    exit 1
  fi
done

if ! grep -q "pre-registered" docs/hypotheses.md; then
  echo "hypotheses.md must say the claims are pre-registered" >&2
  exit 1
fi

docker compose -f infra/compose/docker-compose.yml config --quiet
echo "scaffold ok"
