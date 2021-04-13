package sessionstructure


// Structure for cookies
type Session struct {
	RenewAt    int64
	ValidUntil int64
}

