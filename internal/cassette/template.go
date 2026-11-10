package cassette

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var templateRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.\-]+)\s*\}\}`)

// GenerateUUID generates a random RFC 4122 UUID v4.
func GenerateUUID() string {
	var uuid [16]byte
	_, _ = rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant RFC 4122
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// ApplyResponseTemplates resolves dynamic placeholders inside response bodies and headers.
func ApplyResponseTemplates(resp ResponseData, req RequestData, rawURL string, reqHeaders map[string]string) ResponseData {
	if !strings.Contains(resp.Body, "{{") && !headersContainTemplate(resp.Headers) {
		return resp
	}

	// Parse request URL and query params
	var queryParams url.Values
	if parsedURL, err := url.Parse(rawURL); err == nil {
		queryParams = parsedURL.Query()
	}

	// Parse request body as JSON map if possible
	var parsedBody map[string]interface{}
	if strings.TrimSpace(req.Body) != "" {
		_ = json.Unmarshal([]byte(req.Body), &parsedBody)
	}

	replacer := func(key string) string {
		switch strings.ToLower(key) {
		case "now", "now_utc":
			return time.Now().UTC().Format(time.RFC3339)
		case "timestamp":
			return strconv.FormatInt(time.Now().Unix(), 10)
		case "timestamp_ms":
			return strconv.FormatInt(time.Now().UnixMilli(), 10)
		case "uuid":
			return GenerateUUID()
		}

		if strings.HasPrefix(key, "request.query.") {
			param := strings.TrimPrefix(key, "request.query.")
			return queryParams.Get(param)
		}

		if strings.HasPrefix(key, "request.header.") {
			headerName := strings.TrimPrefix(key, "request.header.")
			for k, v := range reqHeaders {
				if strings.EqualFold(k, headerName) {
					return v
				}
			}
			return ""
		}

		if strings.HasPrefix(key, "request.body.") {
			path := strings.TrimPrefix(key, "request.body.")
			if val, ok := extractNestedJSONValue(parsedBody, path); ok {
				return fmt.Sprintf("%v", val)
			}
			return ""
		}

		return "{{" + key + "}}"
	}

	// Replace in response body
	newBody := templateRegex.ReplaceAllStringFunc(resp.Body, func(match string) string {
		submatches := templateRegex.FindStringSubmatch(match)
		if len(submatches) > 1 {
			return replacer(submatches[1])
		}
		return match
	})

	// Replace in response headers
	newHeaders := make(map[string]string, len(resp.Headers))
	for k, v := range resp.Headers {
		newHeaders[k] = templateRegex.ReplaceAllStringFunc(v, func(match string) string {
			submatches := templateRegex.FindStringSubmatch(match)
			if len(submatches) > 1 {
				return replacer(submatches[1])
			}
			return match
		})
	}

	resp.Body = newBody
	resp.Headers = newHeaders
	return resp
}

func headersContainTemplate(headers map[string]string) bool {
	for _, v := range headers {
		if strings.Contains(v, "{{") {
			return true
		}
	}
	return false
}

func extractNestedJSONValue(data map[string]interface{}, path string) (interface{}, bool) {
	if data == nil {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}
	return current, true
}
