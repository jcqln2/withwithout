// Package config reads the collector's environment.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config is the process configuration. Secrets stay in the environment.
type Config struct {
	BeaconAPIURL string
	NodeID       string
	Arm          string
	DatabaseURL  string
	MetricsAddr  string
	BlockGossip  bool
}

// FromEnv loads configuration. BEACON_API_URL and DATABASE_URL are required.
// BLOCK_GOSSIP defaults off. Set it to true only after the beacon client is seen
// emitting that topic.
func FromEnv() (Config, error) {
	cfg := Config{
		BeaconAPIURL: strings.TrimRight(os.Getenv("BEACON_API_URL"), "/"),
		NodeID:       envDefault("NODE_ID", "hoodi-public-1"),
		Arm:          envDefault("ARM", "observer"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		MetricsAddr:  envDefault("METRICS_ADDR", ":8080"),
		BlockGossip:  strings.EqualFold(os.Getenv("BLOCK_GOSSIP"), "true") || os.Getenv("BLOCK_GOSSIP") == "1",
	}
	if cfg.BeaconAPIURL == "" {
		return Config{}, fmt.Errorf("BEACON_API_URL is required")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.NodeID == "" || cfg.Arm == "" {
		return Config{}, fmt.Errorf("NODE_ID and ARM must be non-empty")
	}
	return cfg, nil
}

// Topics are the SSE subscriptions. block and head are always on.
func (c Config) Topics() []string {
	topics := []string{"block", "head"}
	if c.BlockGossip {
		topics = append(topics, "block_gossip")
	}
	return topics
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
