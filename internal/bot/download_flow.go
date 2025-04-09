package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	common "github.com/aquare11e/media-downloader-bot/common/protogen/common"
	coordinatorpb "github.com/aquare11e/media-downloader-bot/common/protogen/coordinator"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type Step int

const (
	StepWaitingForLink Step = iota + 1
	StepWaitingForCategory
	StepDownloading
)

type downloadState struct {
	step     Step
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
		step: StepWaitingForLink,
	}

	response := tgbotapi.NewMessage(chatID, "✨ Awesome! Please send me a magnet link or torrent file to begin your download journey!")
	df.bot.api.Send(response)
}

func (df *DownloadFlow) HandleMessage(msg *tgbotapi.Message) {
	response := tgbotapi.NewMessage(msg.Chat.ID, "")

	if state, exists := df.States[msg.Chat.ID]; exists {
		switch state.step {
		case StepWaitingForLink:
			df.handleWaitingForLinkStep(msg, state, response)
		case StepWaitingForCategory:
			df.handleWaitingForCategoryStep(msg, state, response)
		}
	} else {
		response.Text = "Please use /download command to start a new download"
		df.bot.api.Send(response)
	}
}

func (df *DownloadFlow) handleWaitingForLinkStep(msg *tgbotapi.Message, state *downloadState, response tgbotapi.MessageConfig) {
	// Check if it's a magnet link
	if strings.HasPrefix(msg.Text, "magnet:?xt=urn:btih:") {
		state.link = msg.Text
		state.step = StepWaitingForCategory
		df.sendCategoryButtons(msg.Chat.ID)
		return
	}

	// Check if it's a document (torrent file)
	if msg.Document != nil && strings.HasSuffix(msg.Document.FileName, ".torrent") {
		// Get file info
		file, err := df.bot.api.GetFile(tgbotapi.FileConfig{FileID: msg.Document.FileID})
		if err != nil {
			log.Printf("Failed to get file info: %v", err)
			response.Text = "❌ Oops! I couldn't process your torrent file. Please try again!"
			delete(df.States, msg.Chat.ID)
			df.bot.api.Send(response)
			return
		}

		// Construct file URL
		fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", df.bot.api.Token, file.FilePath)
		state.link = fileURL
		state.step = StepWaitingForCategory
		df.sendCategoryButtons(msg.Chat.ID)
		return
	}

	// Invalid input
	response.Text = "❌ Please send a valid magnet link or torrent file. I'm here to help you download your content!"
	delete(df.States, msg.Chat.ID)
	df.bot.api.Send(response)
}

func (df *DownloadFlow) handleWaitingForCategoryStep(msg *tgbotapi.Message, state *downloadState, response tgbotapi.MessageConfig) {
	var category common.RequestType
	switch msg.Text {
	case filmsCategory:
		category = common.RequestType_FILMS
	case seriesCategory:
		category = common.RequestType_SERIES
	case cartoonsCategory:
		category = common.RequestType_CARTOONS
	case cartoonsSeriesCategory:
		category = common.RequestType_CARTOONS_SERIES
	case cartoonsShortsCategory:
		category = common.RequestType_SHORTS
	default:
		response.Text = "❌ Please select a valid category from the options below"
		df.bot.api.Send(response)
		return
	}

	state.category = category
	state.step = StepDownloading

	// Start the download
	resp, err := df.bot.coordClient.AddTorrentByMagnet(context.Background(), &coordinatorpb.AddTorrentByMagnetRequest{
		RequestId:  uuid.New().String(),
		MagnetLink: state.link,
		Category:   state.category,
	})

	if err != nil {
		log.Printf("Failed to start download: %v", err)
		response.Text = "❌ Oops! I couldn't start the download. Please try again later!"
		delete(df.States, msg.Chat.ID)
		df.bot.api.Send(response)
		return
	}

	status := &DownloadStatus{
		Name:     resp.Name,
		Status:   resp.Status,
		Message:  resp.Message,
		ETA:      time.Duration(resp.Eta) * time.Second,
		Progress: resp.Progress,
	}

	err1 := df.bot.redisClient.HSet(context.Background(), fmt.Sprintf(KeyTorrentInProgress, resp.RequestId), status.ToRedisMap()).Err()
	err2 := df.bot.redisClient.SAdd(context.Background(), KeyTorrentInProgressKeys, resp.RequestId).Err()
	err3 := df.bot.redisClient.Set(context.Background(), fmt.Sprintf(KeyTorrentDownloadOwner, resp.RequestId), msg.Chat.ID, 24*time.Hour).Err()
	if err1 != nil || err2 != nil || err3 != nil {
		log.Printf("Failed to set status in Redis: \ndetails: %v, \nkeys: %v, \nowner: %v", err1, err2, err3)
		response.Text = "⚠️ Download started, but I couldn't save the status locally. You can check the status using /status command"
		delete(df.States, msg.Chat.ID)
		df.bot.api.Send(response)
		return
	}

	response.Text = "✅ Download started!\n📁 Torrent name: " + resp.Name
	delete(df.States, msg.Chat.ID)

	// Remove the keyboard
	response.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	df.bot.api.Send(response)
}

func (df *DownloadFlow) sendCategoryButtons(chatID int64) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(filmsCategory),
			tgbotapi.NewKeyboardButton(seriesCategory),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(cartoonsCategory),
			tgbotapi.NewKeyboardButton(cartoonsSeriesCategory),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(cartoonsShortsCategory),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🎬 Please select a category for your content:")
	msg.ReplyMarkup = keyboard
	df.bot.api.Send(msg)
}
