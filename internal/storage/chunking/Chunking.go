package chunking

import (
	"errors"
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/helper"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
)

type ChunkInfo struct {
	Index              int
	TotalFilesizeBytes int64
	Size               int64
	TotalChunkCount    int
	Offset             int64
	UUID               string
}
type FileHeader struct {
	Filename    string
	ContentType string
	Size        int64
}

func ParseChunkInfo(r *http.Request) (ChunkInfo, error) {
	info := ChunkInfo{}
	err := r.ParseForm()
	if err != nil {
		return ChunkInfo{}, err
	}
	buf := r.Form.Get("dzchunkindex")
	info.Index, err = strconv.Atoi(buf)
	if err != nil {
		return ChunkInfo{}, err
	}
	buf = r.Form.Get("dztotalfilesize")
	info.TotalFilesizeBytes, err = strconv.ParseInt(buf, 10, 64)
	if err != nil {
		return ChunkInfo{}, err
	}
	buf = r.Form.Get("dzchunksize")
	info.Size, err = strconv.ParseInt(buf, 10, 64)
	if err != nil {
		return ChunkInfo{}, err
	}
	buf = r.Form.Get("dztotalchunkcount")
	info.TotalChunkCount, err = strconv.Atoi(buf)
	if err != nil {
		return ChunkInfo{}, err
	}
	buf = r.Form.Get("dzchunkbyteoffset")
	info.Offset, err = strconv.ParseInt(buf, 10, 64)
	if err != nil {
		return ChunkInfo{}, err
	}
	info.UUID = r.Form.Get("dzuuid")
	if len(info.UUID) < 10 {
		return ChunkInfo{}, errors.New("invalid uuid submitted")
	}
	return info, nil
}

func ParseFileHeader(r *http.Request) (FileHeader, error) {
	err := r.ParseForm()
	if err != nil {
		return FileHeader{}, err
	}
	name := r.Form.Get("filename")
	if name == "" {
		return FileHeader{}, errors.New("empty filename provided")
	}
	contentType := r.Form.Get("filecontenttype")
	if contentType == "" {
		return FileHeader{}, errors.New("empty content-type provided")
	}
	size := r.Form.Get("filesize")
	if size == "" {
		return FileHeader{}, errors.New("empty size provided")
	}
	sizeInt, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		return FileHeader{}, err
	}
	return FileHeader{
		Filename:    name,
		Size:        sizeInt,
		ContentType: contentType,
	}, nil
}

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

func GetFileByChunkId(id string) (*os.File, error) {
	file, err := os.OpenFile(getChunkFilePath(id), os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func NewChunk(chunkContent io.Reader, fileHeader *multipart.FileHeader, info ChunkInfo) error {
	if info.Index == 0 {
		err := allocateFile(info)
		if err != nil {
			return err
		}
	}
	return writeChunk(chunkContent, fileHeader, info)
}

func DeleteChunkFile(id string) error {
	return os.Remove(getChunkFilePath(id))
}

func allocateFile(info ChunkInfo) error {
	file, err := os.OpenFile(getChunkFilePath(info.UUID), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	err = file.Truncate(info.TotalFilesizeBytes)
	return err
}

func writeChunk(chunkContent io.Reader, fileHeader *multipart.FileHeader, info ChunkInfo) error {
	if info.Offset+fileHeader.Size > info.TotalFilesizeBytes {
		return errors.New("invalid offset specified")
	}
	if fileHeader.Size > info.Size {
		return errors.New("chunk size mismatch")
	}
	if !helper.FileExists(getChunkFilePath(info.UUID)) {
		return errors.New("file has not been allocated yet, first chunk has probably not been sent")
	}
	file, err := os.OpenFile(getChunkFilePath(info.UUID), os.O_RDWR|os.O_CREATE, 0600)
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
