package postgres

import (
	"database/sql"
	"errors"

	"github.com/forceu/gokapi/internal/helper"
)

const statIdTraffic = 1
const statIdTrafficSince = 2

// GetStatTraffic returns the total traffic from statistics
func (p DatabaseProvider) GetStatTraffic() uint64 {
	var result uint64
	row := p.sqlDb.QueryRow("SELECT value FROM statistics WHERE type = $1", statIdTraffic)
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
	_, err := p.sqlDb.Exec(`INSERT INTO statistics (type, value) VALUES ($1, $2)
					ON CONFLICT (type) DO UPDATE SET value = $2`, statIdTraffic, totalTraffic)
	helper.Check(err)
}

// SaveTrafficSince stores the beginning of traffic counting
func (p DatabaseProvider) SaveTrafficSince(since int64) {
	_, err := p.sqlDb.Exec(`INSERT INTO statistics (type, value) VALUES ($1, $2)
					ON CONFLICT (type) DO UPDATE SET value = $2`, statIdTrafficSince, since)
	helper.Check(err)
}

// GetTrafficSince gets the beginning of traffic counting
func (p DatabaseProvider) GetTrafficSince() (int64, bool) {
	var result int64
	row := p.sqlDb.QueryRow("SELECT value FROM statistics WHERE type = $1", statIdTrafficSince)
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
