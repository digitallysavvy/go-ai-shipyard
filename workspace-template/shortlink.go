// Package shortlink stores short codes that redirect to long URLs.
package shortlink

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("shortlink: code not found")
	ErrExpired  = errors.New("shortlink: link expired")
)

type link struct {
	URL       string
	ExpiresAt time.Time // zero means the link never expires
}

// Store is an in-memory, concurrency-safe link store.
type Store struct {
	mu    sync.RWMutex
	links map[string]link
	now   func() time.Time
}

// NewStore returns an empty store that uses the given clock.
func NewStore(now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{links: make(map[string]link), now: now}
}

// Add registers code -> url. A ttl of zero means the link never expires.
func (s *Store) Add(code, url string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l := link{URL: url}
	if ttl > 0 {
		l.ExpiresAt = s.now().Add(ttl)
	}
	s.links[code] = l
}

// Resolve returns the URL for code, or ErrNotFound / ErrExpired.
func (s *Store) Resolve(code string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.links[code]
	if !ok {
		return "", ErrNotFound
	}
	if !l.ExpiresAt.IsZero() && s.now().Before(l.ExpiresAt) {
		return "", ErrExpired
	}
	return l.URL, nil
}
