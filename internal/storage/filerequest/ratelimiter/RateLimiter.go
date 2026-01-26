package ratelimiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var uuidLimiter = newLimiter()

// Currently unused
var byteLimiter = newLimiter()

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type Store struct {
	mu             sync.Mutex
	limiters       map[string]*limiterEntry
	cleanupStarted bool
}

func newLimiter() *Store {
	return &Store{
		limiters: make(map[string]*limiterEntry),
	}
}

func IsAllowedNewUuid(key string) bool {
	return uuidLimiter.Get(key, 1, 4).Allow()
}

func (s *Store) Get(key string, r rate.Limit, burst int) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.limiters[key]
	if !ok {
		e = &limiterEntry{
			limiter: rate.NewLimiter(r, burst),
		}
	}

	e.lastSeen = time.Now()
	s.limiters[key] = e
	s.StartCleanup(12 * time.Hour)
	return e.limiter
}

func (s *Store) StartCleanup(maxIdle time.Duration) {
	if s.cleanupStarted {
		return
	}
	s.cleanupStarted = true
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		for range ticker.C {
			now := time.Now()
			s.mu.Lock()
			for k, v := range s.limiters {
				if now.Sub(v.lastSeen) > maxIdle {
					delete(s.limiters, k)
				}
			}
			s.mu.Unlock()
		}
	}()
}
