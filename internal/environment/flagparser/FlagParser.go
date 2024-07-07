package flagparser

import (
	"flag"
	"fmt"
	"os"
)

// ParseFlags reads the passed program arguments
func ParseFlags() MainFlags {

	var aliases []alias

	// Is disabled during testing, as otherwise it will raise an error if called in the test's init() function,
	// which replaces the arguments
	if DisableParsing {
		return MainFlags{}
	}
	flags, ok := parseMigration()
	if ok {
		return flags
	}

	passedFlags := flag.FlagSet{}
	versionFlagLong := passedFlags.Bool("version", false, "Show version info")
	versionFlagShort := passedFlags.Bool("v", false, "alias")
	aliases = append(aliases, alias{
		Long:  "version",
		Short: "v",
	})
	reconfigureFlag := passedFlags.Bool("reconfigure", false, "Runs setup again to change Gokapi configuration / passwords")
	createSslFlag := passedFlags.Bool("create-ssl", false, "Creates a new SSL certificate valid for 365 days")
	configPathFlagLong := passedFlags.String("config", "", "Use provided config file instead of the default one in the config directory.\n"+
		"                               Can point to a different directory than default or the one provided by -config-path or env variable GOKAPI_CONFIG_DIR")
	configPathFlagShort := passedFlags.String("c", "", "alias")
	aliases = append(aliases, alias{
		Long:  "config",
		Short: "c",
	})
	configDirFlagLong := passedFlags.String("config-dir", "", "Sets the config directory. Same as env variable GOKAPI_CONFIG_DIR")
	configDirFlagShort := passedFlags.String("cd", "", "alias")
	aliases = append(aliases, alias{
		Long:  "config-dir",
		Short: "cd",
	})
	dataDirFlagLong := passedFlags.String("data", "", "Sets the data directory. Same as env variable GOKAPI_DATA_DIR")
	dataDirFlagShort := passedFlags.String("d", "", "alias")
	aliases = append(aliases, alias{
		Long:  "data",
		Short: "d",
	})
	databaseUrlFlagLong := passedFlags.String("database", "", "Sets the data directory. Same as env variable GOKAPI_DATABASE_URL")
	databaseUrlFlagShort := passedFlags.String("db", "", "alias")
	aliases = append(aliases, alias{
		Long:  "database",
		Short: "db",
	})
	portFlagLong := passedFlags.Int("port", 0, "Sets the port for setup. Same as env variable GOKAPI_PORT")
	portFlagShort := passedFlags.Int("p", 0, "alias")
	aliases = append(aliases, alias{
		Long:  "port",
		Short: "p",
	})
	disableCorsCheck := passedFlags.Bool("disable-cors-check", false, "Disables the CORS check on startup")

	installService := passedFlags.Bool("install-service", false, "Installs Gokapi as a systemd service")
	uninstallService := passedFlags.Bool("uninstall-service", false, "Uninstalls the Gokapi systemd service")
	deploymentPassword := passedFlags.String("deployment-password", "", "Sets a new password. This should only be used for non-interactive deployment")

	passedFlags.Usage = showUsage(passedFlags, aliases)
	err := passedFlags.Parse(os.Args[1:])

	if err != nil {
		if err.Error() == "flag: help requested" {
			os.Exit(0)
		}
		os.Exit(2)
	}

	result := MainFlags{
		ShowVersion:        *versionFlagShort || *versionFlagLong,
		Reconfigure:        *reconfigureFlag,
		CreateSsl:          *createSslFlag,
		DatabaseUrl:        getAliasedString(databaseUrlFlagLong, databaseUrlFlagShort),
		ConfigPath:         getAliasedString(configPathFlagLong, configPathFlagShort),
		ConfigDir:          getAliasedString(configDirFlagLong, configDirFlagShort),
		DataDir:            getAliasedString(dataDirFlagLong, dataDirFlagShort),
		Port:               getAliasedInt(portFlagLong, portFlagShort),
		DisableCorsCheck:   *disableCorsCheck,
		InstallService:     *installService,
		UninstallService:   *uninstallService,
		DeploymentPassword: *deploymentPassword,
	}
	result.setBoolValues()
	return result
}

