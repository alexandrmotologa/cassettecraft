package chaos

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/config"
)

// Chaos manages synthetic latency and error injection during replay and proxying.
type Chaos struct {
	mu           sync.RWMutex
	cfg          config.ChaosConfig
	requestCount uint64
}

// NewChaos creates a chaos simulator instance.
func NewChaos(cfg config.ChaosConfig) *Chaos {
	return &Chaos{
		cfg: cfg,
	}
}

// UpdateConfig updates the chaos settings in real time.
func (c *Chaos) UpdateConfig(cfg config.ChaosConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfg = cfg
}

// GetConfig returns the active chaos settings.
func (c *Chaos) GetConfig() config.ChaosConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg
}

// ApplyLatency sleeps for the configured latency plus optional random jitter.
func (c *Chaos) ApplyLatency() time.Duration {
	c.mu.RLock()
	latency := c.cfg.Latency
	jitter := c.cfg.Jitter
	c.mu.RUnlock()

	if latency <= 0 && jitter <= 0 {
		return 0
	}

	total := latency
	if jitter > 0 {
		n, err := rand.Int(rand.Reader, big.NewInt(jitter.Nanoseconds()))
		if err == nil {
			total += time.Duration(n.Int64())
		}
	}

	if total > 0 {
		time.Sleep(total)
	}
	return total
}

// ShouldSimulateError checks if an error or rate limit should be triggered for the current request.
// Returns (shouldError, statusCode, statusText).
func (c *Chaos) ShouldSimulateError() (bool, int, string) {
	c.mu.RLock()
	cfg := c.cfg
	c.mu.RUnlock()

	count := atomic.AddUint64(&c.requestCount, 1)

	// Rate limit frequency check (e.g., fail every N requests with 429)
	if cfg.RateLimitFreq > 0 && (count%uint64(cfg.RateLimitFreq) == 0) {
		return true, http.StatusTooManyRequests, "429 Too Many Requests (Synthetic Chaos)"
	}

	// Error rate percentage check (0.0 to 1.0)
	if cfg.ErrorRate > 0 {
		n, err := rand.Int(rand.Reader, big.NewInt(10000))
		if err == nil {
			threshold := int64(cfg.ErrorRate * 10000)
			if n.Int64() < threshold {
				status := cfg.ErrorStatus
				if status == 0 {
					status = http.StatusInternalServerError
				}
				return true, status, http.StatusText(status) + " (Synthetic Chaos)"
			}
		}
	}

	return false, 0, ""
}
