// Package store writes measurement rows to TimescaleDB.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jcqln2/withwithout/collector/internal/ingest"
)

// Store is a Postgres connection pool.
type Store struct {
	Pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() {
	s.Pool.Close()
}

// LoadArrivalRoots returns block roots already stored for nodeID.
func (s *Store) LoadArrivalRoots(ctx context.Context, nodeID string) ([]string, error) {
	return s.load(ctx, `SELECT DISTINCT block_root FROM block_arrivals WHERE node_id = $1`, nodeID)
}

// LoadHeadRoots returns block roots already stored as heads for nodeID.
func (s *Store) LoadHeadRoots(ctx context.Context, nodeID string) ([]string, error) {
	return s.load(ctx, `SELECT DISTINCT block_root FROM head_events WHERE node_id = $1`, nodeID)
}

func (s *Store) load(ctx context.Context, query, nodeID string) ([]string, error) {
	rows, err := s.Pool.Query(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var root string
		if err := rows.Scan(&root); err != nil {
			return nil, err
		}
		out = append(out, root)
	}
	return out, rows.Err()
}

// InsertArrival inserts the first row for this node and block root.
func (s *Store) InsertArrival(ctx context.Context, row ingest.Row) (bool, error) {
	tag, err := s.Pool.Exec(ctx, `
		INSERT INTO block_arrivals (
			seen_at_utc, slot, block_root, node_id, arm, topic,
			delay_ms_from_slot_start, client, client_version, gateway_version
		)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, '')
		WHERE NOT EXISTS (
			SELECT 1 FROM block_arrivals WHERE node_id = $4 AND block_root = $3
		)`,
		row.SeenAt, row.Slot, row.BlockRoot, row.NodeID, row.Arm, row.Topic,
		row.DelayMS, row.Client, row.ClientVersion, row.GatewayVersion,
	)
	if err != nil {
		return false, fmt.Errorf("insert arrival: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// InsertHead inserts the first head row for this node and block root.
func (s *Store) InsertHead(ctx context.Context, row ingest.Row) (bool, error) {
	tag, err := s.Pool.Exec(ctx, `
		INSERT INTO head_events (
			seen_at_utc, slot, block_root, node_id, arm, delay_ms, client, client_version, gateway_version
		)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, '')
		WHERE NOT EXISTS (
			SELECT 1 FROM head_events WHERE node_id = $4 AND block_root = $3
		)`,
		row.SeenAt, row.Slot, row.BlockRoot, row.NodeID, row.Arm, row.DelayMS, row.Client, row.ClientVersion, row.GatewayVersion,
	)
	if err != nil {
		return false, fmt.Errorf("insert head: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
