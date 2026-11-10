package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/certs"
	"github.com/alexandrmotologa/cassettecraft/internal/chaos"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
)

// Server is the cassettecraft HTTP/HTTPS proxy daemon.
type Server struct {
	cfg        *config.Config
	ca         *certs.CA
	storage    *cassette.Storage
	sanitizer  *cassette.Sanitizer
	matcher    *Matcher
	chaos      *chaos.Chaos
	buffer     *InteractionBuffer
	httpServer *http.Server
	apiHandler http.Handler
	mu         sync.RWMutex
	mode       config.Mode
}

// NewServer initializes a new proxy server instance.
func NewServer(cfg *config.Config, ca *certs.CA, storage *cassette.Storage) *Server {
	sanitizer := cassette.NewSanitizer(cfg.Redaction)
	matcher := NewMatcher(cfg.Matching)
	chaosSim := chaos.NewChaos(cfg.Chaos)
	buffer := NewInteractionBuffer(1000)

	s := &Server{
		cfg:       cfg,
		ca:        ca,
		storage:   storage,
		sanitizer: sanitizer,
		matcher:   matcher,
		chaos:     chaosSim,
		buffer:    buffer,
		mode:      cfg.Mode,
	}

	return s
}

// SetAPIHandler attaches the REST control and Studio API router to the server.
func (s *Server) SetAPIHandler(h http.Handler) {
	s.apiHandler = h
}

// GetMode returns the current operating proxy mode.
func (s *Server) GetMode() config.Mode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

// SetMode dynamically changes the operating proxy mode.
func (s *Server) SetMode(m config.Mode) error {
	if !m.IsValid() {
		return fmt.Errorf("invalid mode: %s", m)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = m
	return nil
}

// GetBuffer returns the interaction ring buffer.
func (s *Server) GetBuffer() *InteractionBuffer {
	return s.buffer
}

// GetChaos returns the chaos simulator instance.
func (s *Server) GetChaos() *chaos.Chaos {
	return s.chaos
}

// GetStorage returns the cassette storage instance.
func (s *Server) GetStorage() *cassette.Storage {
	return s.storage
}

// GetSanitizer returns the secret sanitizer instance.
func (s *Server) GetSanitizer() *cassette.Sanitizer {
	return s.sanitizer
}

// Start begins listening on the configured port.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      http.HandlerFunc(s.ServeHTTP),
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  180 * time.Second,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind proxy to %s: %w", addr, err)
	}

	return s.httpServer.Serve(listener)
}

// StartWithListener starts the proxy using an existing net.Listener.
func (s *Server) StartWithListener(l net.Listener) error {
	s.httpServer = &http.Server{
		Handler:      http.HandlerFunc(s.ServeHTTP),
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  180 * time.Second,
	}
	return s.httpServer.Serve(l)
}

// Close gracefully terminates the proxy server.
func (s *Server) Close() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

// ServeHTTP handles both direct API requests and forwarded proxy traffic.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 1. Check if this is an API or Studio request
	if s.isInternalRequest(req) {
		if s.apiHandler != nil {
			s.apiHandler.ServeHTTP(w, req)
			return
		}
	}

	// 2. Handle HTTPS CONNECT tunnels
	if req.Method == http.MethodConnect {
		s.handleConnect(w, req)
		return
	}

	// 3. Handle plain HTTP forward proxy requests
	s.handlePlainHTTP(w, req)
}

func (s *Server) isInternalRequest(req *http.Request) bool {
	// If the host is localhost, 127.0.0.1, or matches the proxy's own port without a remote scheme
	hostWithoutPort, _, _ := net.SplitHostPort(req.Host)
	if hostWithoutPort == "" {
		hostWithoutPort = req.Host
	}

	isLocalHost := hostWithoutPort == "localhost" || hostWithoutPort == "127.0.0.1" || hostWithoutPort == "0.0.0.0"

	if isLocalHost {
		if strings.HasPrefix(req.URL.Path, "/api/") ||
			req.URL.Path == "/ca.crt" ||
			req.URL.Path == "/" ||
			strings.HasPrefix(req.URL.Path, "/assets/") ||
			strings.HasPrefix(req.URL.Path, "/studio") {
			return true
		}
	}

	return false
}

