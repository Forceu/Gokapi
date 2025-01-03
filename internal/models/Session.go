package models

// Session contains cookie parameter
type Session struct {
	RenewAt    int64 `redis:"renew_at"`
	ValidUntil int64 `redis:"valid_until"`
	UserId     int   `redis:"user_id"`
}
