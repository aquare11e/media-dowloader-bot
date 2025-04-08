package coordinator

import (
	"time"

	common "github.com/aquare11e/media-dowloader-bot/common/protogen/common"
)

const (
	// KeyTorrentFormat is the format for Redis keys storing torrent information
	KeyTorrentAll    = "coordinator:torrent:*"
	KeyTorrentFormat = "coordinator:torrent:%d"
	// StaleThreshold is the time after which a record is considered stale
	StaleThreshold = 10 * time.Minute
	// CheckInterval is how often the recovery service checks for stale records
	CheckInterval = 1 * time.Minute
	// EtaErrorSeconds is the number of seconds to add to ETA (because download is not always accurate)
	EtaErrorSeconds = 10
)

// TorrentRecord represents a torrent download record stored in Redis
type TorrentRecord struct {
	RequestID  string
	Category   common.RequestType
	LastUpdate time.Time
}

// ToRedisMap converts TorrentRecord to a map of field-value pairs for Redis
func (r *TorrentRecord) ToRedisMap() map[string]any {
	return map[string]any{
		"request_id":  r.RequestID,
		"category":    int32(r.Category),
		"last_update": r.LastUpdate.Format(time.RFC3339),
	}
}
