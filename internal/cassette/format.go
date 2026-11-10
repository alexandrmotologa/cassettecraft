package cassette

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// CurrentVersion is the current cassette schema version.
const CurrentVersion = 1

// RequestData represents the recorded HTTP request.
type RequestData struct {
	Method   string            `yaml:"method" json:"method"`
	URL      string            `yaml:"url" json:"url"`
	Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body     string            `yaml:"body,omitempty" json:"body,omitempty"`
	BodyHash string            `yaml:"body_hash,omitempty" json:"body_hash,omitempty"`
}

// StreamChunk represents a single chunk or event in a streaming response (e.g. SSE or chunked transfer).
type StreamChunk struct {
	OffsetMs int64  `yaml:"offset_ms" json:"offset_ms"`
	Data     string `yaml:"data" json:"data"`
}

// ResponseData represents the recorded HTTP response.
type ResponseData struct {
	Status       int               `yaml:"status" json:"status"`
	StatusText   string            `yaml:"status_text,omitempty" json:"status_text,omitempty"`
	Headers      map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body         string            `yaml:"body,omitempty" json:"body,omitempty"`
	BodyHash     string            `yaml:"body_hash,omitempty" json:"body_hash,omitempty"`
	LatencyMs    int64             `yaml:"latency_ms,omitempty" json:"latency_ms,omitempty"`
	StreamChunks []StreamChunk     `yaml:"stream_chunks,omitempty" json:"stream_chunks,omitempty"`
}

// Interaction represents a single request and response cycle.
type Interaction struct {
	ID         string       `yaml:"id" json:"id"`
	RecordedAt time.Time    `yaml:"recorded_at" json:"recorded_at"`
	Request    RequestData  `yaml:"request" json:"request"`
	Response   ResponseData `yaml:"response" json:"response"`
}

// Cassette represents a complete recorded file containing interactions and metadata.
type Cassette struct {
	Version      int           `yaml:"version" json:"version"`
	RecordedAt   time.Time     `yaml:"recorded_at" json:"recorded_at"`
	RecordedBy   string        `yaml:"recorded_by" json:"recorded_by"`
	Interactions []Interaction `yaml:"interactions" json:"interactions"`
}

// NewCassette initializes an empty cassette with default metadata.
func NewCassette() *Cassette {
	return &Cassette{
		Version:      CurrentVersion,
		RecordedAt:   time.Now().UTC(),
		RecordedBy:   "cassettecraft/v1.0",
		Interactions: make([]Interaction, 0),
	}
}

// ComputeHash computes SHA-256 hex digest of a byte slice.
func ComputeHash(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// NormalizeJSONBody formats a JSON string deterministically with 2-space indentation.
func NormalizeJSONBody(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}
	if !((strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"))) {
		return body
	}

	var parsed interface{}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return body
	}

	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return body
	}
	return string(formatted)
}

// SortHeaders returns a new map with normalized header casing and keys.
func SortHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	sorted := make(map[string]string, len(headers))
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sorted[k] = headers[k]
	}
	return sorted
}

// ToYAML serializes the cassette deterministically to YAML.
func (c *Cassette) ToYAML() ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(c); err != nil {
		return nil, fmt.Errorf("failed to encode cassette to YAML: %w", err)
	}
	return buf.Bytes(), nil
}

// FromYAML parses a cassette from YAML bytes.
func FromYAML(data []byte) (*Cassette, error) {
	cass := &Cassette{}
	if err := yaml.Unmarshal(data, cass); err != nil {
		return nil, fmt.Errorf("failed to parse cassette from YAML: %w", err)
	}
	if cass.Version == 0 {
		cass.Version = CurrentVersion
	}
	return cass, nil
}

// ToJSON serializes the cassette to formatted JSON.
func (c *Cassette) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// FromJSON parses a cassette from JSON bytes.
func FromJSON(data []byte) (*Cassette, error) {
	cass := &Cassette{}
	if err := json.Unmarshal(data, cass); err != nil {
		return nil, fmt.Errorf("failed to parse cassette from JSON: %w", err)
	}
	if cass.Version == 0 {
		cass.Version = CurrentVersion
	}
	return cass, nil
}
