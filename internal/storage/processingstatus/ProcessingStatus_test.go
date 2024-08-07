package processingstatus

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	configuration.ConnectDatabase()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestSetStatus(t *testing.T) {
	const id = "testchunk"
	status, ok := database.GetUploadStatus(id)
	test.IsEqualBool(t, ok, false)
	test.IsEmpty(t, status.ChunkId)
	Set(id, 2)
	status, ok = database.GetUploadStatus(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, status.ChunkId, id)
	test.IsEqualInt(t, status.CurrentStatus, 2)
	Set(id, 1)
	status, ok = database.GetUploadStatus(id)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, status.ChunkId, id)
	test.IsEqualInt(t, status.CurrentStatus, 2)

}
