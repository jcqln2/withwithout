// Package ingest turns one SSE event into a stored row.
package ingest

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/jcqln2/withwithout/collector/internal/dedup"
	"github.com/jcqln2/withwithout/collector/internal/metrics"
	"github.com/jcqln2/withwithout/collector/internal/slot"
)

// Row is one measurement. GatewayVersion is empty on this single public node.
type Row struct {
	Slot           int64
	BlockRoot      string
	NodeID         string
	Arm            string
	Topic          string
	SeenAt         time.Time
	DelayMS        int64
	Client         string
	ClientVersion  string
	GatewayVersion string
}

// Store writes arrival and head rows. Insert returns true when a new row was written.
type Store interface {
	InsertArrival(ctx context.Context, row Row) (bool, error)
	InsertHead(ctx context.Context, row Row) (bool, error)
}

// Handler applies the slot clock and the dedup rules.
type Handler struct {
	NodeID        string
	Arm           string
	Clock         slot.Clock
	Arrivals      *dedup.Set
	Heads         *dedup.Set
	Store         Store
	Metrics       *metrics.Metrics
	Log           *slog.Logger
	Client        string
	ClientVersion string
}

// Event handles one SSE message. seen is the collector's receipt time in UTC.
//
// block and block_gossip share one dedup key per node and block root. The first
// topic is the arrival. head is stored on its own and is not dropped because a
// block event for the same root already arrived.
func (h *Handler) Event(ctx context.Context, topic, data string, seen time.Time) error {
	if h.Log == nil {
		h.Log = slog.Default()
	}
	seen = seen.UTC()
	slotNum, root, ok := parse(data)
	if !ok || root == "" {
		h.Metrics.Events.WithLabelValues(topicOrUnknown(topic), "invalid").Inc()
		h.Log.Warn("invalid event", "topic", topic, "data", data)
		return nil
	}
	delay := h.Clock.DelayMS(slotNum, seen)
	row := Row{
		Slot:          int64(slotNum),
		BlockRoot:     root,
		NodeID:        h.NodeID,
		Arm:           h.Arm,
		Topic:         topic,
		SeenAt:        seen,
		DelayMS:       delay,
		Client:        h.Client,
		ClientVersion: h.ClientVersion,
	}

	switch topic {
	case "block", "block_gossip":
		return h.storeArrival(ctx, row)
	case "head":
		return h.storeHead(ctx, row)
	default:
		h.Metrics.Events.WithLabelValues(topicOrUnknown(topic), "invalid").Inc()
		h.Log.Warn("unexpected topic", "topic", topic, "slot", slotNum, "block_root", root)
		return nil
	}
}

func (h *Handler) storeArrival(ctx context.Context, row Row) error {
	key := h.NodeID + "|" + row.BlockRoot
	if h.Arrivals.Seen(key) {
		h.Metrics.Events.WithLabelValues(row.Topic, "duplicate").Inc()
		h.Log.Info("duplicate block", "topic", row.Topic, "slot", row.Slot, "block_root", row.BlockRoot, "node_id", h.NodeID)
		return nil
	}
	inserted, err := h.Store.InsertArrival(ctx, row)
	if err != nil {
		h.Arrivals.Forget(key)
		h.Metrics.Events.WithLabelValues(row.Topic, "store_error").Inc()
		h.Log.Error("store arrival", "err", err, "slot", row.Slot, "block_root", row.BlockRoot, "topic", row.Topic)
		return err
	}
	if !inserted {
		h.Metrics.Events.WithLabelValues(row.Topic, "duplicate").Inc()
		h.Log.Info("duplicate block already stored", "topic", row.Topic, "slot", row.Slot, "block_root", row.BlockRoot)
		return nil
	}
	h.Metrics.Events.WithLabelValues(row.Topic, "stored").Inc()
	h.Metrics.DelayMS.Observe(float64(row.DelayMS))
	if row.DelayMS < 0 {
		h.Log.Warn("negative delay stored", "topic", row.Topic, "slot", row.Slot, "block_root", row.BlockRoot, "delay_ms", row.DelayMS)
	}
	h.Log.Info("stored block", "topic", row.Topic, "slot", row.Slot, "block_root", row.BlockRoot, "delay_ms", row.DelayMS, "client", row.Client, "client_version", row.ClientVersion)
	return nil
}

func (h *Handler) storeHead(ctx context.Context, row Row) error {
	key := h.NodeID + "|" + row.BlockRoot
	if h.Heads.Seen(key) {
		h.Metrics.Events.WithLabelValues("head", "duplicate").Inc()
		h.Log.Info("duplicate head", "slot", row.Slot, "block_root", row.BlockRoot, "node_id", h.NodeID)
		return nil
	}
	inserted, err := h.Store.InsertHead(ctx, row)
	if err != nil {
		h.Heads.Forget(key)
		h.Metrics.Events.WithLabelValues("head", "store_error").Inc()
		h.Log.Error("store head", "err", err, "slot", row.Slot, "block_root", row.BlockRoot)
		return err
	}
	if !inserted {
		h.Metrics.Events.WithLabelValues("head", "duplicate").Inc()
		h.Log.Info("duplicate head already stored", "slot", row.Slot, "block_root", row.BlockRoot)
		return nil
	}
	h.Metrics.Events.WithLabelValues("head", "stored").Inc()
	h.Log.Info("stored head", "slot", row.Slot, "block_root", row.BlockRoot, "delay_ms", row.DelayMS)
	return nil
}

func parse(data string) (uint64, string, bool) {
	var payload struct {
		Slot  string `json:"slot"`
		Block string `json:"block"`
	}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return 0, "", false
	}
	slotNum, err := strconv.ParseUint(payload.Slot, 10, 64)
	if err != nil {
		return 0, "", false
	}
	return slotNum, payload.Block, true
}

func topicOrUnknown(topic string) string {
	if topic == "" {
		return "unknown"
	}
	return topic
}
