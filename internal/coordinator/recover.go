package coordinator

import (
	"context"
	"fmt"
	"log"
	"time"

	common "github.com/aquare11e/media-downloader-bot/common/protogen/common"
	"github.com/aquare11e/media-downloader-bot/common/protogen/plex"
	"github.com/aquare11e/media-downloader-bot/common/protogen/transmission"
)

const (
	staleThreshold = 10 * time.Minute
	checkInterval  = 1 * time.Minute
)

func (s *Service) StartRecoveryService(ctx context.Context) {
	log.Printf("Starting recovery service")

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Recovery service stopped")
			return
		case <-ticker.C:
			s.checkStaleRecords(ctx)
		}
	}
}

func (s *Service) checkStaleRecords(ctx context.Context) {
	// Get all torrent keys
	keys, err := s.redisClient.Keys(ctx, KeyTorrentAll).Result()
	if err != nil {
		log.Printf("Error getting torrent keys: %v", err)
		return
	}

	for _, key := range keys {
		// Get torrent record
		record, err := s.getTorrentRecord(ctx, key)
		if err != nil {
			log.Printf("Error getting torrent record for key %s: %v", key, err)
			continue
		}

		// Check if record is stale
		if time.Since(record.LastUpdate) > staleThreshold {
			log.Printf("Torrent record is stale (key: %s, record: %v)", key, record)
			s.handleStaleRecord(ctx, key, record)
		}
	}
}

func (s *Service) getTorrentRecord(ctx context.Context, key string) (*TorrentRecord, error) {
	values, err := s.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	lastUpdate, err := time.Parse(time.RFC3339, values["last_update"])
	if err != nil {
		return nil, err
	}

	// Parse category as int32 and convert to RequestType
	var categoryInt int32
	_, err = fmt.Sscanf(values["category"], "%d", &categoryInt)
	if err != nil {
		return nil, err
	}

	return &TorrentRecord{
		RequestID:  values["request_id"],
		Category:   common.RequestType(categoryInt),
		LastUpdate: lastUpdate,
	}, nil
}

func (s *Service) handleStaleRecord(ctx context.Context, key string, record *TorrentRecord) {
	// Extract torrent ID from key
	var torrentID int64
	_, err := fmt.Sscanf(key, KeyTorrentFormat, &torrentID)
	if err != nil {
		log.Printf("Error parsing torrent ID from key %s: %v", key, err)
		return
	}

	// Get torrent status from Transmission
	statusReq := &transmission.GetTorrentStatusRequest{
		TorrentId: torrentID,
	}
	statusResp, err := s.transmissionClient.GetTorrentStatus(ctx, statusReq)
	if err != nil {
		log.Printf("Error getting torrent status for ID %d: %v", torrentID, err)
		return
	}

	switch statusResp.Status {
	case transmission.TorrentStatus_STATUS_DONE:
		// Update Plex library
		plexReq := &plex.UpdateCategoryRequest{
			RequestId: record.RequestID,
			Type:      record.Category,
		}
		_, err := s.plexClient.UpdateCategory(ctx, plexReq)
		if err != nil {
			log.Printf("Error updating Plex library for torrent %d: %v", torrentID, err)
		}
		// Clean up Redis record
		s.redisClient.Del(ctx, key)

	case transmission.TorrentStatus_STATUS_ERROR:
		// Just delete the record
		s.redisClient.Del(ctx, key)

	case transmission.TorrentStatus_STATUS_IN_PROGRESS:
		// Start a goroutine to monitor this torrent
		go s.monitorTorrent(ctx, key, torrentID, record)
	}
}

func (s *Service) monitorTorrent(ctx context.Context, key string, torrentID int64, record *TorrentRecord) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get torrent status
			statusReq := &transmission.GetTorrentStatusRequest{
				TorrentId: torrentID,
			}
			statusResp, err := s.transmissionClient.GetTorrentStatus(ctx, statusReq)
			if err != nil {
				log.Printf("Error getting torrent status for ID %d: %v", torrentID, err)
				return
			}

			// Update last update time
			record.LastUpdate = time.Now()
			err = s.redisClient.HSet(ctx, key, record.ToRedisMap()).Err()
			if err != nil {
				log.Printf("Error updating last update time for torrent %d: %v", torrentID, err)
				return
			}

			switch statusResp.Status {
			case transmission.TorrentStatus_STATUS_DONE:
				// Update Plex library
				plexReq := &plex.UpdateCategoryRequest{
					RequestId: record.RequestID,
					Type:      record.Category,
				}
				_, err := s.plexClient.UpdateCategory(ctx, plexReq)
				if err != nil {
					log.Printf("Error updating Plex library for torrent %d: %v", torrentID, err)
				}
				// Clean up Redis record
				s.redisClient.Del(ctx, key)
				return

			case transmission.TorrentStatus_STATUS_ERROR:
				// Just delete the record
				s.redisClient.Del(ctx, key)
				return
			}

			// Wait based on ETA
			waitTime := time.Duration(statusResp.Eta)*time.Second + EtaErrorSeconds*time.Second
			if waitTime > 0 && waitTime < time.Minute {
				time.Sleep(waitTime)
			} else {
				time.Sleep(time.Minute)
			}
		}
	}
}
