package dbabstraction

import (
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"testing"
)

func TestGetNew(t *testing.T) {
	result := GetNew(models.DbConnection{Type: 0})
	test.IsEqualInt(t, result.GetType(), 0)
	result = GetNew(models.DbConnection{Type: 1})
	test.IsEqualInt(t, result.GetType(), 1)

	defer test.ExpectPanic(t)
	_ = GetNew(models.DbConnection{Type: 2})
}
