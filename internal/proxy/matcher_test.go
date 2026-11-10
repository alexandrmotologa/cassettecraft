package proxy_test

import (
	"testing"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
	"github.com/alexandrmotologa/cassettecraft/internal/proxy"
)

func TestMatcherStandardStrategy(t *testing.T) {
	matcher := proxy.NewMatcher(config.DefaultConfig().Matching)

	cass := cassette.NewCassette()
	cass.Interactions = append(cass.Interactions, cassette.Interaction{
		ID:         "test_int_1",
		RecordedAt: time.Now(),
		Request: cassette.RequestData{
			Method: "POST",
			URL:    "https://api.stripe.com/v1/customers?limit=10&page=1",
			Body:   `{"name":"Alice","tier":"gold"}`,
		},
		Response: cassette.ResponseData{
			Status: 200,
			Body:   `{"id":"cus_1"}`,
		},
	})

	// Match 1: Same method, URL with reversed query param order, identical semantic JSON body with different whitespace
	res := matcher.FindMatch(
		cass,
		"POST",
		"https://api.stripe.com/v1/customers?page=1&limit=10",
		nil,
		[]byte("{\n  \"tier\": \"gold\",\n  \"name\": \"Alice\"\n}"),
	)

	if res.Matched == nil {
		t.Fatalf("expected match, got diagnostic: %v", res.Diagnostics.Reasons)
	}
	if res.Matched.ID != "test_int_1" {
		t.Fatalf("expected ID test_int_1, got %s", res.Matched.ID)
	}

	// Match 2: Method mismatch
	resDiffMethod := matcher.FindMatch(
		cass,
		"GET",
		"https://api.stripe.com/v1/customers",
		nil,
		nil,
	)
	if resDiffMethod.Matched != nil {
		t.Fatal("expected no match on method difference")
	}

	// Match 3: Query mismatch
	resDiffQuery := matcher.FindMatch(
		cass,
		"POST",
		"https://api.stripe.com/v1/customers?page=2&limit=10",
		nil,
		[]byte(`{"name":"Alice","tier":"gold"}`),
	)
	if resDiffQuery.Matched != nil {
		t.Fatal("expected no match on query difference")
	}
}

func TestMatcherSequentialMode(t *testing.T) {
	cfg := config.DefaultConfig().Matching
	cfg.Sequential = true
	matcher := proxy.NewMatcher(cfg)

	cass := cassette.NewCassette()
	// Endpoint called 3 times with different statuses (e.g. polling)
	cass.Interactions = append(cass.Interactions,
		cassette.Interaction{
			ID: "poll_1",
			Request: cassette.RequestData{
				Method: "GET",
				URL:    "https://api.example.com/v1/job/123",
			},
			Response: cassette.ResponseData{Status: 202, Body: `{"status":"pending"}`},
		},
		cassette.Interaction{
			ID: "poll_2",
			Request: cassette.RequestData{
				Method: "GET",
				URL:    "https://api.example.com/v1/job/123",
			},
			Response: cassette.ResponseData{Status: 202, Body: `{"status":"running"}`},
		},
		cassette.Interaction{
			ID: "poll_3",
			Request: cassette.RequestData{
				Method: "GET",
				URL:    "https://api.example.com/v1/job/123",
			},
			Response: cassette.ResponseData{Status: 200, Body: `{"status":"succeeded"}`},
		},
	)

	// Call 1
	r1 := matcher.FindMatch(cass, "GET", "https://api.example.com/v1/job/123", nil, nil)
	if r1.Matched == nil || r1.Matched.ID != "poll_1" {
		t.Fatalf("expected poll_1, got %v", r1.Matched)
	}

	// Call 2
	r2 := matcher.FindMatch(cass, "GET", "https://api.example.com/v1/job/123", nil, nil)
	if r2.Matched == nil || r2.Matched.ID != "poll_2" {
		t.Fatalf("expected poll_2, got %v", r2.Matched)
	}

	// Call 3
	r3 := matcher.FindMatch(cass, "GET", "https://api.example.com/v1/job/123", nil, nil)
	if r3.Matched == nil || r3.Matched.ID != "poll_3" {
		t.Fatalf("expected poll_3, got %v", r3.Matched)
	}
}

func TestInteractionBuffer(t *testing.T) {
	buf := proxy.NewInteractionBuffer(3)

	buf.Add(proxy.LiveInteraction{ID: "1"})
	buf.Add(proxy.LiveInteraction{ID: "2"})
	buf.Add(proxy.LiveInteraction{ID: "3"})

	items := buf.GetAll()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Adding 4th item should evict 1st item
	buf.Add(proxy.LiveInteraction{ID: "4"})
	itemsAfter := buf.GetAll()
	if len(itemsAfter) != 3 {
		t.Fatalf("expected 3 items, got %d", len(itemsAfter))
	}
	if itemsAfter[0].ID != "2" || itemsAfter[2].ID != "4" {
		t.Fatalf("expected items 2,3,4, got %s, ..., %s", itemsAfter[0].ID, itemsAfter[2].ID)
	}

	buf.Clear()
	if len(buf.GetAll()) != 0 {
		t.Fatal("expected empty buffer after clear")
	}
}
