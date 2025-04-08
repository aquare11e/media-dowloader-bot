package bot

import (
	"log"
	"strings"

	coordinator "github.com/aquare11e/media-dowloader-bot/common/protogen/coordinator"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api          *tgbotapi.BotAPI
	allowedUsers map[string]bool
	coordClient  coordinator.CoordinatorServiceClient
	downloadFlow *DownloadFlow
}

type Config struct {
	Token             string
	AllowedUsers      []string
	CoordinatorClient coordinator.CoordinatorServiceClient
}

func NewBot(cfg Config) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, err
	}

	allowedUsers := make(map[string]bool)
	for _, user := range cfg.AllowedUsers {
		allowedUsers[strings.TrimSpace(user)] = true
	}

	b := &Bot{
		api:          bot,
		allowedUsers: allowedUsers,
		coordClient:  cfg.CoordinatorClient,
	}

	b.downloadFlow = NewDownloadFlow(b)
	return b, nil
}

func (b *Bot) Start() {
	log.Printf("Authorized on account %s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			if !b.allowedUsers[update.Message.From.UserName] {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry, you are not authorized to use this bot.")
				b.api.Send(msg)
				continue
			}

			if update.Message.IsCommand() {
				b.handleCommand(update.Message)
			} else {
				b.downloadFlow.HandleMessage(update.Message)
			}
		}
	}
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	response := tgbotapi.NewMessage(msg.Chat.ID, "")

	switch msg.Command() {
	case "start":
		response.Text = "Hello! I am a private bot that can help you download torrents."
	case "download":
		b.downloadFlow.Start(msg.Chat.ID)
	default:
		response.Text = "I don't know that command"
	}

	b.api.Send(response)
}
