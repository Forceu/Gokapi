package models

// ApiKey contains data of a single api key
type ApiKey struct {
	Id           string `json:"Id"`
	FriendlyName string `json:"FriendlyName"`
	LastUsed     int64  `json:"LastUsed"`
}

// UploadItem is the result for the "list uploads" api call
type UploadItem struct {
	Id                 string `json:"Id"`
	Name               string `json:"Name"`
	Filesize           int64  `json:"Filesize"`
	Expiry             int64  `json:"Expiry"`
	DownloadsRemaining int    `json:"DownloadsRemaining"`
	Url                string `json:"Url"`
}
