package csrftoken

import (
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// resetStateBlockCleanup resets the package-level state between tests.
func resetStateBlockCleanup() {
	mutex.Lock()
	tokens = make(map[string]int64)
	mutex.Unlock()
	cleanupOnce = sync.Once{}
	cleanupOnce.Do(func() {
		// Do nothing, this prevents cleanup deadlock
	})
}

// TestGenerate_ReturnsNonEmptyToken checks that Generate returns a non-empty token string.
func TestGenerate_ReturnsNonEmptyToken(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate()
	if token == "" {
		t.Error("expected non-empty token, got empty string")
	}
}

// TestGenerate_TokenLength checks that the generated token has the expected length (20 chars).
func TestGenerate_TokenLength(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate()
	if len(token) != 20 {
		t.Errorf("expected token length 20, got %d", len(token))
	}
}

// TestGenerate_UniqueTokens checks that two calls produce different tokens.
func TestGenerate_UniqueTokens(t *testing.T) {
	resetStateBlockCleanup()
	token1 := Generate()
	token2 := Generate()
	if token1 == token2 {
		t.Error("expected unique tokens, got identical tokens")
	}
}

// TestGenerate_StoresToken checks that the token is stored in the map.
func TestGenerate_StoresToken(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate()
	mutex.Lock()
	_, ok := tokens[token]
	mutex.Unlock()
	if !ok {
		t.Error("expected token to be stored in map, but it was not found")
	}
}

// TestGenerate_ExpiryIsInFuture checks that the stored token has an expiry in the future.
func TestGenerate_ExpiryIsInFuture(t *testing.T) {
	resetStateBlockCleanup()
	before := time.Now().Unix()
	token := Generate()
	mutex.Lock()
	expiry := tokens[token]
	mutex.Unlock()
	if expiry <= before {
		t.Errorf("expected expiry > %d, got %d", before, expiry)
	}
}

// TestTTL_ExactExpiry checks that the token expiry is exactly now+ttl.
func TestTTL_ExactExpiry(t *testing.T) {
	resetStateBlockCleanup()
	now := time.Now()
	token := Generate()
	mutex.Lock()
	expiry := tokens[token]
	mutex.Unlock()
	expected := now.Add(ttl).Unix()
	if expiry != expected {
		t.Errorf("expected expiry %d, got %d", expected, expiry)
	}
}

// TestIsValid_ValidToken checks that a freshly generated token is valid.
func TestIsValid_ValidToken(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate()
	if !IsValid(token) {
		t.Error("expected IsValid to return true for a fresh token")
	}
}

// TestIsValid_UnknownToken checks that an unknown token is invalid.
func TestIsValid_UnknownToken(t *testing.T) {
	resetStateBlockCleanup()
	if IsValid("nonexistent-token-xyz") {
		t.Error("expected IsValid to return false for unknown token")
	}
}

// TestIsValid_SingleUse checks that a token cannot be used twice.
func TestIsValid_SingleUse(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate()
	if !IsValid(token) {
		t.Fatal("expected first IsValid to return true")
	}
	if IsValid(token) {
		t.Error("expected second IsValid to return false — token must be single-use")
	}
}

// TestIsValid_DeletesTokenAfterUse checks that the token is removed from the map after validation.
func TestIsValid_DeletesTokenAfterUse(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate()
	IsValid(token)
	mutex.Lock()
	_, ok := tokens[token]
	mutex.Unlock()
	if ok {
		t.Error("expected token to be deleted from map after use")
	}
}

// TestIsValid_ExpiredToken checks that a manually inserted expired token is rejected and removed.
func TestIsValid_ExpiredToken(t *testing.T) {
	resetStateBlockCleanup()
	token := "test-expired-token"
	mutex.Lock()
	tokens[token] = time.Now().Add(-1 * time.Second).Unix()
	mutex.Unlock()

	if IsValid(token) {
		t.Error("expected IsValid to return false for expired token")
	}

	// Verify the expired token was deleted from the map.
	mutex.Lock()
	_, ok := tokens[token]
	mutex.Unlock()
	if ok {
		t.Error("expected expired token to be deleted from map, but it still exists")
	}
}

// TestIsValid_EmptyToken checks that an empty token string is handled gracefully.
func TestIsValid_EmptyToken(t *testing.T) {
	resetStateBlockCleanup()
	if IsValid("") {
		t.Error("expected IsValid to return false for empty token")
	}
}

// TestIsValid_TokenExpiresAfterTTL advances the fake clock past the TTL and confirms
// the token is rejected — no real sleeping required.
func TestIsValid_TokenExpiresAfterTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resetStateBlockCleanup()
		token := Generate()
		cleanup(false)

		// Advance fake clock beyond the 5-minute TTL.
		time.Sleep(ttl + time.Second)
		cleanup(false)
		synctest.Wait()

		if IsValid(token) {
			t.Error("expected IsValid to return false after TTL has elapsed")
		}
	})
}

