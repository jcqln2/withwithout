package config

import "testing"

func TestTopics(t *testing.T) {
	tests := []struct {
		name     string
		gossip   bool
		wantLen  int
		wantLast string
	}{
		{name: "default", gossip: false, wantLen: 2, wantLast: "head"},
		{name: "gossip on", gossip: true, wantLen: 3, wantLast: "block_gossip"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (Config{BlockGossip: tt.gossip}).Topics()
			if len(got) != tt.wantLen || got[len(got)-1] != tt.wantLast {
				t.Fatalf("topics %#v", got)
			}
		})
	}
}

func TestFromEnvRequiresURL(t *testing.T) {
	t.Setenv("BEACON_API_URL", "")
	t.Setenv("DATABASE_URL", "")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected missing url error")
	}
}
