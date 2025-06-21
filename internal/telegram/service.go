package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"fmt"
	"log"
)

// Service provides methods for interacting with the Telegram Bot API.
type Service struct {
	logger   *log.Logger
	bot      *tgbotapi.BotAPI
	httpPort int
}

// NewService creates a new Telegram Service.
func NewService(botToken string, httpPort int, logger *log.Logger) (*Service, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, err
	}
	bot.Debug = true // Enable debug mode for development

	logger.Printf("Authorized on account %s", bot.Self.UserName)

	return &Service{
		logger:   logger,
		bot:      bot,
		httpPort: httpPort,
	}, nil
}

// SendMessage sends a text message to a given chat ID.
func (s *Service) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := s.bot.Send(msg)
	return err
}

// StartPolling starts a long-polling loop to receive updates from Telegram.
// It should be run in a separate goroutine.
func (s *Service) StartPolling() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := s.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}

		if !update.Message.IsCommand() { // ignore any non-command Messages
			continue
		}

		// Handle the /start command
		if update.Message.Command() == "start" {
			s.handleStartCommand(update.Message)
		}
	}
}

func (s *Service) handleStartCommand(message *tgbotapi.Message) {
	s.logger.Printf("Received /start command from user %d in chat %d", message.From.ID, message.Chat.ID)

	// Generate a unique link for the user to connect their account.
	token := message.From.ID
	connectURL := fmt.Sprintf("http://localhost:%d/telegram/connect?token=%d", s.httpPort, token)

	responseText := fmt.Sprintf("Welcome! To connect your account and receive email digests, please click this link:\n\n%s", connectURL)

	if err := s.SendMessage(message.Chat.ID, responseText); err != nil {
		s.logger.Printf("Failed to send connect message to user %d: %v", message.From.ID, err)
	}
} 