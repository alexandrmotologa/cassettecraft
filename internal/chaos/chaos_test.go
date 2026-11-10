package chaos_test

import (
	"testing"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/chaos"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
)

func TestChaosLatency(t *testing.T) {
	cfg := config.ChaosConfig{
		Latency: 50 * time.Millisecond,
		Jitter:  10 * time.Millisecond,
	}

	sim := chaos.NewChaos(cfg)
	start := time.Now()
	applied := sim.ApplyLatency()
	elapsed := time.Since(start)

	if applied < 50*time.Millisecond {
		t.Fatalf("expected applied latency >= 50ms, got %v", applied)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("expected elapsed time >= 50ms, got %v", elapsed)
	}
}

func TestChaosRateLimiting(t *testing.T) {
	cfg := config.ChaosConfig{
		RateLimitFreq: 3, // Fail every 3rd request
	}

	sim := chaos.NewChaos(cfg)

	// Call 1: pass
	err1, _, _ := sim.ShouldSimulateError()
	if err1 {
		t.Fatal("call 1 should not error")
	}

	// Call 2: pass
	err2, _, _ := sim.ShouldSimulateError()
	if err2 {
		t.Fatal("call 2 should not error")
	}

	// Call 3: fail with 429
	err3, code3, _ := sim.ShouldSimulateError()
	if !err3 || code3 != 429 {
		t.Fatalf("call 3 should fail with 429, got %v (%d)", err3, code3)
	}

	// Call 4: pass
	err4, _, _ := sim.ShouldSimulateError()
	if err4 {
		t.Fatal("call 4 should not error")
	}
}

func TestChaosErrorRate(t *testing.T) {
	cfg := config.ChaosConfig{
		ErrorRate:   1.0, // 100% fail
		ErrorStatus: 503,
	}

	sim := chaos.NewChaos(cfg)
	shouldErr, status, _ := sim.ShouldSimulateError()
	if !shouldErr || status != 503 {
		t.Fatalf("expected 100%% fail with 503, got %v (%d)", shouldErr, status)
	}
}
