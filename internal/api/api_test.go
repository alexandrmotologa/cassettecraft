package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alexandrmotologa/cassettecraft/internal/api"
	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/certs"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
	"github.com/alexandrmotologa/cassettecraft/internal/proxy"
)

func setupTestServer(t *testing.T) (*proxy.Server, *certs.CA, *api.Handler, func()) {
	tempDir, err := os.MkdirTemp("", "cassettecraft-api-test-*")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.CassetteDir = tempDir

	ca, err := certs.GenerateEphemeralCA()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := cassette.NewStorage(tempDir, "default.yaml")
	if err != nil {
		t.Fatal(err)
	}

	server := proxy.NewServer(cfg, ca, storage)
	handler := api.NewHandler(server, ca, nil)
	server.SetAPIHandler(handler)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return server, ca, handler, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	_, _, handler, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if res["status"] != "ok" {
		t.Fatalf("expected status 'ok', got %q", res["status"])
	}
}

func TestStatusAndModeEndpoint(t *testing.T) {
	server, _, handler, cleanup := setupTestServer(t)
	defer cleanup()

	// Initial status
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Change mode to replay
	modePayload := []byte(`{"mode":"replay"}`)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/mode", bytes.NewReader(modePayload))
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", postRec.Code, postRec.Body.String())
	}

	if server.GetMode() != config.ModeReplay {
		t.Fatalf("expected server mode replay, got %s", server.GetMode())
	}
}

func TestCADownloadEndpoint(t *testing.T) {
	_, ca, handler, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/ca.crt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !bytes.Equal(rec.Body.Bytes(), ca.CertPEM) {
		t.Fatal("downloaded cert does not match CA cert PEM")
	}
}

func TestRedactionSandboxEndpoint(t *testing.T) {
	_, _, handler, cleanup := setupTestServer(t)
	defer cleanup()

	fakeKey := "sk_" + "live_MockCustomerToken123456789012345"
	payload := []byte(`{"text":"Stripe token ` + fakeKey + ` in response"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/redact/test", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res["sanitized"] == res["original"] {
		t.Fatal("expected secret to be sanitized")
	}
}
