package har

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
)

// HAR represents the root HTTP Archive structure.
type HAR struct {
	Log Log `json:"log"`
}

// Log represents the log container in a HAR file.
type Log struct {
	Version string  `json:"version"`
	Creator Creator `json:"creator"`
	Entries []Entry `json:"entries"`
}

// Creator holds metadata about the HAR generator.
type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Entry represents a single HTTP transaction in HAR.
type Entry struct {
	StartedDateTime time.Time `json:"startedDateTime"`
	Time            float64   `json:"time"` // Total elapsed time in milliseconds
	Request         Request   `json:"request"`
	Response        Response  `json:"response"`
}

// Header represents a key-value header in HAR.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PostData represents request body payload.
type PostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// Request represents the HTTP request in HAR.
type Request struct {
	Method      string   `json:"method"`
	URL         string   `json:"url"`
	HTTPVersion string   `json:"httpVersion"`
	Headers     []Header `json:"headers"`
	PostData    PostData `json:"postData"`
}

// Content represents the HTTP response payload in HAR.
type Content struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// Response represents the HTTP response in HAR.
type Response struct {
	Status      int      `json:"status"`
	StatusText  string   `json:"statusText"`
	HTTPVersion string   `json:"httpVersion"`
	Headers     []Header `json:"headers"`
	Content     Content  `json:"content"`
}

// ConvertHARToCassette converts a HAR JSON file into a sanitized Cassette.
func ConvertHARToCassette(harData []byte, sanitizer *cassette.Sanitizer) (*cassette.Cassette, error) {
	var har HAR
	if err := json.Unmarshal(harData, &har); err != nil {
		return nil, fmt.Errorf("failed to parse HAR JSON: %w", err)
	}

	cass := cassette.NewCassette()
	cass.RecordedBy = fmt.Sprintf("cassettecraft/har-import (%s)", har.Log.Creator.Name)

	for i, entry := range har.Log.Entries {
		reqHeaders := make(map[string]string)
		for _, h := range entry.Request.Headers {
			reqHeaders[h.Name] = h.Value
		}

		resHeaders := make(map[string]string)
		for _, h := range entry.Response.Headers {
			resHeaders[h.Name] = h.Value
		}

		inter := cassette.Interaction{
			ID:         fmt.Sprintf("har_int_%04d", i+1),
			RecordedAt: entry.StartedDateTime,
			Request: cassette.RequestData{
				Method:  entry.Request.Method,
				URL:     entry.Request.URL,
				Headers: reqHeaders,
				Body:    entry.Request.PostData.Text,
			},
			Response: cassette.ResponseData{
				Status:     entry.Response.Status,
				StatusText: entry.Response.StatusText,
				Headers:    resHeaders,
				Body:       entry.Response.Content.Text,
				LatencyMs:  int64(entry.Time),
			},
		}

		if sanitizer != nil {
			sanitizer.SanitizeInteraction(&inter)
		}

		cass.Interactions = append(cass.Interactions, inter)
	}

	return cass, nil
}

// ImportHARFile reads a .har file from disk, sanitizes it, and saves it as a YAML cassette.
func ImportHARFile(harPath, outPath string) error {
	data, err := os.ReadFile(harPath)
	if err != nil {
		return fmt.Errorf("failed to read HAR file %q: %w", harPath, err)
	}

	sanitizer := cassette.NewSanitizer(config.DefaultConfig().Redaction)
	cass, err := ConvertHARToCassette(data, sanitizer)
	if err != nil {
		return err
	}

	yamlBytes, err := cass.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to format YAML: %w", err)
	}

	if err := os.WriteFile(outPath, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write cassette file %q: %w", outPath, err)
	}

	return nil
}

// GenerateCurl returns an equivalent executable cURL command for an interaction request.
func GenerateCurl(req cassette.RequestData) string {
	var parts []string
	parts = append(parts, "curl", "-X", req.Method)
	parts = append(parts, fmt.Sprintf("%q", req.URL))

	for k, v := range req.Headers {
		parts = append(parts, "-H", fmt.Sprintf("%q", fmt.Sprintf("%s: %s", k, v)))
	}

	if req.Body != "" {
		// Escape any double quotes in body
		escapedBody := strings.ReplaceAll(req.Body, `"`, `\"`)
		parts = append(parts, "-d", fmt.Sprintf("%q", escapedBody))
	}

	return strings.Join(parts, " ")
}
