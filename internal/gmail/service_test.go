package gmail

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/gmail/v1"
)

func TestService_ParseEmail(t *testing.T) {
	service := &Service{logger: log.New(io.Discard, "", 0)}

	t.Run("handles multipart body", func(t *testing.T) {
		msg := &gmail.Message{
			Id:      "multipart-msg",
			Payload: &gmail.MessagePart{
				Headers: []*gmail.MessagePartHeader{{Name: "Subject", Value: "Multipart"}},
				Parts: []*gmail.MessagePart{
					{
						MimeType: "text/plain",
						Body:     &gmail.MessagePartBody{Data: "VGhpcyBpcyB0aGUgcGFydCBib2R5Lg=="}, // "This is the part body."
					},
				},
			},
		}
		email, err := service.parseEmail(msg)
		require.NoError(t, err)
		assert.Equal(t, "This is the part body.", email.Body)
	})
} 