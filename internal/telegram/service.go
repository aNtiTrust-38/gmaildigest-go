package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

// Service provides methods for interacting with the Telegram Bot API.
type Service struct {
	logger *log.Logger
	bot    *tgbotapi.BotAPI
}

// NewService creates a new Telegram Service.
func NewService(botToken string, logger *log.Logger) (*Service, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, err
	}
	bot.Debug = true // Enable debug mode for development

	logger.Printf("Authorized on account %s", bot.Self.UserName)

	return &Service{
		logger: logger,
		bot:    bot,
	}, nil
}

// SendMessage sends a text message to a given chat ID.
func (s *Service) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := s.bot.Send(msg)
	return err
} 