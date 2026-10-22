package beacon

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", time.Unix(1_700_000_000, 0).UTC().Format(http.TimeFormat))
		switch r.URL.Path {
		case "/eth/v1/beacon/genesis":
			_, _ = w.Write([]byte(`{"data":{"genesis_time":"1742213400","genesis_validators_root":"0xabc","genesis_fork_version":"0x10000910"}}`))
		case "/eth/v1/config/spec":
			_, _ = w.Write([]byte(`{"data":{"SECONDS_PER_SLOT":"12","CONFIG_NAME":"hoodi"}}`))
		case "/eth/v1/node/version":
			_, _ = w.Write([]byte(`{"data":{"version":"teku/v26.10.1/linux-x86_64"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	info, err := (&Client{BaseURL: srv.URL}).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !info.Genesis.Equal(time.Unix(1742213400, 0).UTC()) {
		t.Fatalf("genesis %s", info.Genesis)
	}
	if info.SecondsPerSlot != 12 {
		t.Fatalf("seconds %d", info.SecondsPerSlot)
	}
	if info.Client.Name != "teku" || info.Client.Version != "teku/v26.10.1/linux-x86_64" {
		t.Fatalf("client %+v", info.Client)
	}
}

func TestParseIdentity(t *testing.T) {
	tests := []struct {
		in      string
		name    string
		version string
	}{
		{in: "teku/v26.10.1/linux-x86_64", name: "teku", version: "teku/v26.10.1/linux-x86_64"},
		{in: "Lighthouse/v8.2.3", name: "Lighthouse", version: "Lighthouse/v8.2.3"},
		{in: "", name: "unknown", version: ""},
	}
	for _, tt := range tests {
		got := ParseIdentity(tt.in)
		if got.Name != tt.name || got.Version != tt.version {
			t.Fatalf("ParseIdentity(%q) = %+v", tt.in, got)
		}
	}
}

func TestEventsURL(t *testing.T) {
	c := &Client{BaseURL: "https://beacon.example/"}
	got, err := c.EventsURL([]string{"block", "head", "block_gossip"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://beacon.example/eth/v1/events?topics=block%2Chead%2Cblock_gossip" {
		t.Fatalf("url %s", got)
	}
}
