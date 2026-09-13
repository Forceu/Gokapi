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

var configMariaDb = models.DbConnection{
	Type: 2, // dbabstraction.TypeMariaDb
}

var configPostgres = models.DbConnection{
	Type: 3, // dbabstraction.TypePostgres
}

func TestGetNew(t *testing.T) {
	result, err := GetNew(configSqlite)
	test.IsNotNil(t, err)
	test.IsEqualInt(t, result.GetType(), 0)
	result, err = GetNew(configRedis)
	test.IsNotNil(t, err)
	test.IsEqualInt(t, result.GetType(), 1)
	result, err = GetNew(configMariaDb)
	test.IsNotNil(t, err)
	test.IsEqualInt(t, result.GetType(), 2)
	result, err = GetNew(configPostgres)
	test.IsNotNil(t, err)
	test.IsEqualInt(t, result.GetType(), 3)

	_, err = GetNew(models.DbConnection{Type: 4})
	test.IsNotNil(t, err)
}
