package apiMutex

import (
	"fmt"
	"sync"
	"testing"

	"github.com/forceu/gokapi/internal/test"
)

// TestInvalidObjectTypePanics verifies that an unknown objectType causes a panic.
func TestInvalidObjectTypePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid object type, but did not panic")
		}
	}()

	getStripe(999, "any")
}

// TestAllValidTypesDoNotPanic verifies that all defined types are accepted.
func TestAllValidTypesDoNotPanic(t *testing.T) {
	types := []int{TypeUser, TypeApiKey, TypeMetaData}
	for _, objectType := range types {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("unexpected panic for objectType %d", objectType)
				}
			}()
			getStripe(objectType, "key")
		}()
	}
}

// TestSameKeyReturnsSameStripe verifies that the same arguments always map to the same stripe.
func TestSameKeyReturnsSameStripe(t *testing.T) {
	m1 := getStripe(TypeUser, "abc")
	m2 := getStripe(TypeUser, "abc")

	test.IsEqual(t, m1, m2)
}

// TestDifferentTypesSameKeyMayDiffer verifies that objectType is factored into the hash.
func TestDifferentTypesSameKeyMayDiffer(t *testing.T) {
	collisions := 0
	keys := []string{"1", "2", "3", "4", "5"}
	for _, key := range keys {
		if getStripe(TypeUser, key) == getStripe(TypeApiKey, key) {
			collisions++
		}
	}
	if collisions == len(keys) {
		t.Error("objectType does not appear to affect stripe selection — all keys collided across types")
	}
}

// TestLockUnlock verifies that Lock and Unlock work without deadlocking.
func TestLockUnlock(t *testing.T) {
	done := make(chan struct{})
	go func() {
		Lock(TypeUser, "1")
		Unlock(TypeUser, "1")
		close(done)
	}()
	<-done
}

// TestLockBlocksUntilUnlocked verifies that a second Lock on the same key blocks
// until the first caller calls Unlock.
func TestLockBlocksUntilUnlocked(t *testing.T) {
	// Find two keys that hash to the same stripe so we can test real blocking.
	key := "1"
	stripe := getStripe(TypeUser, key)

	stripe.Lock()

	acquired := make(chan struct{})
	go func() {
		Lock(TypeUser, key)
		close(acquired)
		Unlock(TypeUser, key)
	}()

	select {
	case <-acquired:
		t.Error("second Lock acquired while first was still held")
	default:
		// expected: goroutine is blocked
	}

	stripe.Unlock()
	<-acquired
}

// TestConcurrentWritesSerialized verifies that concurrent writers on the same key
// do not race and that the counter reaches the expected value.
func TestConcurrentWritesSerialized(t *testing.T) {
	const goroutines = 100
	counter := 0
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			Lock(TypeUser, "counter")
			counter++
			Unlock(TypeUser, "counter")
		}()
	}

	wg.Wait()

	test.IsEqualInt(t, counter, goroutines)
}

// TestIndependentTypesDoNotNecessarilyBlock verifies that different object types
// with different keys are handled independently across the stripe space.
func TestIndependentTypesDoNotNecessarilyBlock(t *testing.T) {
	// Only meaningful if they hash to different stripes — find such a pair.
	var keyA, keyB string
	found := false
	for i := 0; i < 1000; i++ {
		a := fmt.Sprintf("key-%d", i)
		b := fmt.Sprintf("key-%d", i+1)
		if getStripe(TypeUser, a) != getStripe(TypeApiKey, b) {
			keyA, keyB = a, b
			found = true
			break
		}
	}

	if !found {
		t.Skip("could not find two keys hashing to different stripes")
	}

	Lock(TypeUser, keyA)
	defer Unlock(TypeUser, keyA)

	done := make(chan struct{})
	go func() {
		Lock(TypeApiKey, keyB)
		Unlock(TypeApiKey, keyB)
		close(done)
	}()

	<-done
}
