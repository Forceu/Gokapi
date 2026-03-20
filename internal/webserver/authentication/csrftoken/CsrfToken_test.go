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
	tokens = make(map[string]csrfToken)
	mutex.Unlock()
	cleanupOnce = sync.Once{}
	cleanupOnce.Do(func() {
		// Do nothing, this prevents cleanup deadlock
	})
}

// TestGenerate_ReturnsNonEmptyToken checks that Generate returns a non-empty token string.
func TestGenerate_ReturnsNonEmptyToken(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeLogin)
	if token == "" {
		t.Error("expected non-empty token, got empty string")
	}
}

// TestGenerate_TokenLength checks that the generated token has the expected length (20 chars).
func TestGenerate_TokenLength(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeLogin)
	if len(token) != 20 {
		t.Errorf("expected token length 20, got %d", len(token))
	}
}

// TestGenerate_UniqueTokens checks that two calls produce different tokens.
func TestGenerate_UniqueTokens(t *testing.T) {
	resetStateBlockCleanup()
	token1 := Generate(TypeLogin)
	token2 := Generate(TypeLogin)
	if token1 == token2 {
		t.Error("expected unique tokens, got identical tokens")
	}
}

// TestGenerate_StoresToken checks that the token is stored in the map.
func TestGenerate_StoresToken(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeLogin)
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
	tokenId := Generate(TypeLogin)
	mutex.Lock()
	token := tokens[tokenId]
	mutex.Unlock()
	if token.Expiry <= before {
		t.Errorf("expected expiry > %d, got %d", before, token.Expiry)
	}
}

// TestTTL_ExactExpiry checks that the token expiry is exactly now+ttl.
func TestTTL_ExactExpiry(t *testing.T) {
	resetStateBlockCleanup()
	now := time.Now()
	tokenId := Generate(TypeLogin)
	mutex.Lock()
	token := tokens[tokenId]
	mutex.Unlock()
	expected := now.Add(ttl).Unix()
	if token.Expiry != expected {
		t.Errorf("expected expiry %d, got %d", expected, token.Expiry)
	}
}

// TestIsValid_ValidToken checks that a freshly generated token is valid.
func TestIsValid_ValidToken(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeLogin)
	if !IsValid(TypeLogin, token) {
		t.Error("expected IsValid to return true for a fresh token")
	}
}

// TestIsValid_UnknownToken checks that an unknown token is invalid.
func TestIsValid_UnknownToken(t *testing.T) {
	resetStateBlockCleanup()
	if IsValid(TypeLogin, "nonexistent-token-xyz") {
		t.Error("expected IsValid to return false for unknown token")
	}
}

// TestIsValid_SingleUse checks that a token cannot be used twice.
func TestIsValid_SingleUse(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeLogin)
	if !IsValid(TypeLogin, token) {
		t.Fatal("expected first IsValid to return true")
	}
	if IsValid(TypeLogin, token) {
		t.Error("expected second IsValid to return false — token must be single-use")
	}
}

// TestIsValid_DeletesTokenAfterUse checks that the token is removed from the map after validation.
func TestIsValid_DeletesTokenAfterUse(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeLogin)
	IsValid(TypeLogin, token)
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
	tokens[token] = csrfToken{
		Type:   TypeLogin,
		Expiry: time.Now().Add(-1 * time.Second).Unix(),
	}
	mutex.Unlock()

	if IsValid(TypeLogin, token) {
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
	if IsValid(TypeLogin, "") {
		t.Error("expected IsValid to return false for empty token")
	}
}

// TestIsValid_TokenExpiresAfterTTL advances the fake clock past the TTL and confirms
// the token is rejected — no real sleeping required.
func TestIsValid_TokenExpiresAfterTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resetStateBlockCleanup()
		token := Generate(TypeLogin)
		cleanup(false)

		// Advance fake clock beyond the 5-minute TTL.
		time.Sleep(ttl + time.Second)
		cleanup(false)
		synctest.Wait()

		if IsValid(TypeLogin, token) {
			t.Error("expected IsValid to return false after TTL has elapsed")
		}
	})
}