func (s *Server) handleConnect(w http.ResponseWriter, req *http.Request) {
	targetHost := req.Host
	if !strings.Contains(targetHost, ":") {
		targetHost += ":443"
	}

	// Hijack the underlying TCP connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to hijack connection: %v", err), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Acknowledge the CONNECT tunnel
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		return
	}

	// Establish TLS server handshake with client using dynamically signed certificate
	tlsConfig := &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			serverName := info.ServerName
			if serverName == "" {
				serverName = targetHost
			}
			return s.ca.GetCertificate(serverName)
		},
		NextProtos: []string{"http/1.1"},
	}
	tlsConn := tls.Server(clientConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return
	}
	defer tlsConn.Close()

	// Process decrypted HTTP requests over the TLS tunnel in a loop for keep-alive
	reader := bufio.NewReader(tlsConn)
	for {
		tlsReq, err := http.ReadRequest(reader)
		if err != nil {
			break
		}

		// Ensure scheme and host are populated
		tlsReq.URL.Scheme = "https"
		if tlsReq.URL.Host == "" {
			tlsReq.URL.Host = targetHost
		}
		tlsReq.Host = targetHost

		// Process request and write response back to TLS connection
		respBytes, shouldClose := s.processDecryptedRequest(tlsReq, true)
		if _, writeErr := tlsConn.Write(respBytes); writeErr != nil {
			break
		}

		if shouldClose || tlsReq.Close {
			break
		}
	}
}

