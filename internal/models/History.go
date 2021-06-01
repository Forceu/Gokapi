package models


// File is a struct used for saving information about an uploaded file
type DownloadHistory struct {
	Id                 string `json:"Id"`
	FileId             string `json:"FileId"`
	DownloaderIP       string `json:"DownloaderIP"`
	DownloaderUA       string `json:"DownloaderUA"`
	DownloadDate       int64 `json:"DownloadDate"`
}

