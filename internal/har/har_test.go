package har_test

import (
	"strings"
	"testing"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
	"github.com/alexandrmotologa/cassettecraft/internal/har"
)

func TestConvertHARToCassette(t *testing.T) {
	harJSON := `{
  "log": {
    "version": "1.2",
    "creator": { "name": "Chrome DevTools", "version": "130.0" },
    "entries": [
      {
        "startedDateTime": "2026-11-10T12:00:00.000Z",
        "time": 150.5,
        "request": {
          "method": "POST",
          "url": "https://api.stripe.com/v1/customers",
          "httpVersion": "HTTP/1.1",
          "headers": [
            { "name": "Authorization", "value": "Bearer sk_test_sample1234567890abcdef" },
            { "name": "Content-Type", "value": "application/json" }
          ],
          "postData": {
            "mimeType": "application/json",
            "text": "{\"email\":\"user@example.com\"}"
          }
        },
        "response": {
          "status": 200,
          "statusText": "OK",
          "httpVersion": "HTTP/1.1",
          "headers": [
            { "name": "Content-Type", "value": "application/json" }
          ],
          "content": {
            "size": 55,
            "mimeType": "application/json",
            "text": "{\"id\":\"cus_123\",\"email\":\"user@example.com\"}"
          }
        }
      }
    ]
  }
}`

	sanitizer := cassette.NewSanitizer(config.DefaultConfig().Redaction)
	cass, err := har.ConvertHARToCassette([]byte(harJSON), sanitizer)
	if err != nil {
		t.Fatalf("failed to convert HAR: %v", err)
	}

	if len(cass.Interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(cass.Interactions))
	}

	inter := cass.Interactions[0]
	if inter.Request.Method != "POST" {
		t.Fatalf("expected POST, got %s", inter.Request.Method)
	}

	// Verify Authorization was sanitized
	if inter.Request.Headers["Authorization"] != "Bearer [REDACTED_TOKEN]" {
		t.Fatalf("expected sanitized auth header, got: %s", inter.Request.Headers["Authorization"])
	}

	if inter.Response.Status != 200 {
		t.Fatalf("expected status 200, got %d", inter.Response.Status)
	}
}

func TestGenerateCurl(t *testing.T) {
	req := cassette.RequestData{
		Method: "POST",
		URL:    "https://api.example.com/v1/users",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"name":"Alice"}`,
	}

	curl := har.GenerateCurl(req)

	if !strings.Contains(curl, "curl -X POST") {
		t.Fatalf("expected curl -X POST, got %s", curl)
	}
	if !strings.Contains(curl, "https://api.example.com/v1/users") {
		t.Fatalf("missing URL in curl: %s", curl)
	}
	if !strings.Contains(curl, "Content-Type: application/json") {
		t.Fatalf("missing header in curl: %s", curl)
	}
}
