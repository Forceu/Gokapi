package redis

const (
	prefixStatisticsTraffic = "stats:traffic"
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
