package dedup

import "testing"

func TestSeen(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		want []bool
	}{
		{name: "empty", keys: nil, want: nil},
		{name: "first is new", keys: []string{"a"}, want: []bool{false}},
		{name: "repeat", keys: []string{"root-1", "root-1"}, want: []bool{false, true}},
		{name: "distinct roots", keys: []string{"a", "b", "a"}, want: []bool{false, false, true}},
		{name: "node prefix keeps roots apart", keys: []string{"n1|a", "n2|a"}, want: []bool{false, false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := New()
			if len(tt.keys) != len(tt.want) {
				t.Fatalf("keys and want length differ")
			}
			for i, key := range tt.keys {
				if got := set.Seen(key); got != tt.want[i] {
					t.Fatalf("Seen(%q) call %d = %v, want %v", key, i, got, tt.want[i])
				}
			}
		})
	}
}

func TestForgetLetsKeyReturn(t *testing.T) {
	set := New()
	if set.Seen("block") {
		t.Fatal("first sighting was treated as a duplicate")
	}
	set.Forget("block")
	if set.Seen("block") {
		t.Fatal("forgotten key was still a duplicate")
	}
}
