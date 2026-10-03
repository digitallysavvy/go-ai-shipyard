package shortlink

import (
	"errors"
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

func TestResolvePermanentLink(t *testing.T) {
	s := NewStore(nil)
	s.Add("go", "https://go.dev", 0)
	got, err := s.Resolve("go")
	if err != nil || got != "https://go.dev" {
		t.Fatalf("Resolve(go) = %q, %v; want https://go.dev, nil", got, err)
	}
}

func TestResolveLiveLink(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)}
	s := NewStore(clock.Now)
	s.Add("launch", "https://goaisdk.com", time.Hour)
	clock.Advance(30 * time.Minute)
	got, err := s.Resolve("launch")
	if err != nil || got != "https://goaisdk.com" {
		t.Fatalf("Resolve(launch) after 30m = %q, %v; want the URL, nil", got, err)
	}
}

func TestResolveExpiredLink(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)}
	s := NewStore(clock.Now)
	s.Add("launch", "https://goaisdk.com", time.Hour)
	clock.Advance(2 * time.Hour)
	if _, err := s.Resolve("launch"); !errors.Is(err, ErrExpired) {
		t.Fatalf("Resolve(launch) after 2h err = %v; want ErrExpired", err)
	}
}

func TestResolveUnknownCode(t *testing.T) {
	s := NewStore(nil)
	if _, err := s.Resolve("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Resolve(nope) err = %v; want ErrNotFound", err)
	}
}
