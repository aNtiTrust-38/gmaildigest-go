package summary

import (
	"context"
	"fmt"
	"gmaildigest-go/pkg/models"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// Service provides methods for summarizing text.
type Service struct {
	client *openai.Client
}

// NewService creates a new Summary Service.
func NewService(apiKey string) *Service {
	client := openai.NewClient(apiKey)
	return &Service{client: client}
}

// Summarize creates a summary of a list of emails using the OpenAI API.
func (s *Service) Summarize(ctx context.Context, emails []models.Email) (string, error) {
	if len(emails) == 0 {
		return "No new emails to summarize.", nil
	}

	// Prepare the content for the prompt
	var contentBuilder strings.Builder
	contentBuilder.WriteString("Please provide a concise summary of the following emails:\n\n")
	for _, email := range emails {
		contentBuilder.WriteString(fmt.Sprintf("From: %s\n", email.From))
		contentBuilder.WriteString(fmt.Sprintf("Subject: %s\n", email.Subject))
		contentBuilder.WriteString(fmt.Sprintf("Body: %s\n\n", email.Body))
	}

	// Call the OpenAI API
	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: contentBuilder.String(),
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no summary returned from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
} 