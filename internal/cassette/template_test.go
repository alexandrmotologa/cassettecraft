package cassette_test

import (
	"strings"
	"testing"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
)

func TestApplyResponseTemplates(t *testing.T) {
	resp := cassette.ResponseData{
		Status: 200,
		Headers: map[string]string{
			"X-Request-Correlation": "{{request.header.X-Correlation-ID}}",
		},
		Body: `{
  "id": "{{uuid}}",
  "created_at": "{{now_utc}}",
  "user_id": "{{request.body.user.id}}",
  "search_query": "{{request.query.q}}"
}`,
	}

	req := cassette.RequestData{
		Method: "POST",
		Body:   `{"user":{"id":"usr_998877"}}`,
	}

	headers := map[string]string{
		"X-Correlation-ID": "corr-1234-abcd",
	}

	rawURL := "https://api.example.com/v1/search?q=golang+testing"

	templated := cassette.ApplyResponseTemplates(resp, req, rawURL, headers)

	if !strings.Contains(templated.Body, `"user_id": "usr_998877"`) {
		t.Fatalf("expected user_id to be substituted from request body, got: %s", templated.Body)
	}

	if !strings.Contains(templated.Body, `"search_query": "golang testing"`) {
		t.Fatalf("expected search_query to be substituted from query param, got: %s", templated.Body)
	}

	if strings.Contains(templated.Body, "{{uuid}}") || strings.Contains(templated.Body, "{{now_utc}}") {
		t.Fatalf("unresolved template variables in body: %s", templated.Body)
	}

	if templated.Headers["X-Request-Correlation"] != "corr-1234-abcd" {
		t.Fatalf("expected correlation header substitution, got %q", templated.Headers["X-Request-Correlation"])
	}
}
