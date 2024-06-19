package redis

import (
	"github.com/forceu/gokapi/internal/models"
)

var redisConnection string

type DatabaseProvider struct {
}

func New() DatabaseProvider {
	return DatabaseProvider{}
}

func (p DatabaseProvider) Init(models.DbConnection) {
	// TODO
}
func (p DatabaseProvider) Upgrade(int) {
	// TODO
}
func (p DatabaseProvider) Close() {
	// TODO
}