// TestIsValid_TokenStillValidBeforeTTL confirms a token remains valid just before expiry.
func TestIsValid_TokenStillValidBeforeTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resetStateBlockCleanup()
		token := Generate()
		cleanup(false)

		// Advance to just before expiry.
		time.Sleep(ttl - time.Second)
		cleanup(false)
		synctest.Wait()

		if !IsValid(token) {
			t.Error("expected IsValid to return true just before TTL elapses")
		}
	})
}

// TestGenerate_ConcurrentSafe checks that concurrent calls to Generate do not race.
func TestGenerate_ConcurrentSafe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resetStateBlockCleanup()
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				Generate()
			}()
		}
		wg.Wait()
		cleanup(false)
		synctest.Wait()

		mutex.Lock()
		count := len(tokens)
		mutex.Unlock()

		if count != 50 {
			t.Errorf("expected 50 tokens after concurrent generation, got %d", count)
		}
	})
}

// TestIsValid_ConcurrentSafe checks that concurrent reads on IsValid do not race.
func TestIsValid_ConcurrentSafe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resetStateBlockCleanup()
		token := Generate()

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				IsValid(token)
			}()
		}
		cleanup(false)
		wg.Wait()
		synctest.Wait()
	})
}

// TestCleanup_RemovesExpiredTokens checks that cleanup removes expired entries
// while leaving valid ones intact.
func TestCleanup_RemovesExpiredTokens(t *testing.T) {
	resetStateBlockCleanup()
	mutex.Lock()
	tokens["expired-1"] = time.Now().Add(-10 * time.Second).Unix()
	tokens["expired-2"] = time.Now().Add(-5 * time.Second).Unix()
	tokens["valid-1"] = time.Now().Add(5 * time.Minute).Unix()
	mutex.Unlock()

	cleanup(false)

	mutex.Lock()
	_, e1 := tokens["expired-1"]
	_, e2 := tokens["expired-2"]
	_, v1 := tokens["valid-1"]
	mutex.Unlock()

	if e1 {
		t.Error("expected expired-1 to be removed by cleanup")
	}
	if e2 {
		t.Error("expected expired-2 to be removed by cleanup")
	}
	if !v1 {
		t.Error("expected valid-1 to still be present after cleanup")
	}
}

// TestCleanup_EmptyMap checks that cleanup on an empty map does not panic.
func TestCleanup_EmptyMap(t *testing.T) {
	resetStateBlockCleanup()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("cleanup panicked on empty map: %v", r)
		}
	}()
	cleanup(false)
}

// TestCleanup_PeriodicRunsAfterOneHour verifies that the periodic cleanup goroutine
// re-runs after one hour. The fake clock advances instantly — no real waiting.
func TestCleanup_PeriodicRunsAfterOneHour(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resetStateBlockCleanup()

		// Insert a token that expires in 30 minutes.
		mutex.Lock()
		tokens["will-expire"] = time.Now().Add(30 * time.Minute).Unix()
		mutex.Unlock()

		// Start the periodic cleanup goroutine.
		cleanup(false)
		synctest.Wait()

		// Token should still be present before one hour elapses.
		mutex.Lock()
		_, stillThere := tokens["will-expire"]
		mutex.Unlock()
		if !stillThere {
			t.Error("expected token to still be present before one hour")
		}

		// Advance fake clock by 1 hour + 1 second to trigger the next cleanup cycle.
		time.Sleep(time.Hour + time.Second)
		cleanup(false)
		synctest.Wait()

		// The token (expired at +30 min) should now be gone.
		mutex.Lock()
		_, stillThere = tokens["will-expire"]
		mutex.Unlock()
		if stillThere {
			t.Error("expected token to be removed after periodic cleanup ran")
		}
	})
}
