package summary

import "strings"

// Service provides methods for summarizing text.
type Service struct{}

// NewService creates a new Summary Service.
func NewService() *Service {
	return &Service{}
}

// Summarize creates a simple summary of a list of texts.
// This is a placeholder and does not use an AI model.
func (s *Service) Summarize(texts []string) (string, error) {
	if len(texts) == 0 {
		return "No new emails to summarize.", nil
	}

	var summary strings.Builder
	summary.WriteString("Here is your email digest:\n\n")

	for i, text := range texts {
		if i >= 5 { // Limit to 5 emails for the summary
			break
		}
		summary.WriteString("- ")
		summary.WriteString(text)
		summary.WriteString("\n")
	}

	return summary.String(), nil
} 