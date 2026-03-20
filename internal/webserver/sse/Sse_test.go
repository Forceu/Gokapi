package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"testing/synctest"
	"time"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"github.com/forceu/gokapi/internal/webserver/authentication"
)

const testUserId = 7

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	configuration.ConnectDatabase()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestAddListener(t *testing.T) {
	channel := listener{Reply: func(reply string) {}, Shutdown: func() {}, UserId: testUserId}
	addListener("test_id", channel)

	mutex.RLock()
	_, exists := listeners["test_id"]
	mutex.RUnlock()
	test.IsEqualBool(t, exists, true)
}

func TestRemoveListener(t *testing.T) {
	removeListener("test_id")

	mutex.RLock()
	_, exists := listeners["test_id"]
	mutex.RUnlock()
	test.IsEqualBool(t, exists, false)
}

func TestPublishNewStatus(t *testing.T) {
	// Use a buffered channel matching the production listener contract so that
	// the non-blocking Reply closure never blocks or drops during the test.
	replyChannel := make(chan string, 64)
	channel := listener{
		Reply: func(reply string) {
			select {
			case replyChannel <- reply:
			default:
			}
		},
		Shutdown: func() {},
		UserId:   testUserId,
	}
	addListener("test_id", channel)

	// Message for the correct user must be delivered.
	PublishNewStatus(models.UploadStatus{
		ChunkId:       "testChunkId",
		CurrentStatus: 4,
		UserId:        testUserId,
	})
	receivedStatus := <-replyChannel
	test.IsEqualString(t, receivedStatus, "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"testChunkId\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":4}\n\n")

	// Message for a different user must NOT be delivered to this listener.
	PublishNewStatus(models.UploadStatus{
		ChunkId:       "otherUserChunk",
		CurrentStatus: 4,
		UserId:        testUserId + 1,
	})
	select {
	case unexpectedMsg := <-replyChannel:
		t.Errorf("listener received message intended for a different user: %s", unexpectedMsg)
	case <-time.After(100 * time.Millisecond):
		// expected: nothing received
	}

	PublishDownloadCount(models.File{
		Id:                 "testFileId",
		DownloadCount:      3,
		DownloadsRemaining: 1,
		UnlimitedDownloads: false,
		UserId:             testUserId,
	})
	receivedStatus = <-replyChannel
	test.IsEqualString(t, receivedStatus, "event: message\ndata: {\"event\":\"download\",\"file_id\":\"testFileId\",\"download_count\":3,\"downloads_remaining\":1}\n\n")

	// Download event for a different user must NOT be delivered.
	PublishDownloadCount(models.File{
		Id:     "otherFileId",
		UserId: testUserId + 1,
	})
	select {
	case unexpectedMsg := <-replyChannel:
		t.Errorf("listener received download event intended for a different user: %s", unexpectedMsg)
	case <-time.After(100 * time.Millisecond):
		// expected: nothing received
	}

	PublishDownloadCount(models.File{
		Id:                 "testFileId",
		DownloadCount:      3,
		DownloadsRemaining: 2,
		UnlimitedDownloads: true,
		UserId:             testUserId,
	})
	receivedStatus = <-replyChannel
	test.IsEqualString(t, receivedStatus, "event: message\ndata: {\"event\":\"download\",\"file_id\":\"testFileId\",\"download_count\":3,\"downloads_remaining\":-1}\n\n")

	removeListener("test_id")
}

func TestShutdown(t *testing.T) {
	// Buffered so the non-blocking Shutdown closure never drops the signal.
	shutdownChannel := make(chan bool, 1)
	channel := listener{
		Reply: func(reply string) {},
		Shutdown: func() {
			select {
			case shutdownChannel <- true:
			default:
			}
		},
		UserId: testUserId,
	}
	addListener("test_id", channel)

	go Shutdown()
	receivedShutdown := <-shutdownChannel
	test.IsEqualBool(t, receivedShutdown, true)
	removeListener("test_id")
}

