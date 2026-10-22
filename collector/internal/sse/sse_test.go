package sse

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	stream := strings.Join([]string{
		": ready",
		"",
		"event: block_gossip",
		`data: {"slot":"10","block":"0xabc"}`,
		"",
		"event: head",
		"data: {\"slot\":\"10\",",
		"data: \"block\":\"0xabc\"}",
		"",
		"data: no-name",
		"",
	}, "\n")

	r := bufio.NewReader(strings.NewReader(stream))
	var got []Event
	for {
		ev, err := Read(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, ev)
	}

	want := []Event{
		{Name: "block_gossip", Data: `{"slot":"10","block":"0xabc"}`},
		{Name: "head", Data: "{\"slot\":\"10\",\n\"block\":\"0xabc\"}"},
		{Name: "", Data: "no-name"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}
