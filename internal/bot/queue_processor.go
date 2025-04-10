package bot

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"time"

	coordinatorpb "github.com/aquare11e/media-downloader-bot/common/protogen/coordinator"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type QueueProcessor struct {
	bot         *Bot
	stopChan    chan struct{}
	isRunning   bool
	updateDelay time.Duration
}

func NewQueueProcessor(bot *Bot) *QueueProcessor {
	return &QueueProcessor{
		bot:         bot,
		stopChan:    make(chan struct{}),
		updateDelay: 30 * time.Second, // Update status every 5 seconds
	}
}

func (qp *QueueProcessor) Start() {
	if qp.isRunning {
		return
	}

	qp.isRunning = true
	go qp.processQueue()
}

func (qp *QueueProcessor) Stop() {
	if !qp.isRunning {
		return
	}

	close(qp.stopChan)
	qp.isRunning = false
}

func (qp *QueueProcessor) processQueue() {
	ctx := context.Background()
	ticker := time.NewTicker(qp.updateDelay)
	defer ticker.Stop()

	for {
		select {
		case <-qp.stopChan:
			return
		case <-ticker.C:
			qp.processMessages(ctx)
		}
	}
}

func (qp *QueueProcessor) processMessages(ctx context.Context) {
	// Process messages one by one until the queue is empty
	for {
		// Get one message from the queue
		message, err := qp.bot.redisClient.LPop(ctx, KeyDownloadProgressQueue).Result()

		if err != nil {
			if err == redis.Nil {
				// Queue is empty
				return
			}
			log.Printf("Failed to get message from queue: %v", err)
			return
		}

		encodedMessage := base64.StdEncoding.EncodeToString([]byte(message))
		log.Printf("Message from queue: %s", encodedMessage)

		var downloadResp coordinatorpb.DownloadResponse
		if err := proto.Unmarshal([]byte(message), &downloadResp); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		// Convert to DownloadStatus
		status := &DownloadStatus{
			Name:     downloadResp.Name,
			Status:   downloadResp.Status,
			Message:  downloadResp.Message,
			ETA:      time.Duration(downloadResp.Eta) * time.Second,
			Progress: downloadResp.Progress,
		}

		log.Printf("Download status: %s", status.ToLogString())

		// Update status in Redis
		key := fmt.Sprintf(KeyTorrentInProgress, downloadResp.RequestId)
		err = qp.bot.redisClient.HSet(ctx, key, status.ToRedisMap()).Err()
		if err != nil {
			log.Printf("Failed to update status in Redis: %v", err)
			continue
		}

		// Add to set of active downloads if not already present
		err = qp.bot.redisClient.SAdd(ctx, KeyTorrentInProgressKeys, downloadResp.RequestId).Err()
		if err != nil {
			log.Printf("Failed to add to active downloads set: %v", err)
			continue
		}

		// If download is completed or failed, remove from active downloads
		if status.Status == coordinatorpb.DownloadStatus_DOWNLOAD_STATUS_SUCCESS || status.Status == coordinatorpb.DownloadStatus_DOWNLOAD_STATUS_ERROR {
			err := qp.bot.redisClient.SRem(ctx, KeyTorrentInProgressKeys, downloadResp.RequestId).Err()
			if err != nil {
				log.Printf("Failed to remove from active downloads set: %v", err)
				continue
			}

			err = qp.bot.redisClient.Del(ctx, fmt.Sprintf(KeyTorrentInProgress, downloadResp.RequestId)).Err()
			if err != nil {
				log.Printf("Failed to remove from active downloads set: %v", err)
			}

			ownerResp := qp.bot.redisClient.GetDel(ctx, fmt.Sprintf(KeyTorrentDownloadOwner, downloadResp.RequestId))
			if ownerResp.Err() != nil {
				log.Printf("Failed to get download owner: %v", ownerResp.Err())
				continue
			}

			ownerID := ownerResp.Val()
			if ownerID == "" {
				log.Printf("Download owner not found for request ID: %s", downloadResp.RequestId)
				continue
			}

			ownerIDInt, err := strconv.ParseInt(ownerID, 10, 64)
			if err != nil {
				log.Printf("Failed to convert ownerID to int64: %v", err)
				continue
			}

			msg := tgbotapi.NewMessage(ownerIDInt, "🎉 Your download is complete!\n📁 File: "+status.Name+"\n📝 Message: "+status.Message+"\n\nIf you encountered any issues, feel free to reach out for help!")
			qp.bot.api.Send(msg)
		}
	}
}
