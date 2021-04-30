package models

type ApiKey struct {
	Id           string `json:"Id"`
	FriendlyName string `json:"FriendlyName"`
	LastUsed     int64  `json:"LastUsed"`
}

type UploadItem struct {
	Id                 string `json:"Id"`
	Name               string `json:"Name"`
	Filesize           int64  `json:"Filesize"`
	Expiry             int64  `json:"Expiry"`
	DownloadsRemaining int    `json:"DownloadsRemaining"`
	Url                string `json:"Url"`
}
