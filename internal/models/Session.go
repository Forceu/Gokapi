package models

// Session contains cookie parameter
type Session struct {
	RenewAt    int64 `redis:"renew_at"` // The earliest the session will be rotated without logout
	ValidUntil int64 `redis:"valid_until"`
	Expiry     int64 `redis:"expiry"`  // Even after rotation, the user has to log in again after this date
	UserId     int   `redis:"user_id"` // User ID belonging to the session
}
