package models

type Presign struct {
	Id       string
	FileIds  []string
	Expiry   int64
	Filename string
}
