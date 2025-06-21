package gmail

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// mockRoundTripper is a custom http.RoundTripper for mocking API responses.
type mockRoundTripper struct {
	roundTripFunc func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

// newMockTransport creates a new http.RoundTripper that serves different
// responses based on the request URL.
func newMockTransport(t *testing.T, responses map[string]string) http.RoundTripper {
	return &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			path := req.URL.Path
			t.Logf("Mock transport received request for path: %s", path)

			// The path includes API version and user ID, so we use Contains.
			for urlPart, body := range responses {
				if strings.Contains(path, urlPart) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
						Header:     make(http.Header),
					}, nil
				}
			}
			t.Errorf("No mock response found for path: %s", path)
			return nil, http.ErrAbortHandler
		},
	}
}

func newTestClient(t *testing.T, responses map[string]string) *http.Client {
	return &http.Client{
		Transport: newMockTransport(t, responses),
	}
}

func TestService_FetchUnreadEmails(t *testing.T) {
	t.Run("successfully fetches and parses an unread email", func(t *testing.T) {
		// Mock responses
		listResponse := &gmail.ListMessagesResponse{
			Messages: []*gmail.Message{
				{Id: "msg1"},
			},
		}
		listBody, _ := json.Marshal(listResponse)

		msg1Response := &gmail.Message{
			Id:      "msg1",
			Snippet: "This is the first email.",
			Payload: &gmail.MessagePart{
				Headers: []*gmail.MessagePartHeader{
					{Name: "Subject", Value: "Subject 1"},
					{Name: "From", Value: "sender1@example.com"},
					{Name: "Date", Value: "Tue, 25 Jun 2024 10:00:00 +0000"},
				},
				Body: &gmail.MessagePartBody{
					Data: "VGhpcyBpcyB0aGUgYm9keSBvZiB0aGUgZmlyc3QgZW1haWwu", // "This is the body of the first email."
				},
			},
		}
		msg1Body, _ := json.Marshal(msg1Response)

		responses := map[string]string{
			"/messages":      string(listBody),
			"/messages/msg1": string(msg1Body),
		}

		client := newTestClient(t, responses)
		ctx := context.Background()

		gmailSrv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
		require.NoError(t, err)

		service := NewService(gmailSrv, log.New(testWriter{t}, "", 0))

		emails, err := service.FetchUnreadEmails(ctx)
		require.NoError(t, err)

		assert.Len(t, emails, 1)
		assert.Equal(t, "Subject 1", emails[0].Subject)
		assert.Equal(t, "sender1@example.com", emails[0].From)
		assert.Equal(t, "This is the body of the first email.", emails[0].Body)
	})
}

// testWriter is a helper to redirect log output to the test log.
type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Log(string(p))
	return len(p), nil
} 