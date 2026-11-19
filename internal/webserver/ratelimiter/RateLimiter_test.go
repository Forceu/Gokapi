package ratelimiter

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/time/rate"
)

// newTestStore returns a store whose background cleanup goroutine can never start.
// That goroutine loops forever, which would keep a synctest bubble from exiting.
func newTestStore() *store {
	s := newLimiter()
	s.cleanupOnce.Do(func() {})
	return s
}

// useFreshStores replaces the package-level stores for the duration of the test,
// so that tests do not share limiter state and no cleanup goroutine is started.
func useFreshStores(t *testing.T) {
	t.Helper()
	oldUuid, oldLogin, oldFileId := newUuidLimiter, failedLoginLimiter, fileIdLimiter
	oldDownload, oldApi, oldSetup := failedDownloadPasswordLimiter, failedApiKeyLimiter, failedSetupLogin

	newUuidLimiter = newTestStore()
	failedLoginLimiter = newTestStore()
	fileIdLimiter = newTestStore()
	failedDownloadPasswordLimiter = newTestStore()
	failedApiKeyLimiter = newTestStore()
	failedSetupLogin = newTestStore()

	t.Cleanup(func() {
		newUuidLimiter, failedLoginLimiter, fileIdLimiter = oldUuid, oldLogin, oldFileId
		failedDownloadPasswordLimiter, failedApiKeyLimiter, failedSetupLogin = oldDownload, oldApi, oldSetup
	})
}

// timed runs f and returns its result and how long it blocked (fake time inside a synctest bubble)
func timed(f func() bool) (bool, time.Duration) {
	start := time.Now()
	ok := f()
	return ok, time.Since(start)
}

// ---- Tests that do not depend on time ----

