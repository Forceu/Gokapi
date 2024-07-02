package migration

import (
	"fmt"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/configuration/database/dbabstraction"
	"github.com/forceu/gokapi/internal/environment/flagparser"
	"os"
)

func Do(flags flagparser.MigrateFlags) {
	oldDb, err := database.ParseUrl(flags.Source, true)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}
	newDb, err := database.ParseUrl(flags.Destination, false)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Printf("Migrating %s database %s to %s database %s\n", getType(oldDb.Type), oldDb.HostUrl, getType(newDb.Type), newDb.HostUrl)
	database.Migrate(oldDb, newDb)
}

func getType(input int) string {
	switch input {
	case dbabstraction.TypeSqlite:
		return "SQLite"
	case dbabstraction.TypeRedis:
		return "Redis"
	}
	return "Invalid"
}
