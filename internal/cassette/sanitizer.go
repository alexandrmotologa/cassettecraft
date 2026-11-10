package cassette

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/cassettecraft/internal/config"
)

type compiledPattern struct {
	re          *regexp.Regexp
	replacement string
}

// Sanitizer manages secret scrubbing across requests and responses.
type Sanitizer struct {
	cfg              config.RedactionConfig
	patterns         []compiledPattern
	redactedHeaders  map[string]bool
	redactedQueries  map[string]bool
	redactedJSONKeys map[string]bool
}

// NewSanitizer initializes a sanitizer with default and custom redaction rules.
func NewSanitizer(cfg config.RedactionConfig) *Sanitizer {
	s := &Sanitizer{
		cfg:              cfg,
		redactedHeaders:  make(map[string]bool),
		redactedQueries:  make(map[string]bool),
		redactedJSONKeys: make(map[string]bool),
	}

	for _, h := range cfg.Headers {
		s.redactedHeaders[strings.ToLower(h)] = true
	}
	for _, q := range cfg.QueryParams {
		s.redactedQueries[strings.ToLower(q)] = true
	}
	for _, k := range cfg.JSONKeys {
		s.redactedJSONKeys[strings.ToLower(k)] = true
	}

	// Built-in standard regex patterns
	defaultPatterns := []struct {
		pattern     string
		replacement string
	}{
		// Stripe secret & restricted keys
		{`\b(sk|rk)_(live|test)_[0-9a-zA-Z]{20,}\b`, `[REDACTED_STRIPE_KEY]`},
		// GitHub tokens (PAT, OAuth, fine-grained)
		{`\b(ghp|gho|ghu|ghs|ghr)_[0-9a-zA-Z]{30,40}\b`, `[REDACTED_GITHUB_TOKEN]`},
		{`\bgithub_pat_[0-9a-zA-Z_]{60,}\b`, `[REDACTED_GITHUB_PAT]`},
		// OpenAI API keys
		{`\bsk-[a-zA-Z0-9]{20,}\b`, `[REDACTED_OPENAI_KEY]`},
		// AWS Access Key ID
		{`\b(AKIA|ASIA)[0-9A-Z]{16}\b`, `[REDACTED_AWS_KEY]`},
		// Slack bot and user tokens
		{`\bxox[baprs]-[0-9a-zA-Z-]{10,}\b`, `[REDACTED_SLACK_TOKEN]`},
		// JSON Web Tokens (three base64 segments)
		{`\bey[A-Za-z0-9_-]{10,}\.ey[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`, `[REDACTED_JWT]`},
		// Private Key PEM blocks
		{`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]+?-----END [A-Z ]*PRIVATE KEY-----`, `[REDACTED_PRIVATE_KEY]`},
		// Credit card 13-19 digit sequences (hyphenated or plain)
		{`\b(?:\d{4}[ -]?){3}\d{4}\b`, `[REDACTED_CARD_NUMBER]`},
	}

	for _, p := range defaultPatterns {
		if re, err := regexp.Compile(p.pattern); err == nil {
			s.patterns = append(s.patterns, compiledPattern{re: re, replacement: p.replacement})
		}
	}

	// Compile user-configured custom patterns
	for _, p := range cfg.Patterns {
		if re, err := regexp.Compile(p.Regex); err == nil {
			replacement := p.Replacement
			if replacement == "" {
				replacement = "[REDACTED]"
			}
			s.patterns = append(s.patterns, compiledPattern{re: re, replacement: replacement})
		}
	}

	return s
}

