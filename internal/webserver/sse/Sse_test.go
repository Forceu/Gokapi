package sse

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
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

func TestGetStatusSSE(t *testing.T) {

	pingInterval = 2 * time.Second

	// Create request and response recorder
	req, err := http.NewRequest("GET", "/statusUpdate", nil)
	test.IsNil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetStatusSSE)

	go handler.ServeHTTP(rr, req)

	// Wait a bit to ensure handler has started
	time.Sleep(100 * time.Millisecond)

	// Test response headers
	test.IsEqualString(t, rr.Header().Get("Content-Type"), "text/event-stream")
	test.IsEqualString(t, rr.Header().Get("Cache-Control"), "no-cache")
	test.IsEqualString(t, rr.Header().Get("Connection"), "keep-alive")
	test.IsEqualString(t, rr.Header().Get("Keep-Alive"), "timeout=20, max=20")
	test.IsEqualString(t, rr.Header().Get("X-Accel-Buffering"), "no")

	// Test initial data sent
	body, err := io.ReadAll(rr.Body)
	test.IsNil(t, err)

	bodyString := string(body)
	isCorrect0 := bodyString == "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_0\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":0}\n\n"+
		"event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_1\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":1}\n\n"
	isCorrect1 := bodyString == "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_1\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":1}\n\n"+
		"event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"validstatus_0\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":0}\n\n"
	test.IsEqualBool(t, isCorrect0 || isCorrect1, true)

	// Test ping message
	time.Sleep(3 * time.Second)
	body, err = io.ReadAll(rr.Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(body), "event: ping\n\n")

	PublishNewStatus(models.UploadStatus{
		ChunkId:       "secondChunkId",
		CurrentStatus: 1,
	})
	time.Sleep(200 * time.Millisecond)
	body, err = io.ReadAll(rr.Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(body), "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"secondChunkId\",\"file_id\":\"\",\"error_message\":\"\",\"upload_status\":1}\n\n")
	PublishNewStatus(models.UploadStatus{
		ChunkId:       "secondChunkId",
		CurrentStatus: 2,
		FileId:        "testfile",
		ErrorMessage:  "123",
	})
	time.Sleep(200 * time.Millisecond)
	body, err = io.ReadAll(rr.Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(body), "event: message\ndata: {\"event\":\"uploadStatus\",\"chunk_id\":\"secondChunkId\",\"file_id\":\"testfile\",\"error_message\":\"123\",\"upload_status\":2}\n\n")

	Shutdown()
}
