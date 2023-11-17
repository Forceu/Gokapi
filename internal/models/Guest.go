package models

// GuestToken contains data of a single guest token
type GuestToken struct {
	Id             string `json:"Id"`
	ExpireAt       int64  `json:"ExpireAt"`
	ExpireAtString string `json:"ExpireAtString"`
	UnlimitedTime  bool   `json:"UnlimitedTime"`
	TimesUsed      int64  `json:"TimesUsed"`
	LastUsed       int64  `json:"LastUsed"`
	LastUsedString string `json:"LastUsedString"`
}
