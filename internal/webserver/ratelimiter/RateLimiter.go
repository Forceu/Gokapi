package ratelimiter

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var newUuidLimiter = newLimiter()
var failedLoginLimiter = newLimiter()
var fileIdLimiter = newLimiter()
var failedDownloadPasswordLimiter = newLimiter()
var failedApiKeyLimiter = newLimiter()
var failedSetupLogin = newLimiter()

// isUnitTest must be false and is only set to true for running test units
// If true, rate limiting is disabled
var isUnitTest = false

const maxEntries = 100000

// maxIdleLimiter must exceed the longest burst/rate refill time of any limiter (currently 200s)
const maxIdleLimiter = 5 * time.Minute
const maxWaitTime = 30 * time.Second
const cleanupInterval = 4 * time.Minute

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// SetUnitTestMode disables all rate limiting
// This is only used for running unit tests
func SetUnitTestMode(enabled bool) {
	fmt.Println("Rate limiting disabled for unit tests")
	isUnitTest = enabled
}

type store struct {
	mu          sync.Mutex
	limiters    map[string]*limiterEntry
	overflow    *rate.Limiter // shared by all new keys while the store is full
	cleanupOnce sync.Once
}

func newLimiter() *store {
	return &store{
		limiters: make(map[string]*limiterEntry),
	}
}

// WaitOnLogin blocks the current goroutine until the rate limiter allows a request or returns false
// if the wait would take longer than maxWaitTime, or if ctx is cancelled (e.g. the client disconnected).
// Three attempts without limiting, thereafter one attempt every 3 seconds
func WaitOnLogin(ctx context.Context, ip string) bool {
	return failedLoginLimiter.wait(ctx, ip, 9, 3)
}

// WaitOnSetupLogin blocks the current goroutine until the rate limiter allows a request or returns false
// if the wait would take longer than maxWaitTime, or if ctx is cancelled (e.g. the client disconnected).
// 15 attempts without limiting, thereafter one attempt every 1 second
func WaitOnSetupLogin(ctx context.Context) bool {
	return failedSetupLogin.wait(ctx, "setup", 15, 1)
}

// WaitOnApiAuthentication blocks the current goroutine until the rate limiter allows a request or returns false
// if the wait would take longer than maxWaitTime, or if ctx is cancelled (e.g. the client disconnected).
// 200 attempts without limiting, thereafter one attempt every second
func WaitOnApiAuthentication(ctx context.Context, ip string) bool {
	return failedApiKeyLimiter.wait(ctx, ip, 200, 1)
}

// WaitOnDownloadPassword blocks the current goroutine until the rate limiter allows a request or returns false
// if the wait would take longer than maxWaitTime, or if ctx is cancelled (e.g. the client disconnected).
// Ten attempts without limiting, thereafter one attempt every 2 seconds
func WaitOnDownloadPassword(ctx context.Context, ip string) bool {
	return failedDownloadPasswordLimiter.wait(ctx, ip, 20, 2)
}

// WaitOnFileId blocks the current goroutine until the rate limiter allows a request or returns false
// if the wait would take longer than maxWaitTime, or if ctx is cancelled (e.g. the client disconnected).
// 50 attempts without limiting, thereafter one attempt every second
func WaitOnFileId(ctx context.Context, ip string) bool {
	return fileIdLimiter.wait(ctx, ip, 50, 1)
}

// IsAllowedNewUuid returns true if a new uuid is not rate-limited
// Seven initial requests are allowed without rate limiting, thereafter one every second
func IsAllowedNewUuid(key string) bool {
	return newUuidLimiter.Get(key, 1, 7).Allow()
}

// Get returns the rate limiter for the given key
func (s *store) Get(key string, r rate.Limit, burst int) *rate.Limiter {
	if isUnitTest {
		return rate.NewLimiter(r, burst)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.limiters[key]
	if !ok {
		if len(s.limiters) >= maxEntries {
			if s.overflow == nil {
				s.overflow = rate.NewLimiter(r, burst)
			}
			return s.overflow
		}
		e = &limiterEntry{
			limiter: rate.NewLimiter(r, burst),
		}
	}

	e.lastSeen = time.Now()
	s.limiters[key] = e
	s.StartCleanup()
	return e.limiter
}

// StartCleanup starts a goroutine that continuously cleans up old entries from the store
func (s *store) StartCleanup() {
	s.cleanupOnce.Do(s.runCleanup)
}

// runCleanup runs a goroutine that continuously cleans up old entries from the store
func (s *store) runCleanup() {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		for range ticker.C {
			now := time.Now()
			s.mu.Lock()
			for k, v := range s.limiters {
				if now.Sub(v.lastSeen) > maxIdleLimiter {
					delete(s.limiters, k)
				}
			}
			s.mu.Unlock()
		}
	}()
}

// wait blocks until the limiter allows n events for key.
// Returns false if the wait would take longer than maxWaitTime,
// or if ctx is cancelled (e.g. the client disconnected).
func (s *store) wait(ctx context.Context, key string, burst, n int) bool {
	ctx, cancel := context.WithTimeout(ctx, maxWaitTime)
	defer cancel()
	return s.Get(limiterKey(key), 1, burst).WaitN(ctx, n) == nil
}

// limiterKey normalizes rate limiter keys by canonicalizing IP addresses.
// It preserves non-IP keys and groups IPv6 addresses by their /64 prefix.
func limiterKey(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip // non-IP keys such as "setup" pass through
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String()
}
