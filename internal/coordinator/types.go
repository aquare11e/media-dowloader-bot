package coordinator

import (
	common "github.com/aquare11e/media-downloader-bot/common/protogen/common"
)

// TorrentRecord represents a torrent download record stored in Redis
type TorrentRecord struct {
	TorrentID int64
	Category  common.RequestType
}

// ToRedisMap converts TorrentRecord to a map of field-value pairs for Redis
func (r *TorrentRecord) ToRedisMap() map[string]any {
	return map[string]any{
		"torrent_id": r.TorrentID,
		"category":   int32(r.Category),
	}
}
