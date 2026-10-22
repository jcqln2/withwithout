-- Pre-registered tables from docs/PLAN.md section 6.
-- All timestamps are timestamptz (UTC). Millisecond precision is native.
-- gateway_version is null on arms with no sidecar.
-- PostgreSQL reserves WINDOW, so the plan's "window" column is window_start/window_end.
-- A Timescale unique index must include the partition column, so block-level
-- dedup is enforced by the collector (Phase 1), not by a primary key on block_root alone.
-- p99_ms is included because the statistical rules require p99, not only p90.

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE block_arrivals (
    seen_at_utc timestamptz NOT NULL,
    slot bigint NOT NULL,
    block_root text NOT NULL,
    node_id text NOT NULL,
    arm text NOT NULL,
    topic text NOT NULL,
    delay_ms_from_slot_start bigint,
    client text,
    client_version text,
    gateway_version text
);

SELECT create_hypertable('block_arrivals', 'seen_at_utc');

CREATE INDEX block_arrivals_block_idx
    ON block_arrivals (block_root, node_id, topic, seen_at_utc DESC);

CREATE TABLE head_events (
    seen_at_utc timestamptz NOT NULL,
    slot bigint NOT NULL,
    block_root text NOT NULL,
    node_id text NOT NULL,
    arm text NOT NULL,
    delay_ms bigint,
    client text,
    client_version text,
    gateway_version text
);

SELECT create_hypertable('head_events', 'seen_at_utc');

CREATE TABLE node_health (
    ts timestamptz NOT NULL,
    node_id text NOT NULL,
    arm text NOT NULL,
    peers integer,
    cpu_pct double precision,
    mem_bytes bigint,
    net_in_bytes bigint,
    net_out_bytes bigint,
    clock_offset_ms double precision,
    client text,
    client_version text,
    gateway_version text
);

SELECT create_hypertable('node_health', 'ts');

CREATE TABLE slots (
    slot bigint PRIMARY KEY,
    slot_start_utc timestamptz,
    proposer_index bigint,
    block_root text,
    missed boolean,
    builder_type text
);

CREATE TABLE validator_epoch (
    epoch bigint NOT NULL,
    validator_index bigint NOT NULL,
    head_ok boolean,
    target_ok boolean,
    source_ok boolean,
    attestation_missed boolean,
    inclusion_delay integer,
    cl_reward_gwei bigint,
    PRIMARY KEY (epoch, validator_index)
);

CREATE TABLE validators (
    validator_index bigint PRIMARY KEY,
    pubkey text,
    operator text,
    source_url text NOT NULL,
    retrieved_at timestamptz NOT NULL
);

CREATE TABLE operator_rollouts (
    operator text PRIMARY KEY,
    mainnet_start date,
    start_epoch bigint,
    source_url text,
    confidence text NOT NULL
);

CREATE TABLE claims_snapshot (
    ts timestamptz NOT NULL,
    source_url text NOT NULL,
    metric text NOT NULL,
    value double precision,
    raw_text text,
    PRIMARY KEY (ts, source_url, metric)
);

CREATE TABLE results_paired_delta (
    run_id text NOT NULL,
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    arm_pair text NOT NULL,
    n bigint NOT NULL,
    median_ms double precision,
    p90_ms double precision,
    p99_ms double precision,
    mean_ms double precision,
    ci_low double precision,
    ci_high double precision,
    PRIMARY KEY (run_id, arm_pair)
);

CREATE TABLE results_did (
    run_id text NOT NULL,
    metric text NOT NULL,
    estimate double precision,
    ci_low double precision,
    ci_high double precision,
    n_treated bigint,
    n_control bigint,
    parallel_trends_p double precision,
    PRIMARY KEY (run_id, metric)
);