// SanitizeHeaders scrubs sensitive header values.
func (s *Sanitizer) SanitizeHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	clean := make(map[string]string, len(headers))
	for k, v := range headers {
		lowerKey := strings.ToLower(k)
		if s.redactedHeaders[lowerKey] {
			if strings.EqualFold(k, "Authorization") {
				if strings.HasPrefix(strings.ToLower(v), "bearer ") {
					clean[k] = "Bearer [REDACTED_TOKEN]"
				} else if strings.HasPrefix(strings.ToLower(v), "basic ") {
					clean[k] = "Basic [REDACTED_CREDENTIALS]"
				} else {
					clean[k] = "[REDACTED_AUTH]"
				}
			} else {
				clean[k] = "[REDACTED_" + strings.ToUpper(strings.ReplaceAll(k, "-", "_")) + "]"
			}
		} else {
			// Also scrub known tokens inside other headers
			clean[k] = s.SanitizeString(v)
		}
	}
	return clean
}

// SanitizeURL scrubs sensitive query parameter values in a URL string.
func (s *Sanitizer) SanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return s.SanitizeString(rawURL)
	}

	if u.RawQuery != "" {
		values := u.Query()
		modified := false
		for k := range values {
			if s.redactedQueries[strings.ToLower(k)] {
				values.Set(k, "[REDACTED]")
				modified = true
			} else {
				// Sanitize values against regex patterns
				for i, val := range values[k] {
					newVal := s.SanitizeString(val)
					if newVal != val {
						values[k][i] = newVal
						modified = true
					}
				}
			}
		}
		if modified {
			u.RawQuery = values.Encode()
		}
	}

	return u.String()
}

// SanitizeBody scrubs sensitive content in a request or response body.
func (s *Sanitizer) SanitizeBody(body string) string {
	if strings.TrimSpace(body) == "" {
		return body
	}

	// Try JSON tree scrubbing first if valid JSON
	var parsed interface{}
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err == nil {
		sanitizedParsed := s.sanitizeJSONTree(parsed)
		if formatted, err := json.Marshal(sanitizedParsed); err == nil {
			// Apply regex patterns across the serialized JSON to catch unkeyed secrets
			return s.SanitizeString(string(formatted))
		}
	}

	// Fallback to text string regex replacement
	return s.SanitizeString(body)
}

// SanitizeString applies compiled regex patterns on an arbitrary string.
func (s *Sanitizer) SanitizeString(input string) string {
	result := input
	for _, p := range s.patterns {
		result = p.re.ReplaceAllString(result, p.replacement)
	}
	return result
}

// SanitizeInteraction performs full scrubbing on an interaction pair.
func (s *Sanitizer) SanitizeInteraction(interaction *Interaction) {
	interaction.Request.URL = s.SanitizeURL(interaction.Request.URL)
	interaction.Request.Headers = s.SanitizeHeaders(interaction.Request.Headers)
	interaction.Request.Body = s.SanitizeBody(interaction.Request.Body)
	interaction.Request.Body = NormalizeJSONBody(interaction.Request.Body)
	interaction.Request.BodyHash = ComputeHash([]byte(interaction.Request.Body))

	interaction.Response.Headers = s.SanitizeHeaders(interaction.Response.Headers)
	interaction.Response.Body = s.SanitizeBody(interaction.Response.Body)
	interaction.Response.Body = NormalizeJSONBody(interaction.Response.Body)
	interaction.Response.BodyHash = ComputeHash([]byte(interaction.Response.Body))
}

func (s *Sanitizer) sanitizeJSONTree(node interface{}) interface{} {
	switch v := node.(type) {
	case map[string]interface{}:
		cleanMap := make(map[string]interface{}, len(v))
		for k, val := range v {
			if s.isSensitiveKey(k) {
				cleanMap[k] = "[REDACTED]"
			} else {
				cleanMap[k] = s.sanitizeJSONTree(val)
			}
		}
		return cleanMap
	case []interface{}:
		cleanSlice := make([]interface{}, len(v))
		for i, item := range v {
			cleanSlice[i] = s.sanitizeJSONTree(item)
		}
		return cleanSlice
	case string:
		return s.SanitizeString(v)
	default:
		return node
	}
}

func (s *Sanitizer) isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	if s.redactedJSONKeys[lower] {
		return true
	}
	// Check common substrings
	for target := range s.redactedJSONKeys {
		if strings.Contains(lower, target) {
			return true
		}
	}
	return false
}
