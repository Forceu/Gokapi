package redis

const (
	prefixStatisticsTraffic      = "stats:traffic"
	prefixStatisticsTrafficSince = "stats:traffic_since"
)

// GetStatTraffic returns the total traffic from statistics
func (p DatabaseProvider) GetStatTraffic() uint64 {
	result, _ := p.getKeyUInt64(prefixStatisticsTraffic)
	return result
}

// SaveStatTraffic stores the total traffic
func (p DatabaseProvider) SaveStatTraffic(totalTraffic uint64) {
	p.setKey(prefixStatisticsTraffic, totalTraffic)
}

// SaveTrafficSince stores the beginning of traffic counting
func (p DatabaseProvider) SaveTrafficSince(since int64) {
	p.setKey(prefixStatisticsTrafficSince, since)
}

// GetTrafficSince gets the beginning of traffic counting
func (p DatabaseProvider) GetTrafficSince() (int64, bool) {
	return p.getKeyInt64(prefixStatisticsTrafficSince)
}
