package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
)

// CapturedResponse holds the captured upstream response details.
type CapturedResponse struct {
	Status       int
	StatusText   string
	Headers      map[string]string
	Body         []byte
	LatencyMs    int64
	StreamChunks []cassette.StreamChunk
}

// ForwardRequest forwards an HTTP request to the real upstream service and captures the response.
func ForwardRequest(req *http.Request, bodyBytes []byte, clientTLS bool, insecureSkipVerify bool) (*CapturedResponse, error) {
	// Build target URL
	targetURL := *req.URL
	if targetURL.Scheme == "" {
		if clientTLS {
			targetURL.Scheme = "https"
		} else {
			targetURL.Scheme = "http"
		}
	}
	if targetURL.Host == "" {
		targetURL.Host = req.Host
	}

	upstreamReq, err := http.NewRequestWithContext(req.Context(), req.Method, targetURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create upstream request: %w", err)
	}

	// Copy headers excluding hop-by-hop headers
	copyHeaders(req.Header, upstreamReq.Header)
	upstreamReq.Header.Del("Proxy-Connection")
	upstreamReq.Header.Del("Proxy-Authorization")

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	startTime := time.Now()
	resp, err := transport.RoundTrip(upstreamReq)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	latency := time.Since(startTime).Milliseconds()

	var respBody []byte
	var streamChunks []cassette.StreamChunk
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	if strings.Contains(contentType, "text/event-stream") || strings.Contains(contentType, "ndjson") {
		reader := bufio.NewReader(resp.Body)
		var buf bytes.Buffer
		streamStart := time.Now()

		for {
			chunk, err := reader.ReadBytes('\n')
			if len(chunk) > 0 {
				buf.Write(chunk)
				offset := time.Since(streamStart).Milliseconds()
				streamChunks = append(streamChunks, cassette.StreamChunk{
					OffsetMs: offset,
					Data:     string(chunk),
				})
			}
			if err != nil {
				break
			}
		}
		respBody = buf.Bytes()
	} else {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read upstream response body: %w", err)
		}
		respBody = bodyBytes
	}

	respHeaders := make(map[string]string)
	for k, vals := range resp.Header {
		if !isHopByHopHeader(k) {
			respHeaders[k] = strings.Join(vals, ", ")
		}
	}

	return &CapturedResponse{
		Status:       resp.StatusCode,
		StatusText:   http.StatusText(resp.StatusCode),
		Headers:      respHeaders,
		Body:         respBody,
		LatencyMs:    latency,
		StreamChunks: streamChunks,
	}, nil
}

func copyHeaders(src, dst http.Header) {
	for k, vv := range src {
		if !isHopByHopHeader(k) {
			for _, v := range vv {
				dst.Add(k, v)
			}
		}
	}
}

func isHopByHopHeader(header string) bool {
	switch strings.ToLower(header) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"te", "trailers", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}
