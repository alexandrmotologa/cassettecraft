package cassette_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
)

func TestLinterCleanCassette(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cassettecraft-lint-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cass := cassette.NewCassette()
	cass.Interactions = append(cass.Interactions, cassette.Interaction{
		ID: "clean_int",
		Request: cassette.RequestData{
			Method: "GET",
			URL:    "https://api.example.com/data",
			Headers: map[string]string{
				"Authorization": "Bearer [REDACTED_TOKEN]",
			},
			Body: `{"username":"alice"}`,
		},
		Response: cassette.ResponseData{
			Status: 200,
			Body:   `{"status":"ok"}`,
		},
	})

	filePath := filepath.Join(tempDir, "clean.yaml")
	data, _ := cass.ToYAML()
	_ = os.WriteFile(filePath, data, 0644)

	report, err := cassette.LintPath(tempDir)
	if err != nil {
		t.Fatalf("lint failed: %v", err)
	}

	if !report.Clean {
		t.Fatalf("expected clean report, got violations: %v", report.Violations)
	}
	if len(report.Violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(report.Violations))
	}
}

func TestLinterDetectsViolations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cassettecraft-lint-leak-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cass := cassette.NewCassette()
	cass.Interactions = append(cass.Interactions, cassette.Interaction{
		ID: "leaky_int",
		Request: cassette.RequestData{
			Method: "POST",
			URL:    "https://api.example.com/login",
			Headers: map[string]string{
				"Authorization": "Bearer my_raw_unmasked_secret_token_12345",
			},
			Body: `{"password":"leaked_super_secret_pw"}`,
		},
		Response: cassette.ResponseData{
			Status: 200,
			Body:   `{"card":"4111 2222 3333 4444"}`,
		},
	})

	filePath := filepath.Join(tempDir, "leaky.yaml")
	data, _ := cass.ToYAML()
	_ = os.WriteFile(filePath, data, 0644)

	report, err := cassette.LintFile(filePath)
	if err != nil {
		t.Fatalf("lint failed: %v", err)
	}

	if report.Clean {
		t.Fatal("expected linter to catch violations")
	}

	if len(report.Violations) < 3 {
		t.Fatalf("expected at least 3 violations (bearer, password, credit card), got %d", len(report.Violations))
	}
}
