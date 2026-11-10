package cassette_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
)

func TestCassetteSerialization(t *testing.T) {
	cass := cassette.NewCassette()
	cass.Interactions = append(cass.Interactions, cassette.Interaction{
		ID:         "test_int_1",
		RecordedAt: time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC),
		Request: cassette.RequestData{
			Method: "POST",
			URL:    "https://api.example.com/v1/data",
			Headers: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
			Body: `{"name":"test","count":42}`,
		},
		Response: cassette.ResponseData{
			Status:     200,
			StatusText: "OK",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:      `{"status":"success"}`,
			LatencyMs: 45,
		},
	})

	yamlBytes, err := cass.ToYAML()
	if err != nil {
		t.Fatalf("failed to serialize YAML: %v", err)
	}

	loaded, err := cassette.FromYAML(yamlBytes)
	if err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	if len(loaded.Interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(loaded.Interactions))
	}
	if loaded.Interactions[0].Request.Method != "POST" {
		t.Fatalf("expected POST, got %s", loaded.Interactions[0].Request.Method)
	}

	// Test JSON serialization
	jsonBytes, err := cass.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize JSON: %v", err)
	}

	loadedJSON, err := cassette.FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if loadedJSON.Interactions[0].Response.Status != 200 {
		t.Fatalf("expected 200, got %d", loadedJSON.Interactions[0].Response.Status)
	}
}

func TestSanitizer(t *testing.T) {
	cfg := config.DefaultConfig().Redaction
	sanitizer := cassette.NewSanitizer(cfg)

	// Test Header Redaction
	headers := map[string]string{
		"Authorization": "Bearer sk_test_secret123456789012345",
		"Content-Type":  "application/json",
		"Cookie":        "session_id=abcdef123456",
		"X-Api-Key":     "secret_api_key_999",
	}

	cleanHeaders := sanitizer.SanitizeHeaders(headers)
	if cleanHeaders["Authorization"] != "Bearer [REDACTED_TOKEN]" {
		t.Fatalf("expected redacted bearer token, got %q", cleanHeaders["Authorization"])
	}
	if cleanHeaders["Cookie"] != "[REDACTED_COOKIE]" {
		t.Fatalf("expected redacted cookie, got %q", cleanHeaders["Cookie"])
	}
	if cleanHeaders["X-Api-Key"] != "[REDACTED_X_API_KEY]" {
		t.Fatalf("expected redacted X-Api-Key, got %q", cleanHeaders["X-Api-Key"])
	}
	if cleanHeaders["Content-Type"] != "application/json" {
		t.Fatalf("expected unchanged content-type, got %q", cleanHeaders["Content-Type"])
	}

	// Test URL Query Parameter Redaction
	rawURL := "https://api.example.com/v1/search?q=test&api_key=secret999&token=tok_abc123"
	cleanURL := sanitizer.SanitizeURL(rawURL)
	if strings.Contains(cleanURL, "secret999") || strings.Contains(cleanURL, "tok_abc123") {
		t.Fatalf("query secrets not redacted: %s", cleanURL)
	}
	if !strings.Contains(cleanURL, "q=test") {
		t.Fatalf("expected non-sensitive query param preserved: %s", cleanURL)
	}

	// Test Regex Token Redaction
	stripeKey := "sk_" + "live_MockCustomerToken123456789012345"
	ghToken := "gh" + "p_MockGitHubToken1234567890ABCDEFGHIJ99"
	bodyText := "User with Stripe key " + stripeKey + " and GitHub token " + ghToken + " card 4111 1111 1111 1111"
	cleanBody := sanitizer.SanitizeBody(bodyText)
	if strings.Contains(cleanBody, "sk_live_") || strings.Contains(cleanBody, "ghp_") || strings.Contains(cleanBody, "4111 1111") {
		t.Fatalf("tokens not redacted in body: %s", cleanBody)
	}
	if !strings.Contains(cleanBody, "[REDACTED_STRIPE_KEY]") {
		t.Fatalf("missing REDACTED_STRIPE_KEY replacement: %s", cleanBody)
	}
	if !strings.Contains(cleanBody, "[REDACTED_GITHUB_TOKEN]") {
		t.Fatalf("missing REDACTED_GITHUB_TOKEN replacement: %s", cleanBody)
	}

	// Test JSON Key Redaction
	jsonBody := `{
		"username": "alice",
		"password": "supersecretpassword",
		"access_token": "random_token_val",
		"nested": {
			"client_secret": "my_client_secret",
			"public_info": "safe"
		}
	}`
	cleanJSON := sanitizer.SanitizeBody(jsonBody)
	if strings.Contains(cleanJSON, "supersecretpassword") || strings.Contains(cleanJSON, "my_client_secret") {
		t.Fatalf("JSON keys not redacted: %s", cleanJSON)
	}
	if !strings.Contains(cleanJSON, "alice") || !strings.Contains(cleanJSON, "safe") {
		t.Fatalf("safe JSON fields removed: %s", cleanJSON)
	}
}

func TestStorage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cassettecraft-storage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := cassette.NewStorage(tempDir, "test.yaml")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	interaction := cassette.Interaction{
		ID: "int_storage_1",
		Request: cassette.RequestData{
			Method: "GET",
			URL:    "https://api.github.com/zen",
		},
		Response: cassette.ResponseData{
			Status: 200,
			Body:   "Responsive is better than fast.",
		},
	}

	err = storage.AppendInteraction("github.yaml", interaction)
	if err != nil {
		t.Fatalf("failed to append interaction: %v", err)
	}

	files, err := storage.ListCassettes()
	if err != nil {
		t.Fatalf("failed to list cassettes: %v", err)
	}
	if len(files) != 1 || files[0] != "github.yaml" {
		t.Fatalf("expected [github.yaml], got %v", files)
	}

	// Load cassette and assert interaction exists
	loaded, err := storage.Load("github.yaml")
	if err != nil {
		t.Fatalf("failed to load cassette: %v", err)
	}
	if len(loaded.Interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(loaded.Interactions))
	}
	if loaded.Interactions[0].Response.Body != "Responsive is better than fast." {
		t.Fatalf("unexpected body: %s", loaded.Interactions[0].Response.Body)
	}

	// Verify file is actually on disk
	onDisk := filepath.Join(tempDir, "github.yaml")
	if _, err := os.Stat(onDisk); err != nil {
		t.Fatalf("cassette file not on disk: %v", err)
	}
}
