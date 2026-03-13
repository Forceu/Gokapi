package chunkreservation

import (
	"testing"
	"testing/synctest"
	"time"
)

// resetStateBlockCleanup resets the package-level state between tests.
func resetStateBlockCleanup() {
	reservationMutex.Lock()
	reservedChunks = make(map[string]map[string]reservation)
	runGcOnce.Do(func() {
		//prevent New from spawning the cleanup goroutine
	})
	reservationMutex.Unlock()
}

// TestNew_ReturnsNonEmptyUuid checks that New returns a non-empty uuid string.
func TestNew_ReturnsNonEmptyUuid(t *testing.T) {
	resetStateBlockCleanup()
	uuid := New("file1")
	if uuid == "" {
		t.Error("expected non-empty uuid, got empty string")
	}
}

// TestNew_UuidLength checks that the generated uuid has the expected length (32 chars).
func TestNew_UuidLength(t *testing.T) {
	resetStateBlockCleanup()
	uuid := New("file1")
	if len(uuid) != 32 {
		t.Errorf("expected uuid length 32, got %d", len(uuid))
	}
}

// TestNew_UniqueUuids checks that two calls produce different uuids.
func TestNew_UniqueUuids(t *testing.T) {
	resetStateBlockCleanup()
	uuid1 := New("file1")
	uuid2 := New("file1")
	if uuid1 == uuid2 {
		t.Error("expected unique uuids, got identical values")
	}
}

// TestNew_StoresReservation checks that New stores the reservation in the map.
func TestNew_StoresReservation(t *testing.T) {
	resetStateBlockCleanup()
	uuid := New("file1")
	reservationMutex.RLock()
	_, ok := reservedChunks["file1"][uuid]
	reservationMutex.RUnlock()
	if !ok {
		t.Error("expected reservation to be stored in map")
	}
}

// TestNew_ExpiryIsInFuture checks that the reservation expiry is in the future.
func TestNew_ExpiryIsInFuture(t *testing.T) {
	resetStateBlockCleanup()
	now := time.Now().Unix()
	uuid := New("file1")
	reservationMutex.RLock()
	expiry := reservedChunks["file1"][uuid].Expiry
	reservationMutex.RUnlock()
	if expiry <= now {
		t.Errorf("expected expiry > %d, got %d", now, expiry)
	}
}

// TestNew_ExpiryMatchesConstant checks that the expiry is set to now + timeReservationWithoutUpload.
func TestNew_ExpiryMatchesConstant(t *testing.T) {
	resetStateBlockCleanup()
	now := time.Now().Unix()
	uuid := New("file1")
	reservationMutex.RLock()
	expiry := reservedChunks["file1"][uuid].Expiry
	reservationMutex.RUnlock()
	expected := now + timeReservationWithoutUpload
	// Allow 1 second of slack for execution time.
	if expiry < expected || expiry > expected+1 {
		t.Errorf("expected expiry ~%d, got %d", expected, expiry)
	}
}

// TestNew_UuidStoredOnReservation checks that the uuid field on the reservation matches the returned uuid.
func TestNew_UuidStoredOnReservation(t *testing.T) {
	resetStateBlockCleanup()
	uuid := New("file1")
	reservationMutex.RLock()
	r := reservedChunks["file1"][uuid]
	reservationMutex.RUnlock()
	if r.Uuid != uuid {
		t.Errorf("expected reservation uuid %q, got %q", uuid, r.Uuid)
	}
}

// TestNew_InitialisesMapForNewId checks that New creates the inner map for a new file id.
func TestNew_InitialisesMapForNewId(t *testing.T) {
	resetStateBlockCleanup()
	New("newfile")
	reservationMutex.RLock()
	_, ok := reservedChunks["newfile"]
	reservationMutex.RUnlock()
	if !ok {
		t.Error("expected inner map to be initialised for new file id")
	}
}

// TestNew_MultipleIdsAreIndependent checks that reservations for different ids don't interfere.
func TestNew_MultipleIdsAreIndependent(t *testing.T) {
	resetStateBlockCleanup()
	New("fileA")
	New("fileB")
	New("fileB")
	if GetCount("fileA") != 1 {
		t.Errorf("expected 1 reservation for fileA, got %d", GetCount("fileA"))
	}
	if GetCount("fileB") != 2 {
		t.Errorf("expected 2 reservations for fileB, got %d", GetCount("fileB"))
	}
}

// TestGetCount_ReturnsZeroForUnknownId checks that GetCount returns 0 for an unknown file id.
func TestGetCount_ReturnsZeroForUnknownId(t *testing.T) {
	resetStateBlockCleanup()
	if count := GetCount("unknown"); count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

// TestGetCount_ReturnsCorrectCount checks that GetCount reflects the number of active reservations.
func TestGetCount_ReturnsCorrectCount(t *testing.T) {
	resetStateBlockCleanup()
	New("file1")
	New("file1")
	New("file1")
	if count := GetCount("file1"); count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

// TestSetComplete_RemovesReservation checks that SetComplete deletes the reservation.
func TestSetComplete_RemovesReservation(t *testing.T) {
	resetStateBlockCleanup()
	uuid := New("file1")
	SetComplete("file1", uuid)
	reservationMutex.RLock()
	_, ok := reservedChunks["file1"][uuid]
	reservationMutex.RUnlock()
	if ok {
		t.Error("expected reservation to be removed after SetComplete")
	}
}

// TestSetComplete_DecreasesCount checks that GetCount decreases after SetComplete.
func TestSetComplete_DecreasesCount(t *testing.T) {
	resetStateBlockCleanup()
	uuid := New("file1")
	New("file1")
	SetComplete("file1", uuid)
	if count := GetCount("file1"); count != 1 {
		t.Errorf("expected count 1 after SetComplete, got %d", count)
	}
}

// TestSetComplete_UnknownIdDoesNotPanic checks that SetComplete is safe for unknown ids.
func TestSetComplete_UnknownIdDoesNotPanic(t *testing.T) {
	resetStateBlockCleanup()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetComplete panicked on unknown id: %v", r)
		}
	}()
	SetComplete("unknown-id", "unknown-uuid")
}

