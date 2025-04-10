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
	progressBarLength     = 10
	progressBarFullLength = 12
	buttonTextLength      = 35
	ellipsisLength        = 2
)

type statusInlineCmd struct {
	Error       error
	MessageText string
	Rows        [][]tgbotapi.InlineKeyboardButton
}

type StatusChecker struct {
	bot *Bot
}

func NewStatusChecker(bot *Bot) *StatusChecker {
	return &StatusChecker{
		bot: bot,
	}
}

func (sc *StatusChecker) CheckStatus(chatID int64, messageID int) {
	statusInlineCmd := sc.makeStatusInlineMessage()
	if statusInlineCmd.Error != nil {
		editMsg := tgbotapi.NewMessage(chatID, statusInlineCmd.MessageText)
		sc.bot.api.Send(editMsg)
		return
	}

	if len(statusInlineCmd.Rows) == 0 {
		editMsg := tgbotapi.NewMessage(chatID, statusInlineCmd.MessageText)
		sc.bot.api.Send(editMsg)
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(statusInlineCmd.Rows...)
	msg := tgbotapi.NewMessage(chatID, "📊 Active Downloads:")
	msg.ReplyMarkup = keyboard
	msg.ReplyToMessageID = messageID
	sc.bot.api.Send(msg)
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

	if callback.Data == "close_status" {
		// Delete the current message
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		sc.bot.api.Send(deleteMsg)

		deleteCommandMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.ReplyToMessage.MessageID)
		sc.bot.api.Send(deleteCommandMsg)

		return
	}

	// Answer the callback to remove the loading state
	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	sc.bot.api.Send(callbackConfig)
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

	return fmt.Sprintf("[%s%s] %.1f%%",
		strings.Repeat("█", filled),
		strings.Repeat("-", empty),
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

func formatDuration(duration time.Duration) string {
	if duration.Hours() >= 1 {
		return fmt.Sprintf("%.1f hours", duration.Hours())
	}
	if duration.Minutes() >= 1 {
		return fmt.Sprintf("%.1f minutes", duration.Minutes())
	}
	return fmt.Sprintf("%d seconds", int(duration.Seconds()))
}

func formatShortDuration(duration time.Duration) string {
	if duration.Hours() >= 1 {
		return fmt.Sprintf("%.1fh", duration.Hours())
	}
	if duration.Minutes() >= 1 {
		return fmt.Sprintf("%.1fm", duration.Minutes())
	}
	return fmt.Sprintf("%ds", int(duration.Seconds()))
}

func (sc *StatusChecker) editStatusMessage(chatID int64, messageID int) {
	statusInlineCmd := sc.makeStatusInlineMessage()
	if statusInlineCmd.Error != nil {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, statusInlineCmd.MessageText)
		sc.bot.api.Send(editMsg)
		return
	}

	if len(statusInlineCmd.Rows) == 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, statusInlineCmd.MessageText)
		sc.bot.api.Send(editMsg)
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(statusInlineCmd.Rows...)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "📊 Active Downloads:")
	editMsg.ReplyMarkup = &keyboard
	sc.bot.api.Send(editMsg)
}

func (sc *StatusChecker) makeStatusInlineMessage() *statusInlineCmd {
	ctx := context.Background()

	// Get all progress requestIds from Redis
	requestIds, err := sc.bot.redisClient.SMembers(ctx, KeyTorrentInProgressKeys).Result()
	if err != nil {
		log.Printf("Failed to get progress updates: %v", err)
		return &statusInlineCmd{
			Error:       err,
			MessageText: "❌ Oops! I couldn't get the download status. Please try again later!",
		}
	}

	if len(requestIds) == 0 {
		return &statusInlineCmd{
			Error:       nil,
			MessageText: "📭 No active downloads found. Start a new download with /download command!",
			Rows:        [][]tgbotapi.InlineKeyboardButton{},
		}
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

		progressBar := createProgressBar(status.Progress)
		etaText := ""
		if status.ETA > 0 {
			etaText = fmt.Sprintf("(ETA: %s)", formatShortDuration(status.ETA))
		}

		nameTextLength := buttonTextLength - len(progressBar) - len(etaText) - ellipsisLength
		nameText := status.Name
		if nameTextLength > 0 && len(nameText) > nameTextLength {
			nameText = nameText[:nameTextLength] + ".."
		}

		buttonText := fmt.Sprintf("%s %s %s", nameText, progressBar, etaText)
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("status_%s", requestId))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	refreshButton := tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh Status", "refresh_status")
	closeButton := tgbotapi.NewInlineKeyboardButtonData("🗑️ Close", "close_status")
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(refreshButton, closeButton))

	return &statusInlineCmd{
		Error: nil,
		Rows:  rows,
	}
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
		etaText = fmt.Sprintf("\n⏱️ ETA: %s", formatDuration(status.ETA))
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
