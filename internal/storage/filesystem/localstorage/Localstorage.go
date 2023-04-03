package localstorage

import (
	"errors"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	fileInterfaces "github.com/forceu/gokapi/internal/storage/filesystem/interfaces"
	"os"
	"strings"
)

// GetDriver returns a driver for the local file system
func GetDriver() fileInterfaces.System {
	return &localStorageDriver{}
}

type localStorageDriver struct {
	dataPath   string
	filePrefix string
}

// Config is the required configuration for the driver
type Config struct {
	// DataPath is the top directory where files are stored
	DataPath string
	// FilePrefix is an optional setting, if files are to be stored with the prefix
	FilePrefix string
}

// MoveToFilesystem moves a file from the local filesystem to data path
func (d *localStorageDriver) MoveToFilesystem(sourceFile *os.File, metaData models.File) error {
	err := sourceFile.Close()
	if err != nil {
		return err
	}
	if metaData.SHA1 == "" {
		return errors.New("empty metadata passed")
	}
	return os.Rename(sourceFile.Name(), d.getPath()+d.filePrefix+metaData.SHA1)
}

// Init sets the driver configurations and returns true if successful
// Requires a Config struct as input
func (d *localStorageDriver) Init(input any) bool {
	config, ok := input.(Config)
	if !ok {
		panic("runtime exception: input for local filesystem is not a config object")
	}
	if config.DataPath == "" {
		panic("empty path has been passed")
	}
	if !strings.HasSuffix(config.DataPath, string(os.PathSeparator)) {
		config.DataPath = config.DataPath + string(os.PathSeparator)
	}
	d.dataPath = config.DataPath
	d.filePrefix = config.FilePrefix
	return true
}

// IsAvailable returns true if the data path is writable
func (d *localStorageDriver) IsAvailable() bool {
	return true
}

// GetFile returns a File struct for the corresponding filename
func (d *localStorageDriver) GetFile(filename string) fileInterfaces.File {
	return &localFile{Directory: d.getPath(), Filename: d.filePrefix + filename}
}

// FileExists returns true if the system contains a file with the given relative filepath
func (d *localStorageDriver) FileExists(filename string) (bool, error) {
	file := localFile{
		Directory: d.getPath(),
		Filename:  d.filePrefix + filename,
	}
	return file.Exists(), nil
}

// GetSystemName returns the name of the driver
func (d *localStorageDriver) GetSystemName() string {
	return fileInterfaces.DriverLocal
}

func (d *localStorageDriver) getPath() string {
	if d.dataPath == "" {
		panic("no path has been set!")
	}
	return d.dataPath
}

type localFile struct {
	Directory string
	Filename  string
}

// Exists returns true if the file exists
func (f *localFile) Exists() bool {
	return helper.FileExists(f.Directory + f.Filename)
}

// GetName returns the name of the file
func (f *localFile) GetName() string {
	return f.Filename
}
