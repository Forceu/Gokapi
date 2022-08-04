package chunking

import (
	"errors"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/helper"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ChunkInfo contains info about the current chunk
type ChunkInfo struct {
	TotalFilesizeBytes int64
	Offset             int64
	UUID               string
}

// FileHeader contains info about the uploaded file
type FileHeader struct {
	Filename    string
	ContentType string
	Size        int64
}

// ParseChunkInfo parses the posted form data and returns it as a ChunkInfo object
func ParseChunkInfo(r *http.Request, isApiCall bool) (ChunkInfo, error) {
	info := ChunkInfo{}
	err := r.ParseForm()
	if err != nil {
		return ChunkInfo{}, err
	}

	formTotalSize := "dztotalfilesize"
	formOffset := "dzchunkbyteoffset"
	formUuid := "dzuuid"

	if isApiCall {
		formTotalSize = "filesize"
		formOffset = "offset"
		formUuid = "uuid"
	}

	buf := r.Form.Get(formTotalSize)
	info.TotalFilesizeBytes, err = strconv.ParseInt(buf, 10, 64)
	if err != nil {
		return ChunkInfo{}, err
	}
	if info.TotalFilesizeBytes < 0 {
		return ChunkInfo{}, errors.New("value cannot be negative")
	}

	buf = r.Form.Get(formOffset)
	info.Offset, err = strconv.ParseInt(buf, 10, 64)
	if err != nil {
		return ChunkInfo{}, err
	}
	if info.Offset < 0 {
		return ChunkInfo{}, errors.New("value cannot be negative")
	}

	info.UUID = r.Form.Get(formUuid)
	if len(info.UUID) < 10 {
		return ChunkInfo{}, errors.New("invalid uuid submitted, needs to be at least 10 characters long")
	}
	info.UUID = sanitseUuid(info.UUID)
	return info, nil
}

func sanitseUuid(input string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9-]")
	helper.Check(err)
	return reg.ReplaceAllString(input, "_")
}

// ParseFileHeader parses the formdata and returns a FileHeader
func ParseFileHeader(r *http.Request) (FileHeader, error) {
	err := r.ParseForm()
	if err != nil {
		return FileHeader{}, err
	}
	name := r.Form.Get("filename")
	if name == "" {
		return FileHeader{}, errors.New("empty filename provided")
	}
	contentType := parseContentType(r)
	size := r.Form.Get("filesize")
	if size == "" {
		return FileHeader{}, errors.New("empty size provided")
	}
	sizeInt, err := strconv.ParseInt(size, 10, 64)
	if sizeInt < 0 {
		return FileHeader{}, errors.New("value cannot be negative")
	}
	if err != nil {
		return FileHeader{}, err
	}
	return FileHeader{
		Filename:    name,
		Size:        sizeInt,
		ContentType: contentType,
	}, nil
}

func parseContentType(r *http.Request) string {
	contentType := r.Form.Get("filecontenttype")
	if contentType != "" {
		return contentType
	}
	fileExt := strings.ToLower(filepath.Ext(r.Form.Get("filename")))
	switch fileExt {
	case ".jpeg":
		fallthrough
	case ".jpg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	case ".bmp":
		contentType = "image/bmp"
	case ".svg":
		contentType = "image/svg+xml"
	case ".tiff":
		fallthrough
	case ".tif":
		contentType = "image/tiff"
	case ".ico":
		contentType = "image/vnd.microsoft.icon"
	default:
		contentType = "application/octet-stream"
	}
	return contentType
}

// ParseMultipartHeader converts a multipart.FileHeader to the internal FileHeader
func ParseMultipartHeader(header *multipart.FileHeader) (FileHeader, error) {
	if header.Filename == "" {
		return FileHeader{}, errors.New("empty filename provided")
	}
	if header.Header.Get("Content-Type") == "" {
		return FileHeader{}, errors.New("empty content-type provided")
	}
	return FileHeader{
		Filename:    header.Filename,
		Size:        header.Size,
		ContentType: header.Header.Get("Content-Type"),
	}, nil
}

func getChunkFilePath(id string) string {
	return configuration.Get().DataDir + "/chunk-" + id
}

// GetFileByChunkId returns a handle to the chunk file
func GetFileByChunkId(id string) (*os.File, error) {
	file, err := os.OpenFile(getChunkFilePath(id), os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// NewChunk allocates the space for the new file and writes the chunk
func NewChunk(chunkContent io.Reader, fileHeader *multipart.FileHeader, info ChunkInfo) error {
	err := allocateFile(info)
	if err != nil {
		return err
	}
	return writeChunk(chunkContent, fileHeader, info)
}

func allocateFile(info ChunkInfo) error {
	if helper.FileExists(getChunkFilePath(info.UUID)) {
		return nil
	}
	file, err := os.OpenFile(getChunkFilePath(info.UUID), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	err = file.Truncate(info.TotalFilesizeBytes)
	return err
}

func writeChunk(chunkContent io.Reader, fileHeader *multipart.FileHeader, info ChunkInfo) error {
	if info.Offset+fileHeader.Size > info.TotalFilesizeBytes {
		return errors.New("chunksize will be bigger than total filesize from this offset")
	}
	file, err := GetFileByChunkId(info.UUID)
	if err != nil {
		return err
	}
	newOffset, err := file.Seek(info.Offset, io.SeekStart)
	if err != nil {
		return err
	}
	if newOffset != info.Offset {
		return errors.New("seek returned invalid offset")
	}
	_, err = io.Copy(file, chunkContent)
	return err
}
