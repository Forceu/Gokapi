package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	configuration.ConnectDatabase()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestAddListener(t *testing.T) {
	channel := listener{Reply: func(reply string) {}, Shutdown: func() {}}
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
	replyChannel := make(chan string)
	channel := listener{Reply: func(reply string) { replyChannel <- reply }, Shutdown: func() {}}
	addListener("test_id", channel)

	go PublishNewStatus(models.UploadStatus{
		ChunkId:       "testChunkId",
		CurrentStatus: 4,
	})
	receivedStatus := <-replyChannel
	test.IsEqualString(t, receivedStatus, "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"testChunkId\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":4}\n\n")

	go PublishDownloadCount(models.File{
		Id:                 "testFileId",
		DownloadCount:      3,
		DownloadsRemaining: 1,
		UnlimitedDownloads: false,
	})
	receivedStatus = <-replyChannel
	test.IsEqualString(t, receivedStatus, "event: message\ndata: {\"event\":\"download\",\"file_id\":\"testFileId\",\"download_count\":3,\"downloads_remaining\":1}\n\n")

	go PublishDownloadCount(models.File{
		Id:                 "testFileId",
		DownloadCount:      3,
		DownloadsRemaining: 2,
		UnlimitedDownloads: true,
	})
	receivedStatus = <-replyChannel
	test.IsEqualString(t, receivedStatus, "event: message\ndata: {\"event\":\"download\",\"file_id\":\"testFileId\",\"download_count\":3,\"downloads_remaining\":-1}\n\n")
	removeListener("test_id")
}

func TestShutdown(t *testing.T) {
	shutdownChannel := make(chan bool)
	channel := listener{Reply: func(reply string) {}, Shutdown: func() { shutdownChannel <- true }}
	addListener("test_id", channel)

	go Shutdown()
	receivedShutdown := <-shutdownChannel
	test.IsEqualBool(t, receivedShutdown, true)
	removeListener("test_id")
}

func TestGetStatusSSE_TimeoutWithSyncTest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/statusUpdate", nil)

		// Use a channel to signal when the handler has actually finished
		done := make(chan struct{})

		go func() {
			GetStatusSSE(rr, req)
			close(done) // Signal completion
		}()

		synctest.Wait()

		time.Sleep(maxConnection + 1*time.Second)
		time.Sleep(pingInterval)
		// Wait for the goroutine to finish its last loop and exit
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
		test.IsNil(t, err)

		rr := httptest.NewRecorder()
		done := make(chan struct{})

		go func() {
			GetStatusSSE(rr, req)
			close(done)
		}()

		synctest.Wait()

		// Test response headers (Headers are set immediately)
		test.IsEqualString(t, rr.Header().Get("Content-Type"), "text/event-stream")
		test.IsEqualString(t, rr.Header().Get("X-Accel-Buffering"), "no")

		// Test initial data (pstatusdb.GetAll())
		bodyString := rr.Body.String()
		isCorrect0 := bodyString == "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_0\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":0}\n\n"+
			"event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_1\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":1}\n\n"
		isCorrect1 := bodyString == "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_1\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":1}\n\n"+
			"event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_0\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":0}\n\n"
		test.IsEqualBool(t, isCorrect0 || isCorrect1, true)

		// Clear the buffer for next checks
		rr.Body.Reset()

		//  Test ping message
		time.Sleep(pingInterval)
		synctest.Wait() // Ensure the select case and WriteString finish
		test.IsEqualString(t, rr.Body.String(), "event: ping\n\n")
		rr.Body.Reset()

		// Test PublishNewStatus
		PublishNewStatus(models.UploadStatus{
			ChunkId:       "secondChunkId",
			CurrentStatus: 1,
		})
		synctest.Wait() // Wait for the 'go channel.Reply' goroutine to execute
		test.IsEqualString(t, rr.Body.String(), "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"secondChunkId\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":1}\n\n")
		rr.Body.Reset()

		// Test another status update
		PublishNewStatus(models.UploadStatus{
			ChunkId:       "secondChunkId",
			CurrentStatus: 2,
			FileId:        "testfile",
			ErrorMessage:  "123",
		})
		synctest.Wait()
		test.IsEqualString(t, rr.Body.String(), "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"secondChunkId\",\"file_id\":\"testfile\",\"error_message\":\"123\",\"upload_status\":2}\n\n")

		Shutdown()
		<-done // Wait for GetStatusSSE to return via shutdownChannel
	})
}