func TestLimiterKey(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"ipv4", "1.2.3.4", "1.2.3.4"},
		{"ipv4 mapped in ipv6", "::ffff:1.2.3.4", "1.2.3.4"},
		{"ipv6 reduced to /64", "2001:db8:1:2:aaaa:bbbb:cccc:dddd", "2001:db8:1:2::"},
		{"ipv6 different spelling", "2001:0DB8:0001:0002:0000:0000:0000:0001", "2001:db8:1:2::"},
		{"ipv6 loopback", "::1", "::"},
		{"non-ip key passes through", "setup", "setup"},
		{"empty key passes through", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := limiterKey(tt.in); got != tt.want {
				t.Errorf("limiterKey(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLimiterKeyIPv6PrefixGrouping(t *testing.T) {
	sameA := limiterKey("2001:db8:1:2::1")
	sameB := limiterKey("2001:db8:1:2:ffff:ffff:ffff:ffff")
	other := limiterKey("2001:db8:1:3::1")

	if sameA != sameB {
		t.Errorf("addresses in the same /64 should share a key: %q != %q", sameA, sameB)
	}
	if sameA == other {
		t.Errorf("addresses in different /64 prefixes must not share a key: %q", sameA)
	}
}

func TestGetReturnsSameLimiterForSameKey(t *testing.T) {
	s := newTestStore()
	a := s.Get("key", 1, 5)
	b := s.Get("key", 1, 5)
	if a != b {
		t.Error("the same key must return the same limiter")
	}
	if s.Get("other", 1, 5) == a {
		t.Error("different keys must not share a limiter")
	}
}

func TestStoreOverflow(t *testing.T) {
	s := newTestStore()
	for i := 0; i < maxEntries; i++ {
		s.limiters[fmt.Sprintf("key-%d", i)] = &limiterEntry{
			limiter:  rate.NewLimiter(1, 1),
			lastSeen: time.Now(),
		}
	}

	existing := s.Get("key-0", 1, 1)
	newA := s.Get("new-a", 1, 1)
	newB := s.Get("new-b", 1, 1)

	if newA != newB {
		t.Error("new keys in a full store must share the overflow limiter")
	}
	if newA == existing {
		t.Error("the overflow limiter must not be an existing entry's limiter")
	}
	if len(s.limiters) != maxEntries {
		t.Errorf("store must not grow beyond maxEntries: got %d", len(s.limiters))
	}
	if s.Get("key-0", 1, 1) != existing {
		t.Error("existing keys must keep their own limiter when the store is full")
	}
}

func TestWaitRejectsWhenCostExceedsBurst(t *testing.T) {
	s := newTestStore()
	// n > burst can never be satisfied, so it must fail closed
	if s.wait(context.Background(), "1.2.3.4", 2, 3) {
		t.Fatal("request with cost above the burst size should be rejected")
	}
}

func TestWaitRejectsCancelledContext(t *testing.T) {
	s := newTestStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if s.wait(ctx, "1.2.3.4", 50, 1) {
		t.Fatal("request with a cancelled context should be rejected")
	}
}

// A second request using the complete burst would need ~50s and is rejected immediately,
// so these tests do not need to wait for anything.
func TestWaitSharesLimiterWithinIPv6Prefix(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	if !s.wait(ctx, "2001:db8:1:2::1", 50, 50) {
		t.Fatal("first request should be allowed")
	}
	if s.wait(ctx, "2001:db8:1:2::2", 50, 50) {
		t.Error("address in the same /64 must share the limiter")
	}
	if !s.wait(ctx, "2001:db8:1:3::1", 50, 50) {
		t.Error("address in a different /64 should have its own limiter")
	}
}

func TestWaitTreatsIPv4MappedAddressAsIPv4(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	if !s.wait(ctx, "1.2.3.4", 50, 50) {
		t.Fatal("first request should be allowed")
	}
	if s.wait(ctx, "::ffff:1.2.3.4", 50, 50) {
		t.Error("IPv4-mapped IPv6 address must share the limiter with the IPv4 address")
	}
}

func TestWaitSeparateKeysAreIndependent(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	if !s.wait(ctx, "1.2.3.4", 50, 50) {
		t.Fatal("first request should be allowed")
	}
	if !s.wait(ctx, "5.6.7.8", 50, 50) {
		t.Error("a different IP must not be affected by the first one")
	}
}

func TestUnitTestModeDisablesLimiting(t *testing.T) {
	SetUnitTestMode(true)
	defer SetUnitTestMode(false)

	s := newTestStore()
	if s.Get("a", 1, 1) == s.Get("a", 1, 1) {
		t.Error("in unit test mode every Get should return a fresh limiter")
	}
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if !s.wait(ctx, "1.2.3.4", 50, 50) {
			t.Fatalf("request %d should be allowed in unit test mode", i+1)
		}
	}
}

// ---- Tests that depend on time, run with a fake clock ----

func TestWaitAllowsBurstImmediately(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newTestStore()
		for i := 0; i < 5; i++ {
			ok, elapsed := timed(func() bool { return s.wait(context.Background(), "1.2.3.4", 5, 1) })
			if !ok {
				t.Fatalf("attempt %d within the burst was rejected", i+1)
			}
			if elapsed != 0 {
				t.Errorf("attempt %d within the burst blocked for %v", i+1, elapsed)
			}
		}
	})
}

func TestWaitBlocksUntilTokenIsAvailable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newTestStore()
		ctx := context.Background()
		for i := 0; i < 5; i++ {
			s.wait(ctx, "1.2.3.4", 5, 1)
		}

		// The burst is used up, tokens refill at one per second
		for i := 1; i <= 2; i++ {
			ok, elapsed := timed(func() bool { return s.wait(ctx, "1.2.3.4", 5, 1) })
			if !ok {
				t.Fatalf("attempt after the burst was rejected")
			}
			if elapsed != time.Second {
				t.Errorf("attempt %d after the burst waited %v, want 1s", i, elapsed)
			}
		}
	})
}

func TestWaitRejectsWhenWaitExceedsMaximum(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newTestStore()
		ctx := context.Background()

		// Uses the complete burst of 50 tokens
		if !s.wait(ctx, "1.2.3.4", 50, 50) {
			t.Fatal("first request should be allowed")
		}

		// 31 tokens take 31s, longer than maxWaitTime (30s): rejected without blocking
		ok, elapsed := timed(func() bool { return s.wait(ctx, "1.2.3.4", 50, 31) })
		if ok {
			t.Fatal("request needing more than maxWaitTime should be rejected")
		}
		if elapsed != 0 {
			t.Errorf("rejection should be immediate, took %v", elapsed)
		}

		// The rejected request must not have reserved any tokens:
		// 29 tokens still take exactly 29s (not 31s + 29s)
		ok, elapsed = timed(func() bool { return s.wait(ctx, "1.2.3.4", 50, 29) })
		if !ok {
			t.Fatal("request needing less than maxWaitTime should be allowed")
		}
		if elapsed != 29*time.Second {
			t.Errorf("waited %v, want 29s", elapsed)
		}
	})
}

func TestWaitRefillsOverTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newTestStore()
		ctx := context.Background()

		if !s.wait(ctx, "1.2.3.4", 50, 50) {
			t.Fatal("first request should be allowed")
		}

		time.Sleep(50 * time.Second)

		ok, elapsed := timed(func() bool { return s.wait(ctx, "1.2.3.4", 50, 50) })
		if !ok || elapsed != 0 {
			t.Errorf("after a full refill the request should pass immediately, got ok=%v elapsed=%v", ok, elapsed)
		}
	})
}

func TestWaitAbortsWhenContextIsCancelledWhileWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newTestStore()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s.wait(ctx, "1.2.3.4", 50, 50) // uses the complete burst

		start := time.Now()
		result := make(chan bool)
		go func() {
			// Needs 20s, which is within maxWaitTime, so the call starts waiting
			result <- s.wait(ctx, "1.2.3.4", 50, 20)
		}()

		time.Sleep(time.Second)
		cancel()

		if <-result {
			t.Fatal("a cancelled wait must return false")
		}
		if elapsed := time.Since(start); elapsed != time.Second {
			t.Errorf("wait should end when the context is cancelled, took %v", elapsed)
		}
	})
}

func TestIsAllowedNewUuid(t *testing.T) {
	useFreshStores(t)
	synctest.Test(t, func(t *testing.T) {
		for i := 0; i < 7; i++ {
			if !IsAllowedNewUuid("a") {
				t.Fatalf("request %d within the burst of 7 should be allowed", i+1)
			}
		}
		if IsAllowedNewUuid("a") {
			t.Error("the 8th immediate request should be rate limited")
		}
		if !IsAllowedNewUuid("b") {
			t.Error("a different key must not be affected")
		}

		// One token is refilled per second
		time.Sleep(time.Second)
		if !IsAllowedNewUuid("a") {
			t.Error("a request should be allowed one second later")
		}
		if IsAllowedNewUuid("a") {
			t.Error("only one token should have been refilled")
		}
	})
}

func TestPublicWaitFunctionsMatchDocumentedLimits(t *testing.T) {
	tests := []struct {
		name  string
		free  int           // attempts without limiting
		delay time.Duration // wait for the next attempt after that
		call  func(ctx context.Context) bool
	}{
		{"login", 3, 3 * time.Second, func(ctx context.Context) bool { return WaitOnLogin(ctx, "1.2.3.4") }},
		{"setupLogin", 15, time.Second, func(ctx context.Context) bool { return WaitOnSetupLogin(ctx) }},
		{"apiAuthentication", 200, time.Second, func(ctx context.Context) bool { return WaitOnApiAuthentication(ctx, "1.2.3.4") }},
		{"downloadPassword", 10, 2 * time.Second, func(ctx context.Context) bool { return WaitOnDownloadPassword(ctx, "1.2.3.4") }},
		{"fileId", 50, time.Second, func(ctx context.Context) bool { return WaitOnFileId(ctx, "1.2.3.4") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useFreshStores(t)
			synctest.Test(t, func(t *testing.T) {
				ctx := context.Background()
				call := func() bool { return tt.call(ctx) }

				for i := 1; i <= tt.free; i++ {
					ok, elapsed := timed(call)
					if !ok || elapsed != 0 {
						t.Fatalf("attempt %d of %d free attempts: ok=%v, blocked %v", i, tt.free, ok, elapsed)
					}
				}

				ok, elapsed := timed(call)
				if !ok {
					t.Fatal("the first limited attempt should be allowed after waiting")
				}
				if elapsed != tt.delay {
					t.Errorf("first limited attempt waited %v, want %v", elapsed, tt.delay)
				}
			})
		})
	}
}