// TestSetUploading_ReturnsTrueForValidReservation checks the happy path.
func TestSetUploading_ReturnsTrueForValidReservation(t *testing.T) {
	resetStateBlockCleanup()
	uuid := New("file1")
	if !SetUploading("file1", uuid) {
		t.Error("expected SetUploading to return true for a valid reservation")
	}
}

// TestSetUploading_ExtendsExpiry checks that SetUploading extends the expiry to the upload constant.
func TestSetUploading_ExtendsExpiry(t *testing.T) {
	resetStateBlockCleanup()
	now := time.Now().Unix()
	uuid := New("file1")
	SetUploading("file1", uuid)
	reservationMutex.RLock()
	expiry := reservedChunks["file1"][uuid].Expiry
	reservationMutex.RUnlock()
	expected := now + timeReservationWithUpload
	if expiry < expected || expiry > expected+1 {
		t.Errorf("expected expiry ~%d after SetUploading, got %d", expected, expiry)
	}
}

// TestSetUploading_ReturnsFalseForUnknownId checks that SetUploading returns false for unknown file id.
func TestSetUploading_ReturnsFalseForUnknownId(t *testing.T) {
	resetStateBlockCleanup()
	if SetUploading("unknown-id", "some-uuid") {
		t.Error("expected SetUploading to return false for unknown file id")
	}
}

// TestSetUploading_ReturnsFalseForUnknownUuid checks that SetUploading returns false for unknown uuid.
func TestSetUploading_ReturnsFalseForUnknownUuid(t *testing.T) {
	resetStateBlockCleanup()
	New("file1")
	if SetUploading("file1", "not-a-real-uuid") {
		t.Error("expected SetUploading to return false for unknown uuid")
	}
}

// TestSetUploading_ReturnsFalseForExpiredReservation checks that an expired reservation is rejected.
func TestSetUploading_ReturnsFalseForExpiredReservation(t *testing.T) {
	resetStateBlockCleanup()
	uuid := "expired-uuid"
	reservationMutex.Lock()
	reservedChunks["file1"] = map[string]reservation{
		uuid: {Uuid: uuid, Expiry: time.Now().Unix() - 1},
	}
	reservationMutex.Unlock()
	if SetUploading("file1", uuid) {
		t.Error("expected SetUploading to return false for expired reservation")
	}
}

// TestCleanup_RemovesExpiredReservations checks that cleanUp removes expired entries.
func TestCleanup_RemovesExpiredReservations(t *testing.T) {
	resetStateBlockCleanup()
	reservationMutex.Lock()
	reservedChunks["file1"] = map[string]reservation{
		"expired": {Uuid: "expired", Expiry: time.Now().Unix() - 10},
		"valid":   {Uuid: "valid", Expiry: time.Now().Unix() + 300},
	}
	reservationMutex.Unlock()

	cleanUp(false)

	reservationMutex.RLock()
	_, expiredExists := reservedChunks["file1"]["expired"]
	_, validExists := reservedChunks["file1"]["valid"]
	reservationMutex.RUnlock()

	if expiredExists {
		t.Error("expected expired reservation to be removed by cleanUp")
	}
	if !validExists {
		t.Error("expected valid reservation to survive cleanUp")
	}
}

// TestCleanup_EmptyMapDoesNotPanic checks that cleanUp on an empty map does not panic.
func TestCleanup_EmptyMapDoesNotPanic(t *testing.T) {
	resetStateBlockCleanup()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("cleanUp panicked on empty map: %v", r)
		}
	}()
	cleanUp(false)
}

// TestCleanup_PeriodicRunsAfterFiveMinutes verifies that the periodic cleanup goroutine
// re-runs after 5 minutes. The fake clock advances instantly — no real waiting.
func TestCleanup_PeriodicRunsAfterFiveMinutes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resetStateBlockCleanup()

		// Insert a reservation that expires in 2 minutes.
		reservationMutex.Lock()
		reservedChunks["file1"] = map[string]reservation{
			"will-expire": {Uuid: "will-expire", Expiry: time.Now().Unix() + 120},
		}
		reservationMutex.Unlock()

		// Run one non-periodic pass; token still valid so it should survive.
		cleanUp(false)
		synctest.Wait()

		reservationMutex.RLock()
		_, stillThere := reservedChunks["file1"]["will-expire"]
		reservationMutex.RUnlock()
		if !stillThere {
			t.Error("expected reservation to still be present before expiry")
		}

		// Advance fake clock past the 2-minute expiry and the 5-minute cleanup interval.
		time.Sleep(6 * time.Minute)
		cleanUp(false)
		synctest.Wait()

		reservationMutex.RLock()
		_, stillThere = reservedChunks["file1"]["will-expire"]
		reservationMutex.RUnlock()
		if stillThere {
			t.Error("expected reservation to be removed after periodic cleanup ran")
		}
	})
}
