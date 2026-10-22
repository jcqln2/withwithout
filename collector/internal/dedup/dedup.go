// Package dedup remembers keys that have already been stored.
package dedup

import "sync"

// Set is an in-memory first-seen set. The first caller of Seen wins.
type Set struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func New() *Set {
	return &Set{seen: make(map[string]struct{})}
}

// Seen reports whether key was already present. The first call for a key returns false
// and records it. Later calls return true.
func (s *Set) Seen(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[key]; ok {
		return true
	}
	s.seen[key] = struct{}{}
	return false
}

// Forget removes key so a failed store can be retried.
func (s *Set) Forget(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.seen, key)
}

// Remember marks key seen without reporting a duplicate. Used when warming from the database.
func (s *Set) Remember(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[key] = struct{}{}
}