// TestPublishNewStatus_SlowClient verifies that publishing to a stalled client
// does not leak goroutines. A slow client is simulated by a Reply closure that
// never drains its channel, causing every non-blocking send to hit the default
// branch and be dropped. After a large number of publishes the goroutine count
// must not have grown.
func TestPublishNewStatus_SlowClient(t *testing.T) {
	// A full, never-drained channel simulates a client that has stopped reading.
	replyChannel := make(chan string) // unbuffered and never read
	channel := listener{
		Reply: func(reply string) {
			select {
			case replyChannel <- reply:
			default:
				// drop — client is too slow
			}
		},
		Shutdown: func() {},
		UserId:   testUserId,
	}
	addListener("slow_client", channel)
	defer removeListener("slow_client")

	before := runtime.NumGoroutine()

	const publishes = 500
	for i := 0; i < publishes; i++ {
		PublishNewStatus(models.UploadStatus{
			ChunkId:       "slowChunk",
			CurrentStatus: i,
			UserId:        testUserId,
		})
	}

	// Give any accidentally spawned goroutines time to surface.
	runtime.Gosched()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()

	// Allow a small margin for unrelated runtime goroutines, but nowhere near
	// the number of publishes.
	const margin = 5
	if after-before > margin {
		t.Errorf("goroutine leak detected: started with %d, now %d after %d publishes (margin %d)",
			before, after, publishes, margin)
	}
}

func TestGetStatusSSE_TimeoutWithSyncTest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/statusUpdate", nil)

		done := make(chan struct{})
		go func() {
			GetStatusSSE(rr, req)
			close(done)
		}()

		synctest.Wait()

		time.Sleep(maxConnection + 1*time.Second)
		time.Sleep(pingInterval)
		<-done

		mutex.RLock()
		count := len(listeners)
		mutex.RUnlock()

		test.IsEqualInt(t, count, 0)
	})
}

func TestGetStatusSSE_ContextCancelWithSyncTest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/statusUpdate", nil)
		req = authentication.SetUserInRequest(req, models.User{Id: testUserId})

		ctx, cancel := context.WithCancel(req.Context())
		req = req.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			GetStatusSSE(rr, req)
			close(done)
		}()

		synctest.Wait()

		mutex.RLock()
		test.IsEqualBool(t, len(listeners) > 0, true)
		mutex.RUnlock()

		cancel()
		<-done

		mutex.RLock()
		count := len(listeners)
		mutex.RUnlock()

		test.IsEqualInt(t, count, 0)
	})
}

func TestGetStatusSSE(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		req, err := http.NewRequest("GET", "/statusUpdate", nil)
		req = authentication.SetUserInRequest(req, models.User{Id: testUserId})
		test.IsNil(t, err)

		rr := httptest.NewRecorder()
		done := make(chan struct{})

		go func() {
			GetStatusSSE(rr, req)
			close(done)
		}()

		synctest.Wait()

		// Test response headers (set immediately before the loop).
		test.IsEqualString(t, rr.Header().Get("Content-Type"), "text/event-stream")
		test.IsEqualString(t, rr.Header().Get("X-Accel-Buffering"), "no")

		// Test initial statuses (pstatusdb.GetAllForUser). Written directly to
		// the response writer before the select loop, reliably present after
		// synctest.Wait().
		bodyString := rr.Body.String()
		isCorrect0 := bodyString == "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_0\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":0}\n\n"+
			"event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_1\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":1}\n\n"
		isCorrect1 := bodyString == "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_1\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":1}\n\n"+
			"event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_0\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":0}\n\n"
		test.IsEqualBool(t, isCorrect0 || isCorrect1, true)

		rr.Body.Reset()

		// Test ping message.
		time.Sleep(pingInterval)
		synctest.Wait()
		test.IsEqualString(t, rr.Body.String(), "event: ping\n\n")
		rr.Body.Reset()

		// Test PublishNewStatus for the correct user -- must be received.
		// Called directly (not via go) since Reply is non-blocking.
		PublishNewStatus(models.UploadStatus{
			ChunkId:       "secondChunkId",
			CurrentStatus: 1,
			UserId:        testUserId,
		})
		synctest.Wait()
		test.IsEqualString(t, rr.Body.String(), "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"secondChunkId\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":1}\n\n")
		rr.Body.Reset()

		// Test PublishNewStatus for a different user -- must NOT appear in the response.
		PublishNewStatus(models.UploadStatus{
			ChunkId:       "otherUserChunk",
			CurrentStatus: 1,
			UserId:        testUserId + 1,
		})
		synctest.Wait()
		test.IsEqualString(t, rr.Body.String(), "")
		rr.Body.Reset()

		// Test a status update with all fields populated.
		PublishNewStatus(models.UploadStatus{
			ChunkId:       "secondChunkId",
			CurrentStatus: 2,
			FileId:        "testfile",
			ErrorMessage:  "123",
			UserId:        testUserId,
		})
		synctest.Wait()
		test.IsEqualString(t, rr.Body.String(), "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"secondChunkId\",\"file_id\":\"testfile\",\"error_message\":\"123\",\"upload_status\":2}\n\n")

		Shutdown()
		<-done
	})
}
