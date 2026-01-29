package sqlite

import (
	"database/sql"
	"errors"

	"github.com/forceu/gokapi/internal/helper"
)

const statIdTraffic = "1"
const statIdTrafficSince = "2"

// GetStatTraffic returns the total traffic from statistics
func (p DatabaseProvider) GetStatTraffic() uint64 {
	var result uint64
	row := p.sqliteDb.QueryRow("SELECT value FROM Statistics WHERE type = ?", statIdTraffic)
	err := row.Scan(&result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0
		}
		helper.Check(err)
		return 0
	}
	return result
}

// SaveStatTraffic stores the total traffic
func (p DatabaseProvider) SaveStatTraffic(totalTraffic uint64) {
	_, err := p.sqliteDb.Exec(`INSERT INTO Statistics (type, value) VALUES (?, ?)
					ON CONFLICT(type) DO UPDATE SET value = ?`, statIdTraffic, totalTraffic, totalTraffic)
	helper.Check(err)
}

// SaveTrafficSince stores the beginning of traffic counting
func (p DatabaseProvider) SaveTrafficSince(since int64) {
	_, err := p.sqliteDb.Exec(`INSERT INTO Statistics (type, value) VALUES (?, ?)
					ON CONFLICT(type) DO UPDATE SET value = ?`, statIdTrafficSince, since, since)
	helper.Check(err)
}

// GetTrafficSince gets the beginning of traffic counting
func (p DatabaseProvider) GetTrafficSince() (int64, bool) {
	var result int64
	row := p.sqliteDb.QueryRow("SELECT value FROM Statistics WHERE type = ?", statIdTrafficSince)
	err := row.Scan(&result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false
		}
		helper.Check(err)
		return 0, false
	}
	return result, true
}
