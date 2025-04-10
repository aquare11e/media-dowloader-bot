package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	coordinatorpb "github.com/aquare11e/media-downloader-bot/common/protogen/coordinator"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	progressBarLength = 10
	progressChars     = "█▉▊▋▌▍▎▏"
)

type StatusChecker struct {
	bot *Bot
}

func NewStatusChecker(bot *Bot) *StatusChecker {
	return &StatusChecker{
		bot: bot,
	}
}

func (sc *StatusChecker) checkStatus(chatID int64) {
	ctx := context.Background()

	// Get all progress requestIds from Redis
	requestIds, err := sc.bot.redisClient.SMembers(ctx, KeyTorrentInProgressKeys).Result()
	if err != nil {
		log.Printf("Failed to get progress updates: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Oops! I couldn't get the download status. Please try again later!")
		sc.bot.api.Send(msg)
		return
	}

	if len(requestIds) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📭 No active downloads found. Start a new download with /download command!")
		sc.bot.api.Send(msg)
		return
	}

	if len(requestIds) > 5 {
		requestIds = requestIds[:5]
	}

	// Create inline keyboard for each download
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, requestId := range requestIds {
		res, err := sc.bot.redisClient.HGetAll(ctx, fmt.Sprintf(KeyTorrentInProgress, requestId)).Result()
		if err != nil {
			log.Printf("Failed to get progress updates: %v", err)
			continue
		}

		status := &DownloadStatus{}
		err = status.FromRedisMap(res)
		if err != nil {
			log.Printf("Failed to parse status: %v", err)
			continue
		}

		statusText := getStatusText(status.Status)
		progressBar := createProgressBar(status.Progress)
		etaText := ""
		if status.ETA > 0 {
			etaText = fmt.Sprintf(" (⏱️ ETA: %s)", formatDuration(int32(status.ETA)))
		}

		buttonText := fmt.Sprintf("📁 %s\n%s %s%s", status.Name, progressBar, statusText, etaText)
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("status_%s", requestId))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	// Add refresh button
	refreshButton := tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh Status", "refresh_status")
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(refreshButton))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, "📊 Active Downloads:")
	msg.ReplyMarkup = keyboard
	sc.bot.api.Send(msg)
}

func createProgressBar(progress float64) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	relativeProgress := progress / 100

	filled := int(relativeProgress * float64(progressBarLength))
	empty := progressBarLength - filled

	// Calculate partial character
	partial := int((relativeProgress*float64(progressBarLength) - float64(filled)) * float64(len(progressChars)))
	partialChar := ""
	if partial > 0 && filled < progressBarLength {
		partialChar = string(progressChars[partial-1])
		empty--
	}

	return fmt.Sprintf("[%s%s%s] %.1f%%",
		strings.Repeat("█", filled),
		partialChar,
		strings.Repeat("░", empty),
		progress)
}

func getStatusText(status coordinatorpb.DownloadStatus) string {
	switch status {
	case coordinatorpb.DownloadStatus_DOWNLOAD_STATUS_IN_PROGRESS:
		return "⏳"
	case coordinatorpb.DownloadStatus_DOWNLOAD_STATUS_SUCCESS:
		return "✅"
	case coordinatorpb.DownloadStatus_DOWNLOAD_STATUS_ERROR:
		return "❌"
	default:
		return "❓"
	}
}

func formatDuration(seconds int32) string {
	duration := time.Duration(seconds) * time.Second
	if duration.Hours() >= 24 {
		return fmt.Sprintf("%.1f days", duration.Hours()/24)
	}
	if duration.Hours() >= 1 {
		return fmt.Sprintf("%.1f hours", duration.Hours())
	}
	if duration.Minutes() >= 1 {
		return fmt.Sprintf("%.1f minutes", duration.Minutes())
	}
	return fmt.Sprintf("%d seconds", seconds)
}

func (sc *StatusChecker) HandleCallback(callback *tgbotapi.CallbackQuery) {
	if callback.Data == "refresh_status" {
		// Edit the existing message
		sc.editStatusMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		return
	}

	if strings.HasPrefix(callback.Data, "status_") {
		requestID := strings.TrimPrefix(callback.Data, "status_")
		sc.editDetailedStatus(callback.Message.Chat.ID, callback.Message.MessageID, requestID)
		return
	}

	// Answer the callback to remove the loading state
	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	sc.bot.api.Send(callbackConfig)
}

func (sc *StatusChecker) editStatusMessage(chatID int64, messageID int) {
	ctx := context.Background()

	// Get all progress requestIds from Redis
	requestIds, err := sc.bot.redisClient.SMembers(ctx, KeyTorrentInProgressKeys).Result()
	if err != nil {
		log.Printf("Failed to get progress updates: %v", err)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ Oops! I couldn't get the download status. Please try again later!")
		sc.bot.api.Send(editMsg)
		return
	}

	if len(requestIds) == 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "📭 No active downloads found. Start a new download with /download command!")
		sc.bot.api.Send(editMsg)
		return
	}

	if len(requestIds) > 5 {
		requestIds = requestIds[:5]
	}

	// Create inline keyboard for each download
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, requestId := range requestIds {
		res, err := sc.bot.redisClient.HGetAll(ctx, fmt.Sprintf(KeyTorrentInProgress, requestId)).Result()
		if err != nil {
			log.Printf("Failed to get progress updates: %v", err)
			continue
		}

		status := &DownloadStatus{}
		err = status.FromRedisMap(res)
		if err != nil {
			log.Printf("Failed to parse status: %v", err)
			continue
		}

		statusText := getStatusText(status.Status)
		progressBar := createProgressBar(status.Progress)
		etaText := ""
		if status.ETA > 0 {
			etaText = fmt.Sprintf(" (⏱️ ETA: %s)", formatDuration(int32(status.ETA)))
		}

		buttonText := fmt.Sprintf("📁 %s\n%s %s%s", status.Name, progressBar, statusText, etaText)
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("status_%s", requestId))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	// Add refresh button
	refreshButton := tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh Status", "refresh_status")
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(refreshButton))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "📊 Active Downloads:")
	editMsg.ReplyMarkup = &keyboard
	sc.bot.api.Send(editMsg)
}

func (sc *StatusChecker) editDetailedStatus(chatID int64, messageID int, requestID string) {
	ctx := context.Background()

	// Get all progress downloadStatusMap from Redis
	downloadStatusMap, err := sc.bot.redisClient.HGetAll(ctx, fmt.Sprintf(KeyTorrentInProgress, requestID)).Result()
	if err != nil {
		log.Printf("Failed to get progress updates: %v", err)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ Oops! I couldn't get the download status. Please try again later!")
		sc.bot.api.Send(editMsg)
		return
	}

	status := &DownloadStatus{}
	err = status.FromRedisMap(downloadStatusMap)
	if err != nil {
		log.Printf("Failed to parse status: %v", err)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ Oops! I couldn't get the download status. Please try again later!")
		sc.bot.api.Send(editMsg)
		return
	}

	statusText := getStatusText(status.Status)
	progressBar := createProgressBar(status.Progress)
	etaText := ""
	if status.ETA > 0 {
		etaText = fmt.Sprintf("\n⏱️ ETA: %s", formatDuration(int32(status.ETA)))
	}

	message := fmt.Sprintf("📥 Download Details:\n\n📁 Name: %s\n%s \n📊 Progress: %s%s\n💬 Message: %s\n",
		status.Name,
		statusText,
		progressBar,
		etaText,
		status.Message,
	)

	// Create back button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back to List", "refresh_status"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, message)
	editMsg.ReplyMarkup = &keyboard
	sc.bot.api.Send(editMsg)
}
