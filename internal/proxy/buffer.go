package proxy

import (
	"sync"
	"time"
)

// LiveInteraction represents an intercepted interaction event for the UI waterfall.
type LiveInteraction struct {
	ID         string            `json:"id"`
	Timestamp  time.Time         `json:"timestamp"`
	Mode       string            `json:"mode"`
	Status     int               `json:"status"`
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Host       string            `json:"host"`
	Path       string            `json:"path"`
	ReqHeaders map[string]string `json:"req_headers"`
	ReqBody    string            `json:"req_body"`
	ResHeaders map[string]string `json:"res_headers"`
	ResBody    string            `json:"res_body"`
	LatencyMs  int64             `json:"latency_ms"`
	Matched    bool              `json:"matched"`
	Cassette   string            `json:"cassette"`
	Error      string            `json:"error,omitempty"`
}

// InteractionBuffer is a thread-safe circular ring buffer for live interactions.
type InteractionBuffer struct {
	mu       sync.RWMutex
	capacity int
	items    []LiveInteraction
}

// NewInteractionBuffer creates an interaction ring buffer.
func NewInteractionBuffer(capacity int) *InteractionBuffer {
	if capacity <= 0 {
		capacity = 500
	}
	return &InteractionBuffer{
		capacity: capacity,
		items:    make([]LiveInteraction, 0, capacity),
	}
}

// Add pushes a new interaction event to the buffer.
func (b *InteractionBuffer) Add(item LiveInteraction) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) >= b.capacity {
		// Evict oldest item
		b.items = append(b.items[1:], item)
	} else {
		b.items = append(b.items, item)
	}
}

// GetAll returns a copy of all interactions in chronological order (newest last).
func (b *InteractionBuffer) GetAll() []LiveInteraction {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]LiveInteraction, len(b.items))
	copy(result, b.items)
	return result
}

// Clear flushes all items from the ring buffer.
func (b *InteractionBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.items = make([]LiveInteraction, 0, b.capacity)
}
