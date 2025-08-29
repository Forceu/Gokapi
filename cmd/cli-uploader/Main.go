package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliapi"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliconfig"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliconstants"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliflags"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/schollz/progressbar/v3"
	"io"
	"os"
	"path/filepath"
)

func main() {
	mode := cliflags.Parse()
	switch mode {
	case cliflags.ModeLogin:
		doLogin()
	case cliflags.ModeLogout:
		doLogout()
	case cliflags.ModeUpload:
		processUpload(false)
	case cliflags.ModeArchive:
		processUpload(true)
	case cliflags.ModeInvalid:
		os.Exit(3)
	}
}

func doLogin() {
	checkDockerFolders()
	cliconfig.CreateLogin()
}

func processUpload(isArchive bool) {
	cliconfig.Load()
	uploadParam := cliflags.GetUploadParameters(isArchive)

	if isArchive {
		zipPath, err := zipFolder(uploadParam.Directory, uploadParam.TmpFolder, !uploadParam.JsonOutput)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		uploadParam.File = zipPath
		defer deleteTempFolder(zipPath) // ensure cleanup
	}

	// Perform the upload
	result, err := cliapi.UploadFile(uploadParam)
	if err != nil {
		fmt.Println()
		if errors.Is(cliapi.ErrUnauthorised, err) {
			fmt.Println("ERROR: Unauthorised API key. Please re-run login.")
		} else {
			fmt.Println("ERROR: Could not upload file")
			fmt.Println(err)
		}
		os.Exit(1)
	}

	// Output result
	if uploadParam.JsonOutput {
		jsonStr, _ := json.Marshal(result)
		fmt.Println(string(jsonStr))
		return
	}
	fmt.Println("Upload successful")
	fmt.Println("File Name: " + result.Name)
	fmt.Println("File ID: " + result.Id)
	fmt.Println("File Download URL: " + result.UrlDownload)
}

func doLogout() {
	err := cliconfig.Delete()
	if err != nil {
		fmt.Println("ERROR: Could not delete configuration file")
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Println("Logged out. To login again, run: gokapi-cli login")
}

func checkDockerFolders() {
	if !environment.IsDockerInstance() {
		return
	}
	_, isDefault := cliflags.GetConfigLocation()
	if !isDefault {
		return
	}
	if !helper.FolderExists(cliconstants.DockerFolderConfig) {
		fmt.Println("Warning: Docker folder does not exist, configuration will be lost when creating a new container")
		helper.CreateDir(cliconstants.DockerFolderConfig)
	}
	if !helper.FolderExists(cliconstants.DockerFolderUpload) {
		helper.CreateDir(cliconstants.DockerFolderUpload)
	}
}

func deleteTempFolder(path string) {
	folder := filepath.Dir(path)
	_ = os.RemoveAll(folder)
}

// zipFolder compresses the contents of srcDir into a zip file at destZip
func zipFolder(srcDir, tmpFolder string, showOutput bool) (string, error) {
	var progressBar *progressbar.ProgressBar
	if showOutput {
		progressBar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription("Compressing files..."),
			progressbar.OptionClearOnFinish(),
		)
		defer progressBar.Finish()
	}
	folder, err := os.MkdirTemp(tmpFolder, "gokapi-cli-")
	if err != nil {
		return "", err
	}
	srcDir, err = filepath.Abs(srcDir)
	if err != nil {
		return "", err
	}
	srcDir = filepath.Clean(srcDir)

	zipPath := filepath.Join(folder, filepath.Base(srcDir)+".zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through every file and folder in the source directory
	return zipPath, filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute the relative path (so zip doesn't store absolute paths)
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// For folders, just add an entry with a trailing slash
		if info.IsDir() {
			_, err := zipWriter.Create(relPath + "/")
			return err
		}

		// For files, create a file entry
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		zipEntry, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		progressBar.Describe("Compressing: " + filepath.Base(relPath))
		_, err = io.Copy(zipEntry, file)
		return err
	})
}
