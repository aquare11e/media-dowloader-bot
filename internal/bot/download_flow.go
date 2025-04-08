package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	common "github.com/aquare11e/media-dowloader-bot/common/protogen/common"
	coordinator "github.com/aquare11e/media-dowloader-bot/common/protogen/coordinator"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type downloadState struct {
	step     int
	link     string
	category common.RequestType
}

type DownloadFlow struct {
	bot    *Bot
	States map[int64]*downloadState
}

func NewDownloadFlow(bot *Bot) *DownloadFlow {
	return &DownloadFlow{
		bot:    bot,
		States: make(map[int64]*downloadState),
	}
}

func (df *DownloadFlow) Start(chatID int64) {
	df.States[chatID] = &downloadState{
		step: 1,
	}

	response := tgbotapi.NewMessage(chatID, "Please send me a magnet link or torrent file")
	df.bot.api.Send(response)
}

func (df *DownloadFlow) HandleMessage(msg *tgbotapi.Message) {
	response := tgbotapi.NewMessage(msg.Chat.ID, "")

	if state, exists := df.States[msg.Chat.ID]; exists {
		switch state.step {
		case 1: // Waiting for magnet link or torrent file
			// Check if it's a magnet link
			if strings.HasPrefix(msg.Text, "magnet:?xt=urn:btih:") {
				state.link = msg.Text
				state.step = 2
				df.sendCategoryButtons(msg.Chat.ID)
				return
			}

			// Check if it's a document (torrent file)
			if msg.Document != nil && strings.HasSuffix(msg.Document.FileName, ".torrent") {
				// Get file info
				file, err := df.bot.api.GetFile(tgbotapi.FileConfig{FileID: msg.Document.FileID})
				if err != nil {
					log.Printf("Failed to get file info: %v", err)
					response.Text = "Failed to process torrent file"
					delete(df.States, msg.Chat.ID)
					df.bot.api.Send(response)
					return
				}

				// Construct file URL
				fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", df.bot.api.Token, file.FilePath)
				state.link = fileURL
				state.step = 2
				df.sendCategoryButtons(msg.Chat.ID)
				return
			}

			// Invalid input
			response.Text = "Please send a valid magnet link or torrent file"
			delete(df.States, msg.Chat.ID)
		case 2: // Waiting for category selection
			var category common.RequestType
			switch msg.Text {
			case "🎬 Films":
				category = common.RequestType_FILMS
			case "📺 Series":
				category = common.RequestType_SERIES
			case "🎨 Cartoons":
				category = common.RequestType_CARTOONS
			case "🕸️ Cartoon Series":
				category = common.RequestType_CARTOONS_SERIES
			case "🩳 Cartoon Shorts":
				category = common.RequestType_SHORTS
			default:
				response.Text = "Please select a valid category"
				df.bot.api.Send(response)
				return
			}

			state.category = category
			state.step = 3

			// Start the download
			stream, err := df.bot.coordClient.AddTorrentByMagnet(context.Background(), &coordinator.AddTorrentByMagnetRequest{
				MagnetLink: state.link,
				Category:   state.category,
			})

			if err != nil {
				log.Printf("Failed to start download: %v", err)
				response.Text = "Failed to start download"
				delete(df.States, msg.Chat.ID)
				df.bot.api.Send(response)
				return
			}

			// Wait for the first response to confirm the download started
			resp, err := stream.Recv()
			if err != nil {
				log.Printf("Failed to get download status: %v", err)
				response.Text = "Failed to get download status"
				delete(df.States, msg.Chat.ID)
				df.bot.api.Send(response)
				return
			}

			// Start a goroutine to track download progress
			go df.trackDownloadProgress(msg.Chat.ID, stream)

			response.Text = "Download started! ID: " + resp.RequestId
			delete(df.States, msg.Chat.ID)

			// Remove the keyboard
			response.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		}
	} else {
		response.Text = "Please use /download command to start a new download"
	}

	df.bot.api.Send(response)
}

func (df *DownloadFlow) sendCategoryButtons(chatID int64) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🎬 Films"),
			tgbotapi.NewKeyboardButton("📺 Series"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🎨 Cartoons"),
			tgbotapi.NewKeyboardButton("📺 Cartoon Series"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🎨 Cartoon Shorts"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "Please select a category:")
	msg.ReplyMarkup = keyboard
	df.bot.api.Send(msg)
}

func (df *DownloadFlow) trackDownloadProgress(chatID int64, stream coordinator.CoordinatorService_AddTorrentByMagnetClient) {
	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Printf("Failed to receive download update: %v", err)
			return
		}

		// Send status update to user
		if resp.Status == coordinator.DownloadStatus_DOWNLOAD_STATUS_SUCCESS ||
			resp.Status == coordinator.DownloadStatus_DOWNLOAD_STATUS_ERROR {
			msg := tgbotapi.NewMessage(chatID, formatStatusMessage(resp))
			df.bot.api.Send(msg)
			return
		}
	}
}

func formatStatusMessage(resp *coordinator.DownloadResponse) string {
	var statusText string
	switch resp.Status {
	case coordinator.DownloadStatus_DOWNLOAD_STATUS_SUCCESS:
		statusText = "Completed"
	case coordinator.DownloadStatus_DOWNLOAD_STATUS_ERROR:
		statusText = "Error"
	default:
		statusText = "Unknown"
	}

	return fmt.Sprintf("Download ID: %s\nStatus: %s\nProgress: %.2f%%\nMessage: %s",
		resp.RequestId, statusText, resp.Progress, resp.Message)
}
