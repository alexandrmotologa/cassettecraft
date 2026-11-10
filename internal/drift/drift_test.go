package drift_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/drift"
)

func TestCheckCassetteDrift(t *testing.T) {
	// Mock live upstream server that has added a new field and changed status code
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "cus_123",
			"tier": "enterprise",
			"new_feature_flag": true
		}`))
	}))
	defer server.Close()

	cass := cassette.NewCassette()
	cass.Interactions = append(cass.Interactions, cassette.Interaction{
		ID: "drift_int_1",
		Request: cassette.RequestData{
			Method: "GET",
			URL:    server.URL + "/customers/123",
		},
		Response: cassette.ResponseData{
			Status: 200,
			Body: `{
				"id": "cus_123",
				"tier": "enterprise",
				"old_deprecated_field": "legacy"
			}`,
		},
	})

	report, err := drift.CheckCassetteDrift(cass, "test.yaml", server.Client())
	if err != nil {
		t.Fatalf("failed to check drift: %v", err)
	}

	if !report.HasDrift {
		t.Fatal("expected drift to be detected")
	}

	if len(report.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(report.Items))
	}

	diffs := report.Items[0].Differences
	hasMissing := false
	hasNew := false

	for _, d := range diffs {
		if d == `Key missing in live response: "old_deprecated_field"` {
			hasMissing = true
		}
		if d == `New key added by upstream API: "new_feature_flag"` {
			hasNew = true
		}
	}

	if !hasMissing {
		t.Fatalf("expected missing key detection in diffs: %v", diffs)
	}
	if !hasNew {
		t.Fatalf("expected new key detection in diffs: %v", diffs)
	}
}
