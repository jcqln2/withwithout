package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jcqln2/withwithout/collector/internal/beacon"
	"github.com/jcqln2/withwithout/collector/internal/config"
	"github.com/jcqln2/withwithout/collector/internal/dedup"
	"github.com/jcqln2/withwithout/collector/internal/ingest"
	"github.com/jcqln2/withwithout/collector/internal/metrics"
	"github.com/jcqln2/withwithout/collector/internal/slot"
	"github.com/jcqln2/withwithout/collector/internal/sse"
	"github.com/jcqln2/withwithout/collector/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	cfg, err := config.FromEnv()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := openDB(ctx, log, cfg.DatabaseURL)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	api := &beacon.Client{BaseURL: cfg.BeaconAPIURL}
	info, err := loadInfo(ctx, log, api)
	if err != nil {
		log.Error("beacon info", "err", err)
		os.Exit(1)
	}
	log.Info("slot clock",
		"genesis_utc", info.Genesis.Format(time.RFC3339Nano),
		"seconds_per_slot", info.SecondsPerSlot,
		"client", info.Client.Name,
		"client_version", info.Client.Version,
		"clock_offset_ms", info.ClockOffsetMS,
		"node_id", cfg.NodeID,
		"arm", cfg.Arm,
		"topics", cfg.Topics(),
	)

	reg := prometheus.NewRegistry()
	m := metrics.New(reg)
	m.ClockMS.Set(float64(info.ClockOffsetMS))
	if err := serveMetrics(ctx, log, cfg.MetricsAddr, reg); err != nil {
		log.Error("metrics", "err", err)
		os.Exit(1)
	}

	arrivals := dedup.New()
	heads := dedup.New()
	if err := warm(ctx, db, cfg.NodeID, arrivals, heads); err != nil {
		log.Error("warm dedup", "err", err)
		os.Exit(1)
	}

	handler := &ingest.Handler{
		NodeID:        cfg.NodeID,
		Arm:           cfg.Arm,
		Clock:         slot.Clock{Genesis: info.Genesis, SecondsPerSlot: info.SecondsPerSlot},
		Arrivals:      arrivals,
		Heads:         heads,
		Store:         db,
		Metrics:       m,
		Log:           log,
		Client:        info.Client.Name,
		ClientVersion: info.Client.Version,
	}

	stream := &http.Client{}
	attempt := 0
	for {
		if ctx.Err() != nil {
			log.Info("shutdown")
			return
		}
		err := consume(ctx, log, api, stream, cfg.Topics(), handler, m)
		if ctx.Err() != nil {
			log.Info("shutdown")
			return
		}
		m.Reconnects.Inc()
		wait := backoff(attempt)
		attempt++
		log.Warn("sse disconnected", "err", err, "retry_in", wait.String(), "attempt", attempt)
		select {
		case <-ctx.Done():
			log.Info("shutdown")
			return
		case <-time.After(wait):
		}
	}
}

func openDB(ctx context.Context, log *slog.Logger, databaseURL string) (*store.Store, error) {
	var last error
	for attempt := 0; attempt < 30; attempt++ {
		db, err := store.Open(ctx, databaseURL)
		if err == nil {
			return db, nil
		}
		last = err
		wait := backoff(attempt)
		log.Warn("database not ready", "err", err, "retry_in", wait.String())
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil, last
}

func loadInfo(ctx context.Context, log *slog.Logger, api *beacon.Client) (beacon.Info, error) {
	var last error
	for attempt := 0; attempt < 10; attempt++ {
		info, err := api.Load(ctx)
		if err == nil {
			return info, nil
		}
		last = err
		wait := backoff(attempt)
		log.Warn("beacon api not ready", "err", err, "retry_in", wait.String())
		select {
		case <-ctx.Done():
			return beacon.Info{}, ctx.Err()
		case <-time.After(wait):
		}
	}
	return beacon.Info{}, last
}

func warm(ctx context.Context, db *store.Store, nodeID string, arrivals, heads *dedup.Set) error {
	roots, err := db.LoadArrivalRoots(ctx, nodeID)
	if err != nil {
		return err
	}
	for _, root := range roots {
		arrivals.Remember(nodeID + "|" + root)
	}
	headRoots, err := db.LoadHeadRoots(ctx, nodeID)
	if err != nil {
		return err
	}
	for _, root := range headRoots {
		heads.Remember(nodeID + "|" + root)
	}
	return nil
}

func consume(ctx context.Context, log *slog.Logger, api *beacon.Client, httpClient *http.Client, topics []string, handler *ingest.Handler, m *metrics.Metrics) error {
	resp, err := api.OpenEvents(ctx, topics, httpClient)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	id := identityFromHeaders(resp.Header, beacon.Identity{Name: handler.Client, Version: handler.ClientVersion})
	if id.Name != handler.Client || id.Version != handler.ClientVersion {
		log.Info("client identity from sse connection", "client", id.Name, "client_version", id.Version, "previous_client", handler.Client, "previous_version", handler.ClientVersion)
		handler.Client = id.Name
		handler.ClientVersion = id.Version
	}
	offset := clockOffset(resp.Header.Get("Date"), time.Now())
	m.ClockMS.Set(float64(offset))
	log.Info("sse connected", "topics", topics, "client", handler.Client, "client_version", handler.ClientVersion, "clock_offset_ms", offset, "date", resp.Header.Get("Date"))

	reader := bufio.NewReader(resp.Body)
	for {
		ev, err := sse.Read(reader)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return err
			}
			return err
		}
		if err := handler.Event(ctx, ev.Name, ev.Data, time.Now().UTC()); err != nil {
			return err
		}
	}
}

func identityFromHeaders(h http.Header, fallback beacon.Identity) beacon.Identity {
	name := h.Get("X-Dugtrio-Endpoint-Type")
	version := h.Get("X-Dugtrio-Endpoint-Version")
	if name == "" && version == "" {
		return fallback
	}
	if name == "" {
		name = fallback.Name
	}
	if version == "" {
		version = fallback.Version
	}
	return beacon.Identity{Name: name, Version: version}
}

func clockOffset(dateHeader string, now time.Time) int64 {
	if dateHeader == "" {
		return 0
	}
	server, err := http.ParseTime(dateHeader)
	if err != nil {
		return 0
	}
	return now.Sub(server).Milliseconds()
}

func serveMetrics(ctx context.Context, log *slog.Logger, addr string, reg *prometheus.Registry) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	srv := &http.Server{Handler: mux}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	go func() {
		log.Info("metrics listening", "addr", addr)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server", "err", err)
		}
	}()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	return nil
}

func backoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if attempt > 5 {
		attempt = 5
	}
	return time.Second * time.Duration(1<<attempt)
}
