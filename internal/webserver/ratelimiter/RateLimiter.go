package ratelimiter

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/logging"
	"golang.org/x/time/rate"
)

var newUuidLimiter = newLimiter()
var failedLoginLimiter = newLimiter()
var failedIdLimiter = newLimiter()

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type store struct {
	mu             sync.Mutex
	limiters       map[string]*limiterEntry
	cleanupStarted bool
}

func newLimiter() *store {
	return &store{
		limiters: make(map[string]*limiterEntry),
	}
}

// WaitOnFailedLogin blocks the current goroutine until the rate limiter allows a request
// Two failed attempts without limiting, thereafter one attempt every 3 seconds
func WaitOnFailedLogin(ip string) {
	_ = failedLoginLimiter.Get(ip, 1, 6).WaitN(context.Background(), 3)
}

// WaitOnFailedId blocks the current goroutine until the rate limiter allows a request
// Ten failed attempts without limiting, thereafter one attempt every second
func WaitOnFailedId(r *http.Request) {
	ip := logging.GetIpAddress(r)
	_ = failedIdLimiter.Get(ip, 1, 10).Wait(context.Background())
}

// IsAllowedNewUuid returns true if a new uuid is not rate-limited
// Four initial requests are allowed without rate limiting, thereafter one every second
func IsAllowedNewUuid(key string) bool {
	return newUuidLimiter.Get(key, 1, 4).Allow()
}

// Get returns the rate limiter for the given key
func (s *store) Get(key string, r rate.Limit, burst int) *rate.Limiter {
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

// StartCleanup starts a goroutine that continuously cleans up old entries from the store
func (s *store) StartCleanup(maxIdle time.Duration) {
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
