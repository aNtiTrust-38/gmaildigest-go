package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gmaildigest-go/pkg/models"

	"google.golang.org/api/gmail/v1"
	"encoding/base64"
)

// Service provides methods for interacting with the Gmail API.
type Service struct {
	logger *log.Logger
	srv    *gmail.Service
}

// NewService creates a new Gmail Service.
func NewService(gmailService *gmail.Service, logger *log.Logger) *Service {
	return &Service{
		logger: logger,
		srv:    gmailService,
	}
}

// FetchUnreadEmails fetches unread emails for the authenticated user.
func (s *Service) FetchUnreadEmails(ctx context.Context) ([]models.Email, error) {
	// 1. List unread messages
	listCall := s.srv.Users.Messages.List("me").Q("is:unread")
	listResp, err := listCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	if len(listResp.Messages) == 0 {
		s.logger.Println("No unread messages found.")
		return nil, nil
	}

	var emails []models.Email
	// 2. Fetch each message
	for _, msgRef := range listResp.Messages {
		getCall := s.srv.Users.Messages.Get("me", msgRef.Id).Format("full")
		msg, err := getCall.Do()
		if err != nil {
			s.logger.Printf("Failed to get message %s: %v", msgRef.Id, err)
			continue // Skip to the next message
		}

		// Add detailed logging
		msgJSON, _ := json.Marshal(msg)
		s.logger.Printf("Received message %s: %s", msgRef.Id, string(msgJSON))

		parsedEmail, err := s.parseEmail(msg)
		if err != nil {
			s.logger.Printf("Failed to parse email %s: %v", msg.Id, err)
			continue
		}
		emails = append(emails, *parsedEmail)
	}

	return emails, nil
}

// parseEmail converts a gmail.Message into our internal models.Email format.
func (s *Service) parseEmail(msg *gmail.Message) (*models.Email, error) {
	if msg == nil {
		return nil, fmt.Errorf("cannot parse nil message")
	}
	if msg.Payload == nil {
		return nil, fmt.Errorf("message %q has no payload", msg.Id)
	}

	email := &models.Email{
		ID: msg.Id,
	}

	for _, h := range msg.Payload.Headers {
		switch h.Name {
		case "Subject":
			email.Subject = h.Value
		case "From":
			email.From = h.Value
		case "Date":
			// Common date formats found in email headers
			formats := []string{
				time.RFC1123Z,
				"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
				"Mon, 2 Jan 2006 15:04:05 -0700",
				"2 Jan 2006 15:04:05 -0700",
			}
			var t time.Time
			var err error
			for _, format := range formats {
				t, err = time.Parse(format, h.Value)
				if err == nil {
					break
				}
			}
			if err != nil {
				s.logger.Printf("Could not parse date %q using known formats: %v", h.Value, err)
				// Fallback to now if date can't be parsed
				t = time.Now()
			}
			email.ReceivedAt = t
		}
	}

	// Find the body part and decode it
	if msg.Payload.Body.Data != "" {
		body, err := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode body: %w", err)
		}
		email.Body = string(body)
	} else {
		// Handle multipart messages
		for i := 0; i < len(msg.Payload.Parts); i++ {
			part := msg.Payload.Parts[i]
			if part == nil {
				continue
			}
			partJSON, _ := json.Marshal(part)
			s.logger.Printf("Processing part: %s", string(partJSON))
			if part.Body != nil && part.MimeType == "text/plain" && part.Body.Data != "" {
				body, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err != nil {
					return nil, fmt.Errorf("failed to decode multipart body: %w", err)
				}
				email.Body = string(body)
				break
			}
		}
	}

	if email.Subject == "" {
		email.Subject = "(No Subject)"
	}
	if email.Body == "" {
		email.Body = msg.Snippet
	}

	return email, nil
} 