// Package slot converts a beacon slot number into a UTC slot start and a delay.
package slot

import "time"

// Clock is the beacon chain's slot schedule. Genesis must be UTC.
type Clock struct {
	Genesis        time.Time
	SecondsPerSlot uint64
}

// Start is genesis + slot * seconds_per_slot, in UTC.
func (c Clock) Start(slot uint64) time.Time {
	return c.Genesis.Add(time.Duration(slot) * time.Duration(c.SecondsPerSlot) * time.Second).UTC()
}

// DelayMS is seen minus slot start, in milliseconds.
// A negative result means seen is before the computed slot start. Callers store it.
func (c Clock) DelayMS(slot uint64, seen time.Time) int64 {
	return seen.UTC().Sub(c.Start(slot)).Milliseconds()
}
