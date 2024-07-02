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
	result, err := GetNew(configSqlite)
	test.IsNotNil(t, err)
	test.IsEqualInt(t, result.GetType(), 0)
	result, err = GetNew(configRedis)
	test.IsNotNil(t, err)
	test.IsEqualInt(t, result.GetType(), 1)

	_, err = GetNew(models.DbConnection{Type: 2})
	test.IsNotNil(t, err)
}