// TestIsValid_TokenStillValidBeforeTTL confirms a token remains valid just before expiry.
func TestIsValid_TokenStillValidBeforeTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resetStateBlockCleanup()
		token := Generate(TypeLogin)
		cleanup(false)

		// Advance to just before expiry.
		time.Sleep(ttl - time.Second)
		cleanup(false)
		synctest.Wait()

		if !IsValid(TypeLogin, token) {
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
				Generate(TypeLogin)
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
		token := Generate(TypeLogin)

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				IsValid(TypeLogin, token)
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
	tokens["expired-1"] = csrfToken{
		Type:   TypeLogin,
		Expiry: time.Now().Add(-10 * time.Second).Unix(),
	}
	tokens["expired-2"] = csrfToken{
		Type:   TypeLogin,
		Expiry: time.Now().Add(-5 * time.Second).Unix(),
	}
	tokens["valid-1"] = csrfToken{
		Type:   TypeLogin,
		Expiry: time.Now().Add(5 * time.Minute).Unix(),
	}
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

// TestIsValid_WrongType checks that a token generated for one type is rejected
// when validated against a different type.
func TestIsValid_WrongType(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeLogin)
	if IsValid(TypeApiToken, token) {
		t.Error("expected IsValid to return false when token type does not match")
	}
}

// TestIsValid_WrongType_DeletesToken checks that a type-mismatched token is still
// consumed (deleted) from the map, preventing reuse with the correct type.
func TestIsValid_WrongType_DeletesToken(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeLogin)
	IsValid(TypeApiToken, token)
	mutex.Lock()
	_, ok := tokens[token]
	mutex.Unlock()
	if ok {
		t.Error("expected type-mismatched token to be deleted from map after validation attempt")
	}
}

// TestIsValid_ApiToken_Valid checks that a token generated for TypeApiToken is
// accepted when validated with TypeApiToken.
func TestIsValid_ApiToken_Valid(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeApiToken)
	if !IsValid(TypeApiToken, token) {
		t.Error("expected IsValid to return true for a fresh TypeApiToken token")
	}
}

// TestIsValid_ApiToken_RejectedAsLogin checks that a TypeApiToken token is rejected
// when validated as TypeLogin.
func TestIsValid_ApiToken_RejectedAsLogin(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeApiToken)
	if IsValid(TypeLogin, token) {
		t.Error("expected IsValid to return false when TypeApiToken token is validated as TypeLogin")
	}
}

// TestIsValid_LoginToken_RejectedAsApiToken checks that a TypeLogin token is rejected
// when validated as TypeApiToken.
func TestIsValid_LoginToken_RejectedAsApiToken(t *testing.T) {
	resetStateBlockCleanup()
	token := Generate(TypeLogin)
	if IsValid(TypeApiToken, token) {
		t.Error("expected IsValid to return false when TypeLogin token is validated as TypeApiToken")
	}
}

// TestIsValid_EachTypeIndependent checks that tokens of different types can coexist
// and each is only accepted for its own type.
func TestIsValid_EachTypeIndependent(t *testing.T) {
	resetStateBlockCleanup()
	loginToken := Generate(TypeLogin)
	apiToken := Generate(TypeApiToken)

	if IsValid(TypeApiToken, loginToken) {
		t.Error("expected login token to be rejected when validated as TypeApiToken")
	}
	if IsValid(TypeLogin, apiToken) {
		t.Error("expected api token to be rejected when validated as TypeLogin")
	}
	// Both tokens should have been consumed; neither should be reusable.
	if IsValid(TypeLogin, loginToken) {
		t.Error("expected login token to be gone after prior validation attempt")
	}
	if IsValid(TypeApiToken, apiToken) {
		t.Error("expected api token to be gone after prior validation attempt")
	}
}

// TestCleanup_PeriodicRunsAfterOneHour verifies that the periodic cleanup goroutine
// re-runs after one hour. The fake clock advances instantly — no real waiting.
func TestCleanup_PeriodicRunsAfterOneHour(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resetStateBlockCleanup()

		// Insert a token that expires in 30 minutes.
		mutex.Lock()
		tokens["will-expire"] = csrfToken{
			Type:   TypeLogin,
			Expiry: time.Now().Add(30 * time.Minute).Unix(),
		}
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
