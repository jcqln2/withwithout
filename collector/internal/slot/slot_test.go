package slot

import (
	"testing"
	"time"
)

func TestDelayMS(t *testing.T) {
	genesis := time.Unix(1_742_213_400, 0).UTC()
	clock := Clock{Genesis: genesis, SecondsPerSlot: 12}

	tests := []struct {
		name    string
		slot    uint64
		seen    time.Time
		want    int64
		wantUTC time.Time
	}{
		{
			name:    "exact slot start",
			slot:    0,
			seen:    genesis,
			want:    0,
			wantUTC: genesis,
		},
		{
			name:    "two seconds into slot 1",
			slot:    1,
			seen:    genesis.Add(14 * time.Second),
			want:    2000,
			wantUTC: genesis.Add(12 * time.Second),
		},
		{
			name:    "hoodi slot 4206974 at genesis plus slot*12 plus 3.5s",
			slot:    4_206_974,
			seen:    genesis.Add(time.Duration(4_206_974)*12*time.Second + 3500*time.Millisecond),
			want:    3500,
			wantUTC: genesis.Add(time.Duration(4_206_974) * 12 * time.Second),
		},
		{
			name:    "seen before slot start",
			slot:    10,
			seen:    genesis.Add(10*12*time.Second - 40*time.Millisecond),
			want:    -40,
			wantUTC: genesis.Add(120 * time.Second),
		},
		{
			name:    "local zone input is converted to utc",
			slot:    2,
			seen:    genesis.Add(25 * time.Second).In(time.FixedZone("EDT", -4*3600)),
			want:    1000,
			wantUTC: genesis.Add(24 * time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clock.Start(tt.slot); !got.Equal(tt.wantUTC) {
				t.Fatalf("Start(%d) = %s, want %s", tt.slot, got, tt.wantUTC)
			}
			if got := clock.DelayMS(tt.slot, tt.seen); got != tt.want {
				t.Fatalf("DelayMS(%d) = %d, want %d", tt.slot, got, tt.want)
			}
		})
	}
}
