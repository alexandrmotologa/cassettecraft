package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Mode defines the operating proxy mode.
type Mode string

const (
	ModeRecord      Mode = "record"
	ModeReplay      Mode = "replay"
	ModeAuto        Mode = "auto"
	ModePassthrough Mode = "passthrough"
)

// IsValid checks if the mode is supported.
func (m Mode) IsValid() bool {
	switch m {
	case ModeRecord, ModeReplay, ModeAuto, ModePassthrough:
		return true
	default:
		return false
	}
}

// RedactionPattern defines a custom regex redaction rule.
type RedactionPattern struct {
	Name        string `yaml:"name" json:"name"`
	Regex       string `yaml:"regex" json:"regex"`
	Replacement string `yaml:"replacement" json:"replacement"`
}

// RedactionConfig holds custom redaction rules.
type RedactionConfig struct {
	Headers     []string           `yaml:"headers" json:"headers"`
	QueryParams []string           `yaml:"query_params" json:"query_params"`
	JSONKeys    []string           `yaml:"json_keys" json:"json_keys"`
	Patterns    []RedactionPattern `yaml:"patterns" json:"patterns"`
}

// ChaosConfig holds settings for artificial latency and error simulation.
type ChaosConfig struct {
	Latency       time.Duration `yaml:"latency" json:"latency"`
	Jitter        time.Duration `yaml:"jitter" json:"jitter"`
	ErrorRate     float64       `yaml:"error_rate" json:"error_rate"` // 0.0 to 1.0
	ErrorStatus   int           `yaml:"error_status" json:"error_status"`
	RateLimitFreq int           `yaml:"rate_limit_freq" json:"rate_limit_freq"` // e.g., fail every Nth request with 429
}

// MatchingConfig defines request matching rules.
type MatchingConfig struct {
	Strategy      string   `yaml:"strategy" json:"strategy"` // exact, standard, method_url
	IgnoreHeaders []string `yaml:"ignore_headers" json:"ignore_headers"`
	IgnoreQueries []string `yaml:"ignore_queries" json:"ignore_queries"`
	Sequential    bool     `yaml:"sequential" json:"sequential"`
}

// Config represents the complete cassettecraft configuration.
type Config struct {
	Port         int             `yaml:"port" json:"port"`
	Mode         Mode            `yaml:"mode" json:"mode"`
	ActiveCass   string          `yaml:"cassette" json:"cassette"`
	CassetteDir  string          `yaml:"cassette_dir" json:"cassette_dir"`
	CertDir            string          `yaml:"cert_dir" json:"cert_dir"`
	InsecureSkipVerify bool            `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	EnableStudio       bool            `yaml:"enable_studio" json:"enable_studio"`
	Verbose      bool            `yaml:"verbose" json:"verbose"`
	Redaction    RedactionConfig `yaml:"redaction" json:"redaction"`
	Chaos        ChaosConfig     `yaml:"chaos" json:"chaos"`
	Matching     MatchingConfig  `yaml:"matching" json:"matching"`
}

// DefaultConfig returns reasonable default configuration values.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	certDir := filepath.Join(homeDir, ".cassettecraft", "ca")

	return &Config{
		Port:         8585,
		Mode:         ModeRecord,
		ActiveCass:   "default.yaml",
		CassetteDir:  "./cassettes",
		CertDir:      certDir,
		EnableStudio: true,
		Verbose:      false,
		Redaction: RedactionConfig{
			Headers: []string{
				"Authorization",
				"Proxy-Authorization",
				"Cookie",
				"Set-Cookie",
				"X-Api-Key",
				"Stripe-Signature",
				"X-Hub-Signature-256",
			},
			QueryParams: []string{
				"api_key",
				"apikey",
				"token",
				"access_token",
				"secret",
				"signature",
				"auth",
			},
			JSONKeys: []string{
				"password",
				"passphrase",
				"client_secret",
				"clientSecret",
				"private_key",
				"privateKey",
				"token",
				"access_token",
				"accessToken",
				"refresh_token",
				"refreshToken",
				"cvv",
				"cvc",
				"pin",
				"card_number",
				"cardNumber",
			},
			Patterns: []RedactionPattern{},
		},
		Chaos: ChaosConfig{
			Latency:     0,
			Jitter:      0,
			ErrorRate:   0,
			ErrorStatus: 500,
		},
		Matching: MatchingConfig{
			Strategy: "standard",
			IgnoreHeaders: []string{
				"Date",
				"User-Agent",
				"X-Request-Id",
				"Traceparent",
				"Accept-Encoding",
				"Connection",
				"Host",
				"Content-Length",
				"X-Amzn-Trace-Id",
			},
			IgnoreQueries: []string{
				"_t",
				"timestamp",
				"nonce",
			},
			Sequential: false,
		},
	}
}

// LoadConfigFromFile loads and merges configuration from a YAML file.
func LoadConfigFromFile(path string) (*Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		// Look for cassettecraft.yaml in current directory
		if _, err := os.Stat("cassettecraft.yaml"); err == nil {
			path = "cassettecraft.yaml"
		} else if _, err := os.Stat("cassettecraft.yml"); err == nil {
			path = "cassettecraft.yml"
		} else {
			return cfg, nil
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	if !cfg.Mode.IsValid() {
		return nil, fmt.Errorf("invalid proxy mode %q in config", cfg.Mode)
	}

	return cfg, nil
}
