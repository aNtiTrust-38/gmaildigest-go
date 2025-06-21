package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"log"
	"gmaildigest-go/pkg/models"
	"time"
)

// Service provides methods for interacting with the Gmail API.
type Service struct {
	logger *log.Logger
	srv    *gmail.Service
}

// NewService creates a new Gmail Service.
func NewService(ctx context.Context, token *oauth2.Token, logger *log.Logger) (*Service, error) {
	config := &oauth2.Config{} // This can be empty as we're providing a token source
	client := config.Client(ctx, token)
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &Service{
		logger: logger,
		srv:    srv,
	}, nil
}

// FetchUnreadEmailSubjects fetches the subjects of unread emails.
// This is a simplified version for now.
func (s *Service) FetchUnreadEmailSubjects(ctx context.Context) ([]string, error) {
	// Implementation to follow.
	return []string{"Subject 1: Test", "Subject 2: Another Test"}, nil
}

// FetchUnreadEmails fetches the subjects and bodies of unread emails.
func (s *Service) FetchUnreadEmails(ctx context.Context) ([]models.Email, error) {
	var emails []models.Email

	listResp, err := s.srv.Users.Messages.List("me").Q("is:unread").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}

	for _, msgRef := range listResp.Messages {
		msg, err := s.srv.Users.Messages.Get("me", msgRef.Id).Format("full").Do()
		if err != nil {
			s.logger.Printf("Failed to get message %s: %v", msgRef.Id, err)
			continue
		}

		email, err := s.parseEmail(msg)
		if err != nil {
			s.logger.Printf("Failed to parse email %s: %v", msg.Id, err)
			continue
		}
		emails = append(emails, *email)

		// Mark email as read
		modifyReq := &gmail.ModifyMessageRequest{
			RemoveLabelIds: []string{"UNREAD"},
		}
		if _, err := s.srv.Users.Messages.Modify("me", msg.Id, modifyReq).Do(); err != nil {
			s.logger.Printf("Failed to mark message %s as read: %v", msg.Id, err)
			// Continue processing even if marking as read fails
		}
	}

	return emails, nil
}

func (s *Service) parseEmail(msg *gmail.Message) (*models.Email, error) {
	email := &models.Email{ID: msg.Id}
	if msg.Payload == nil {
		return nil, fmt.Errorf("message %q has no payload", msg.Id)
	}

	for _, h := range msg.Payload.Headers {
		switch h.Name {
		case "Subject":
			email.Subject = h.Value
		case "From":
			email.From = h.Value
		case "Date":
			t, err := time.Parse(time.RFC1123Z, h.Value)
			if err == nil {
				email.Date = t
			}
		}
	}

	if msg.Payload.Body.Data != "" {
		body, err := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		if err == nil {
			email.Body = string(body)
		}
	} else {
		for _, part := range msg.Payload.Parts {
			if part == nil || part.Body == nil {
				continue
			}
			if part.MimeType == "text/plain" && part.Body.Data != "" {
				body, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err == nil {
					email.Body = string(body)
					break
				}
			}
		}
	}

	return email, nil
} 