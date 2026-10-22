// Package metrics exposes the collector's Prometheus series.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics counts events, reconnects, and the arrival-delay histogram.
type Metrics struct {
	Events     *prometheus.CounterVec
	Reconnects prometheus.Counter
	DelayMS    prometheus.Histogram
	ClockMS    prometheus.Gauge
}

func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		Events: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "withwithout_events_total",
			Help: "Beacon SSE events by topic and outcome (stored, duplicate, invalid, store_error).",
		}, []string{"topic", "outcome"}),
		Reconnects: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "withwithout_reconnects_total",
			Help: "SSE reconnects after a disconnect or failed dial.",
		}),
		DelayMS: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "withwithout_block_delay_milliseconds",
			Help:    "Block arrival delay from slot start, in milliseconds, for stored block and block_gossip rows.",
			Buckets: []float64{50, 100, 250, 500, 1000, 1500, 2000, 3000, 4000, 6000, 8000, 12000, 20000},
		}),
		ClockMS: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "withwithout_clock_offset_milliseconds",
			Help: "Collector clock minus the beacon HTTP Date header, in milliseconds. Not subtracted from delays.",
		}),
	}
	reg.MustRegister(m.Events, m.Reconnects, m.DelayMS, m.ClockMS)
	return m
}
