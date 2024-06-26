package dbabstraction

import (
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"testing"
)

var configSqlite = models.DbConnection{
	Type: 0, // dbabstraction.TypeSqlite
}

var configRedis = models.DbConnection{
	Type: 1, // dbabstraction.TypeRedis
}

func TestGetNew(t *testing.T) {
	result := GetNew(configSqlite)
	test.IsEqualInt(t, result.GetType(), 0)
	result = GetNew(configRedis)
	test.IsEqualInt(t, result.GetType(), 1)

	defer test.ExpectPanic(t)
	_ = GetNew(models.DbConnection{Type: 2})
}
