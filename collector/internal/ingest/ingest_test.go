package ingest

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/jcqln2/withwithout/collector/internal/dedup"
	"github.com/jcqln2/withwithout/collector/internal/metrics"
	"github.com/jcqln2/withwithout/collector/internal/slot"
)

type memStore struct {
	arrivals []Row
	heads    []Row
	fail     bool
}

func (m *memStore) InsertArrival(_ context.Context, row Row) (bool, error) {
	if m.fail {
		return false, errors.New("db down")
	}
	for _, have := range m.arrivals {
		if have.NodeID == row.NodeID && have.BlockRoot == row.BlockRoot {
			return false, nil
		}
	}
	m.arrivals = append(m.arrivals, row)
	return true, nil
}

func (m *memStore) InsertHead(_ context.Context, row Row) (bool, error) {
	if m.fail {
		return false, errors.New("db down")
	}
	for _, have := range m.heads {
		if have.NodeID == row.NodeID && have.BlockRoot == row.BlockRoot {
			return false, nil
		}
	}
	m.heads = append(m.heads, row)
	return true, nil
}

func TestEventDedupAndDelay(t *testing.T) {
	genesis := time.Unix(1_742_213_400, 0).UTC()
	seen := genesis.Add(time.Duration(10)*12*time.Second + 1500*time.Millisecond)
	store := &memStore{}
	reg := prometheus.NewRegistry()
	h := &Handler{
		NodeID:        "hoodi-public-1",
		Arm:           "observer",
		Clock:         slot.Clock{Genesis: genesis, SecondsPerSlot: 12},
		Arrivals:      dedup.New(),
		Heads:         dedup.New(),
		Store:         store,
		Metrics:       metrics.New(reg),
		Log:           slog.New(slog.DiscardHandler),
		Client:        "teku",
		ClientVersion: "teku/v26.10.1",
	}
	body := `{"slot":"10","block":"0xabc"}`

	if err := h.Event(context.Background(), "block_gossip", body, seen); err != nil {
		t.Fatal(err)
	}
	if err := h.Event(context.Background(), "block", body, seen.Add(20*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := h.Event(context.Background(), "head", body, seen.Add(5*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := h.Event(context.Background(), "head", body, seen.Add(8*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := h.Event(context.Background(), "block_gossip", `{not-json`, seen); err != nil {
		t.Fatal(err)
	}

	if len(store.arrivals) != 1 {
		t.Fatalf("arrivals %d, want 1", len(store.arrivals))
	}
	got := store.arrivals[0]
	if got.Topic != "block_gossip" || got.DelayMS != 1500 || got.Client != "teku" || got.Slot != 10 {
		t.Fatalf("arrival %+v", got)
	}
	if len(store.heads) != 1 || store.heads[0].DelayMS != 1505 {
		t.Fatalf("heads %+v", store.heads)
	}
	if n := testutil.ToFloat64(h.Metrics.Events.WithLabelValues("block_gossip", "stored")); n != 1 {
		t.Fatalf("stored gossip %v", n)
	}
	if n := testutil.ToFloat64(h.Metrics.Events.WithLabelValues("block", "duplicate")); n != 1 {
		t.Fatalf("duplicate block %v", n)
	}
	if n := testutil.ToFloat64(h.Metrics.Events.WithLabelValues("head", "duplicate")); n != 1 {
		t.Fatalf("duplicate head %v", n)
	}
	if n := testutil.ToFloat64(h.Metrics.Events.WithLabelValues("block_gossip", "invalid")); n != 1 {
		t.Fatalf("invalid %v", n)
	}
}

func TestStoreErrorForgetsKey(t *testing.T) {
	genesis := time.Unix(1_000, 0).UTC()
	store := &memStore{fail: true}
	h := &Handler{
		NodeID:   "n",
		Arm:      "observer",
		Clock:    slot.Clock{Genesis: genesis, SecondsPerSlot: 12},
		Arrivals: dedup.New(),
		Heads:    dedup.New(),
		Store:    store,
		Metrics:  metrics.New(prometheus.NewRegistry()),
		Log:      slog.New(slog.DiscardHandler),
		Client:   "teku",
	}
	err := h.Event(context.Background(), "block", `{"slot":"1","block":"0x1"}`, genesis.Add(13*time.Second))
	if err == nil {
		t.Fatal("expected store error")
	}
	store.fail = false
	if err := h.Event(context.Background(), "block", `{"slot":"1","block":"0x1"}`, genesis.Add(13*time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(store.arrivals) != 1 {
		t.Fatalf("retry did not store, got %d", len(store.arrivals))
	}
}
