package e2emutex

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestAutoUnlockAfter10Seconds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		Lock(1)

		// Simulate 10 seconds passing
		time.Sleep(autoUnlockDuration)
		synctest.Wait()

		// Should be unlockable again (i.e. auto-unlock fired)
		done := make(chan struct{})
		go func() {
			Lock(1)
			close(done)
		}()
		synctest.Wait()

		select {
		case <-done:
			// success: mutex was auto-released
		default:
			t.Fatal("expected mutex to be auto-unlocked after 10 seconds")
		}

		Unlock(1)
	})
}

func TestManualUnlockStopsTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		Lock(2)
		Unlock(2) // manual unlock before 10 seconds

		// Simulate 10 seconds passing — timer should have been stopped
		time.Sleep(autoUnlockDuration)
		synctest.Wait()

		// Lock again and ensure no double-unlock panic occurs
		Lock(2)
		Unlock(2)
	})
}

func TestUnlockWithoutLockIsNoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic on unlock without lock: %v", r)
			}
		}()

		// Should not panic or fatal
		Unlock(3)
	})
}

func TestAutoUnlockThenManualUnlockIsNoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic: %v", r)
			}
		}()

		Lock(4)

		// Let auto-unlock fire
		time.Sleep(autoUnlockDuration)
		synctest.Wait()

		// Manual unlock after auto-unlock should be a safe noop
		Unlock(4)
	})
}
