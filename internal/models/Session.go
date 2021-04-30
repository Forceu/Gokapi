package models

// Session contains cookie parameter
type Session struct {
	RenewAt    int64
	ValidUntil int64
}
