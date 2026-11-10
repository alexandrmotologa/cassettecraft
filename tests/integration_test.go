package tests_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/certs"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
	"github.com/alexandrmotologa/cassettecraft/internal/proxy"
)

func setupTestProxy(t *testing.T, cassetteDir string, mode config.Mode) (*proxy.Server, string, *certs.CA, func()) {
	cfg := config.DefaultConfig()
	cfg.Mode = mode
	cfg.CassetteDir = cassetteDir
	cfg.ActiveCass = "test_cassette.yaml"
	cfg.InsecureSkipVerify = true

	ca, err := certs.GenerateEphemeralCA()
	if err != nil {
		t.Fatalf("failed to generate CA: %v", err)
	}

	storage, err := cassette.NewStorage(cassetteDir, cfg.ActiveCass)
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	server := proxy.NewServer(cfg, ca, storage)

	// Bind to an ephemeral TCP port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on ephemeral port: %v", err)
	}

	go func() {
		_ = server.StartWithListener(listener)
	}()

	proxyAddr := listener.Addr().String()

	cleanup := func() {
		_ = server.Close()
		_ = listener.Close()
	}

	return server, proxyAddr, ca, cleanup
}

func createProxyClient(proxyAddr string, rootCA *certs.CA) *http.Client {
	proxyURL, _ := url.Parse("http://" + proxyAddr)

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(rootCA.RootCert)

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}
}

func TestRecordAndReplayCycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cassettecraft-integration-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Create Mock Upstream Server
	upstreamCallCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		upstreamCallCount++
		if req.URL.Path == "/v1/charge" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"ch_123","amount":2000,"status":"succeeded"}`))
			return
		}
		http.NotFound(w, req)
	}))
	defer upstream.Close()

	// 2. Start Proxy in RECORD Mode
	server, proxyAddr, ca, cleanup := setupTestProxy(t, tempDir, config.ModeRecord)
	client := createProxyClient(proxyAddr, ca)

	chargeURL := upstream.URL + "/v1/charge"
	reqPayload := []byte(`{"amount":2000,"currency":"usd","token":"tok_visa"}`)

	req, err := http.NewRequest(http.MethodPost, chargeURL, bytes.NewReader(reqPayload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk_test_secret_key_123456789012345")
	req.Header.Set("Content-Type", "application/json")

	// Execute request through proxy
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request in record mode: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "ch_123") {
		t.Fatalf("unexpected body: %s", string(body))
	}

	if upstreamCallCount != 1 {
		t.Fatalf("expected 1 upstream call, got %d", upstreamCallCount)
	}

	// Verify cassette was written to disk and secrets were redacted
	cassetteFile := filepath.Join(tempDir, "test_cassette.yaml")
	cassData, err := os.ReadFile(cassetteFile)
	if err != nil {
		t.Fatalf("cassette file not found on disk: %v", err)
	}

	cassContent := string(cassData)
	if strings.Contains(cassContent, "sk_test_secret_key") {
		t.Fatalf("unredacted secret found in cassette: %s", cassContent)
	}
	if !strings.Contains(cassContent, "[REDACTED_TOKEN]") && !strings.Contains(cassContent, "[REDACTED_STRIPE_KEY]") {
		t.Fatalf("expected redacted token placeholder in cassette")
	}

	// 3. Switch Proxy to REPLAY Mode and Close Upstream Server
	_ = server.SetMode(config.ModeReplay)
	upstream.Close() // Shut down upstream server completely!

	// Execute exact same request in Replay Mode
	replayReq, err := http.NewRequest(http.MethodPost, chargeURL, bytes.NewReader(reqPayload))
	if err != nil {
		t.Fatal(err)
	}
	replayReq.Header.Set("Authorization", "Bearer sk_test_secret_key_123456789012345")
	replayReq.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	replayResp, err := client.Do(replayReq)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("failed to replay request: %v", err)
	}
	defer replayResp.Body.Close()

	if replayResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from replay, got %d", replayResp.StatusCode)
	}

	if replayResp.Header.Get("X-CassetteCraft") != "REPLAY" {
		t.Fatalf("expected X-CassetteCraft header REPLAY, got %q", replayResp.Header.Get("X-CassetteCraft"))
	}

	replayBody, _ := io.ReadAll(replayResp.Body)
	if !strings.Contains(string(replayBody), "ch_123") {
		t.Fatalf("replay body mismatch: %s", string(replayBody))
	}

	// Verify sub-millisecond or fast local response
	t.Logf("Replay response served in %v", duration)

	cleanup()
}

func TestAutoMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cassettecraft-auto-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	callCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"call_number":%d}`, callCount)))
	}))
	defer upstream.Close()

	_, proxyAddr, ca, cleanup := setupTestProxy(t, tempDir, config.ModeAuto)
	defer cleanup()

	client := createProxyClient(proxyAddr, ca)
	url := upstream.URL + "/items"

	// Call 1: Not recorded yet -> should forward to upstream and record (callCount = 1)
	req1, _ := http.NewRequest(http.MethodGet, url, nil)
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("Call 1 failed: %v", err)
	}
	defer resp1.Body.Close()
	body1, _ := io.ReadAll(resp1.Body)
	if !strings.Contains(string(body1), `"call_number":1`) {
		t.Fatalf("expected call_number:1, got: %s", string(body1))
	}

	// Call 2: Already recorded -> should replay from cassette, NOT hitting upstream!
	req2, _ := http.NewRequest(http.MethodGet, url, nil)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("Call 2 failed: %v", err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	if !strings.Contains(string(body2), "call_number") || !strings.Contains(string(body2), "1") {
		t.Fatalf("expected replayed call_number 1, got: %s", string(body2))
	}

	// Upstream callCount must still be 1!
	if callCount != 1 {
		t.Fatalf("expected upstream call count to stay 1, got %d", callCount)
	}
}

func TestReplayNoMatchDiagnostic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cassettecraft-nomatch-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	_, proxyAddr, ca, cleanup := setupTestProxy(t, tempDir, config.ModeReplay)
	defer cleanup()

	client := createProxyClient(proxyAddr, ca)

	// Make request to an unrecorded endpoint
	unrecordedURL := "http://unrecorded-api.internal/v1/unknown"
	req, _ := http.NewRequest(http.MethodGet, unrecordedURL, nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502 Bad Gateway for unrecorded request, got %d", resp.StatusCode)
	}

	if resp.Header.Get("X-CassetteCraft-Status") != "NO_MATCH" {
		t.Fatalf("expected X-CassetteCraft-Status NO_MATCH, got %q", resp.Header.Get("X-CassetteCraft-Status"))
	}

	var diag map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &diag); err != nil {
		t.Fatalf("expected valid JSON diagnostic body, got: %s", string(body))
	}
}

func TestHTTPSInterception(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cassettecraft-https-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create HTTPS mock upstream server
	httpsUpstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"secure":true,"message":"hello from TLS"}`))
	}))
	defer httpsUpstream.Close()

	// Start Proxy in RECORD Mode
	_, proxyAddr, ca, cleanup := setupTestProxy(t, tempDir, config.ModeRecord)
	defer cleanup()

	client := createProxyClient(proxyAddr, ca)

	// Request to HTTPS upstream through proxy CONNECT tunnel
	req, _ := http.NewRequest(http.MethodGet, httpsUpstream.URL+"/v1/secure", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTPS request failed through proxy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "hello from TLS") {
		t.Fatalf("unexpected HTTPS response: %s", string(body))
	}
}
