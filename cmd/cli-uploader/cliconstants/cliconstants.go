package cliconstants

// MinGokapiVersionInt is the minimum version of the gokapi server that is supported by the cli
const MinGokapiVersionInt = 20200

// MinGokapiVersionStr is the minimum version of the gokapi server that is supported by the cli
const MinGokapiVersionStr = "2.2.0"

// DefaultConfigFileName is the default config file name
const DefaultConfigFileName = "gokapi-cli.json"

// DefaultUserConfigPathNoHome is the second config path, if ./DefaultConfigFileName does not exist
// Important: This path requires looking up with os.UserHomeDir() first
const DefaultUserConfigPathNoHome = ".config/gokapi-cli/" + DefaultConfigFileName

// DefaultUserConfigPathGlobal is the last config path, if no other path exists
const DefaultUserConfigPathGlobal = "/etc/gokapi-cli/" + DefaultConfigFileName

// DockerFolderConfig is the default config folder for an docker instance
const DockerFolderConfig = "/app/config/"

// DockerFolderConfigFile is the default config path for a docker instance
const DockerFolderConfigFile = DockerFolderConfig + "config.json"

// DockerFolderUpload is the default upload folder for a docker instance
const DockerFolderUpload = "/upload/"