func (s *Server) handlePlainHTTP(w http.ResponseWriter, req *http.Request) {
	respBytes, _ := s.processDecryptedRequest(req, false)

	// Parse generated response bytes and forward to ResponseWriter
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(respBytes)), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Proxy internal error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) processDecryptedRequest(req *http.Request, isTLS bool) ([]byte, bool) {
	// Read request body
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Determine active mode and target cassette
	activeMode := s.GetMode()
	if headerMode := req.Header.Get("X-Cassette-Mode"); headerMode != "" {
		m := config.Mode(strings.ToLower(headerMode))
		if m.IsValid() {
			activeMode = m
		}
	}

	targetCassName := s.storage.GetActiveName()
	if headerCass := req.Header.Get("X-Cassette-Name"); headerCass != "" {
		targetCassName = headerCass
	}

	// Check chaos simulator for synthetic errors
	if shouldError, errStatus, errText := s.chaos.ShouldSimulateError(); shouldError {
		return s.formatHTTPResponse(errStatus, map[string]string{
			"Content-Type":             "text/plain",
			"X-CassetteCraft-SimError": "true",
		}, []byte(errText)), true
	}

	// Apply synthetic latency if configured
	s.chaos.ApplyLatency()

	fullURL := s.buildFullURL(req, isTLS)
	reqHeadersMap := extractHeaderMap(req.Header)

	// Pre-sanitize request parameters for deterministic replay matching
	matchURL := s.sanitizer.SanitizeURL(fullURL)
	matchHeaders := s.sanitizer.SanitizeHeaders(reqHeadersMap)
	matchBody := []byte(s.sanitizer.SanitizeBody(string(bodyBytes)))

	var statusCode int
	var resHeaders map[string]string
	var resBody []byte
	var latencyMs int64
	var matched bool
	var errDetails string

	startTime := time.Now()

	switch activeMode {
	case config.ModeReplay:
		cass, err := s.storage.Load(targetCassName)
		if err != nil {
			errDetails = fmt.Sprintf("Failed to load cassette %q: %v", targetCassName, err)
			statusCode = http.StatusBadGateway
			resHeaders = map[string]string{"Content-Type": "application/json", "X-CassetteCraft-Status": "ERROR"}
			resBody = []byte(fmt.Sprintf(`{"error": %q}`, errDetails))
			break
		}

		result := s.matcher.FindMatch(cass, req.Method, matchURL, matchHeaders, matchBody)
		if result.Matched != nil {
			matched = true
			statusCode = result.Matched.Response.Status
			resHeaders = copyMap(result.Matched.Response.Headers)
			if resHeaders == nil {
				resHeaders = make(map[string]string)
			}
			resHeaders["X-CassetteCraft"] = "REPLAY"
			resHeaders["X-CassetteCraft-ID"] = result.Matched.ID
			resBody = []byte(result.Matched.Response.Body)
			latencyMs = time.Since(startTime).Milliseconds()
		} else {
			statusCode = http.StatusBadGateway
			diagJSON, _ := json.MarshalIndent(result.Diagnostics, "", "  ")
			resHeaders = map[string]string{
				"Content-Type":             "application/json",
				"X-CassetteCraft-Status":   "NO_MATCH",
				"X-CassetteCraft-Cassette": targetCassName,
			}
			resBody = diagJSON
			errDetails = fmt.Sprintf("No match found in %s", targetCassName)
			latencyMs = time.Since(startTime).Milliseconds()
		}

	case config.ModeRecord:
		captured, err := ForwardRequest(req, bodyBytes, isTLS, s.cfg.InsecureSkipVerify)
		if err != nil {
			errDetails = fmt.Sprintf("Upstream forward error: %v", err)
			statusCode = http.StatusBadGateway
			resHeaders = map[string]string{"Content-Type": "application/json", "X-CassetteCraft-Status": "FORWARD_ERROR"}
			resBody = []byte(fmt.Sprintf(`{"error": %q}`, errDetails))
			latencyMs = time.Since(startTime).Milliseconds()
			break
		}

		statusCode = captured.Status
		resHeaders = copyMap(captured.Headers)
		resHeaders["X-CassetteCraft"] = "RECORD"
		resBody = captured.Body
		latencyMs = captured.LatencyMs

		// Create and sanitize interaction
		inter := cassette.Interaction{
			ID:         fmt.Sprintf("int_%d", time.Now().UnixNano()),
			RecordedAt: time.Now().UTC(),
			Request: cassette.RequestData{
				Method:  req.Method,
				URL:     fullURL,
				Headers: reqHeadersMap,
				Body:    string(bodyBytes),
			},
			Response: cassette.ResponseData{
				Status:     captured.Status,
				StatusText: captured.StatusText,
				Headers:    captured.Headers,
				Body:       string(captured.Body),
				LatencyMs:  captured.LatencyMs,
			},
		}

		s.sanitizer.SanitizeInteraction(&inter)
		_ = s.storage.AppendInteraction(targetCassName, inter)
		matched = true

	case config.ModeAuto:
		// Attempt replay first
		cass, err := s.storage.Load(targetCassName)
		if err == nil {
			result := s.matcher.FindMatch(cass, req.Method, matchURL, matchHeaders, matchBody)
			if result.Matched != nil {
				matched = true
				statusCode = result.Matched.Response.Status
				resHeaders = copyMap(result.Matched.Response.Headers)
				if resHeaders == nil {
					resHeaders = make(map[string]string)
				}
				resHeaders["X-CassetteCraft"] = "REPLAY"
				resHeaders["X-CassetteCraft-ID"] = result.Matched.ID
				resBody = []byte(result.Matched.Response.Body)
				latencyMs = time.Since(startTime).Milliseconds()
				break
			}
		}

		// Fallback to record
		captured, err := ForwardRequest(req, bodyBytes, isTLS, s.cfg.InsecureSkipVerify)
		if err != nil {
			errDetails = fmt.Sprintf("Upstream forward error: %v", err)
			statusCode = http.StatusBadGateway
			resHeaders = map[string]string{"Content-Type": "application/json", "X-CassetteCraft-Status": "FORWARD_ERROR"}
			resBody = []byte(fmt.Sprintf(`{"error": %q}`, errDetails))
			latencyMs = time.Since(startTime).Milliseconds()
			break
		}

		statusCode = captured.Status
		resHeaders = copyMap(captured.Headers)
		resHeaders["X-CassetteCraft"] = "RECORD_AUTO"
		resBody = captured.Body
		latencyMs = captured.LatencyMs

		inter := cassette.Interaction{
			ID:         fmt.Sprintf("int_%d", time.Now().UnixNano()),
			RecordedAt: time.Now().UTC(),
			Request: cassette.RequestData{
				Method:  req.Method,
				URL:     fullURL,
				Headers: reqHeadersMap,
				Body:    string(bodyBytes),
			},
			Response: cassette.ResponseData{
				Status:     captured.Status,
				StatusText: captured.StatusText,
				Headers:    captured.Headers,
				Body:       string(captured.Body),
				LatencyMs:  captured.LatencyMs,
			},
		}

		s.sanitizer.SanitizeInteraction(&inter)
		_ = s.storage.AppendInteraction(targetCassName, inter)
		matched = true

	case config.ModePassthrough:
		captured, err := ForwardRequest(req, bodyBytes, isTLS, s.cfg.InsecureSkipVerify)
		if err != nil {
			errDetails = fmt.Sprintf("Upstream forward error: %v", err)
			statusCode = http.StatusBadGateway
			resHeaders = map[string]string{"Content-Type": "application/json", "X-CassetteCraft-Status": "FORWARD_ERROR"}
			resBody = []byte(fmt.Sprintf(`{"error": %q}`, errDetails))
			latencyMs = time.Since(startTime).Milliseconds()
			break
		}

		statusCode = captured.Status
		resHeaders = copyMap(captured.Headers)
		resHeaders["X-CassetteCraft"] = "PASSTHROUGH"
		resBody = captured.Body
		latencyMs = captured.LatencyMs
	}

	// Push interaction event to live buffer for Studio
	s.buffer.Add(LiveInteraction{
		ID:         fmt.Sprintf("live_%d", time.Now().UnixNano()),
		Timestamp:  time.Now().UTC(),
		Mode:       string(activeMode),
		Status:     statusCode,
		Method:     req.Method,
		URL:        fullURL,
		Host:       req.Host,
		Path:       req.URL.Path,
		ReqHeaders: reqHeadersMap,
		ReqBody:    string(bodyBytes),
		ResHeaders: resHeaders,
		ResBody:    string(resBody),
		LatencyMs:  latencyMs,
		Matched:    matched,
		Cassette:   targetCassName,
		Error:      errDetails,
	})

	return s.formatHTTPResponse(statusCode, resHeaders, resBody), false
}

func (s *Server) buildFullURL(req *http.Request, isTLS bool) string {
	u := *req.URL
	if u.Scheme == "" {
		if isTLS {
			u.Scheme = "https"
		} else {
			u.Scheme = "http"
		}
	}
	if u.Host == "" {
		u.Host = req.Host
	}
	return u.String()
}

func (s *Server) formatHTTPResponse(status int, headers map[string]string, body []byte) []byte {
	var buf bytes.Buffer
	statusText := http.StatusText(status)
	if statusText == "" {
		statusText = "Custom"
	}
	fmt.Fprintf(&buf, "HTTP/1.1 %d %s\r\n", status, statusText)

	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Length"] = fmt.Sprintf("%d", len(body))
	if _, ok := headers["Connection"]; !ok {
		headers["Connection"] = "keep-alive"
	}

	for k, v := range headers {
		fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
	}
	buf.WriteString("\r\n")
	buf.Write(body)

	return buf.Bytes()
}

func extractHeaderMap(h http.Header) map[string]string {
	res := make(map[string]string)
	for k, vv := range h {
		res[k] = strings.Join(vv, ", ")
	}
	return res
}

func copyMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
