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
	portFlagLong := passedFlags.Int("port", 0, "Sets the port for setup. Same as env variable GOKAPI_PORT")
	portFlagShort := passedFlags.Int("p", 0, "alias")
	aliases = append(aliases, alias{
		Long:  "port",
		Short: "p",
	})
	disableCorsCheck := passedFlags.Bool("disable-cors-check", false, "Disables the CORS check on startup")

	passedFlags.Usage = showUsage(passedFlags, aliases)
	err := passedFlags.Parse(os.Args[1:])

	if err != nil {
		if err.Error() == "flag: help requested" {
			os.Exit(0)
		}
		os.Exit(2)
	}

	result := MainFlags{
		ShowVersion:      *versionFlagShort || *versionFlagLong,
		Reconfigure:      *reconfigureFlag,
		CreateSsl:        *createSslFlag,
		ConfigPath:       getAliasedString(configPathFlagLong, configPathFlagShort),
		ConfigDir:        getAliasedString(configDirFlagLong, configDirFlagShort),
		DataDir:          getAliasedString(dataDirFlagLong, dataDirFlagShort),
		Port:             getAliasedInt(portFlagLong, portFlagShort),
		DisableCorsCheck: *disableCorsCheck,
	}
	result.setBoolValues()
	return result
}

func showUsage(flags flag.FlagSet, aliases []alias) func() {
	return func() {
		fmt.Print("Usage:\n\n")
		flags.VisitAll(func(f *flag.Flag) {
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
	ShowVersion      bool
	Reconfigure      bool
	CreateSsl        bool
	ConfigPath       string
	ConfigDir        string
	DataDir          string
	Port             int
	IsConfigPathSet  bool
	IsConfigDirSet   bool
	IsDataDirSet     bool
	IsPortSet        bool
	DisableCorsCheck bool
}

func (mf *MainFlags) setBoolValues() {
	mf.IsConfigPathSet = mf.ConfigPath != ""
	mf.IsConfigDirSet = mf.ConfigDir != ""
	mf.IsDataDirSet = mf.DataDir != ""
	mf.IsPortSet = mf.Port != 0
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