func parseMigration() (MainFlags, bool) {
	if len(os.Args) > 1 && os.Args[1] == "migrate-database" {
		migrateFlags := parseMigrateFlags(os.Args[2:])
		return MainFlags{
			Migration: migrateFlags}, true
	}
	return MainFlags{}, false
}

func parseMigrateFlags(args []string) MigrateFlags {
	migrateFlags := flag.NewFlagSet("migrate-database", flag.ExitOnError)
	source := migrateFlags.String("source", "", "Source database connection string")
	destination := migrateFlags.String("destination", "", "Destination database connection string")

	migrateFlags.Usage = func() {
		fmt.Println("Usage of migrate-database:")
		migrateFlags.PrintDefaults()
	}

	err := migrateFlags.Parse(args)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	if *source == "" {
		fmt.Println("No source path for migration was passed")
		os.Exit(1)
	}
	if *destination == "" {
		fmt.Println("No destination path for migration was passed")
		os.Exit(1)
	}
	if *source == *destination {
		fmt.Println("Source and destination path cannot be the same")
		os.Exit(1)
	}

	return MigrateFlags{
		DoMigration: true,
		Source:      *source,
		Destination: *destination,
	}
}

// MigrateFlags contains flags passed if migration is requested
type MigrateFlags struct {
	DoMigration bool
	Source      string
	Destination string
}

func showUsage(mainFlags flag.FlagSet, aliases []alias) func() {
	return func() {
		fmt.Print("Usage:\n\n")
		mainFlags.VisitAll(func(f *flag.Flag) {
			if isAlias(f.Name, aliases) {
				return
			}
			output := "--" + f.Name
			aliasExists, aliasName := hasAlias(f.Name, aliases)
			if aliasExists {
				output = "-" + aliasName + ", " + output
			}
			if f.DefValue == "" {
				output = output + " <string>"
			}
			if f.DefValue == "0" {
				output = output + " <int>"
			}
			fmt.Printf("%-30s %s\n", output, f.Usage)
		})
		fmt.Printf("\n%-30s %s\n", "migrate-database", "Migrate an old database to a new database (e.g. SQLite to Redis)")
		fmt.Printf("%-30s %s\n", "--source", "Original database path")
		fmt.Printf("%-30s %s\n", "--destination", "New database path")

	}
}

func getAliasedString(flag1, flag2 *string) string {
	if *flag1 != "" {
		return *flag1
	}
	return *flag2
}
func getAliasedInt(flag1, flag2 *int) int {
	if *flag1 != 0 {
		return *flag1
	}
	return *flag2
}

// MainFlags holds info for the parsed program arguments
type MainFlags struct {
	ConfigPath         string
	ConfigDir          string
	DataDir            string
	DatabaseUrl        string
	DeploymentPassword string
	ShowVersion        bool
	Reconfigure        bool
	CreateSsl          bool
	IsConfigPathSet    bool
	IsConfigDirSet     bool
	IsDataDirSet       bool
	IsPortSet          bool
	IsDatabaseUrlSet   bool
	DisableCorsCheck   bool
	InstallService     bool
	UninstallService   bool
	Port               int
	Migration          MigrateFlags
}

func (mf *MainFlags) setBoolValues() {
	mf.IsConfigPathSet = mf.ConfigPath != ""
	mf.IsConfigDirSet = mf.ConfigDir != ""
	mf.IsDataDirSet = mf.DataDir != ""
	mf.IsPortSet = mf.Port != 0
	mf.IsDatabaseUrlSet = mf.DatabaseUrl != ""
}

type alias struct {
	Long  string
	Short string
}

func isAlias(value string, aliases []alias) bool {
	for _, name := range aliases {
		if name.Short == value {
			return true
		}
	}
	return false
}
func hasAlias(value string, aliases []alias) (bool, string) {
	for _, name := range aliases {
		if name.Long == value {
			return true, name.Short
		}
	}
	return false, ""
}
