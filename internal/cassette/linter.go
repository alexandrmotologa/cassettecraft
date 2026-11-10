package cassette

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Violation represents an unredacted secret found in a cassette.
type Violation struct {
	File          string `json:"file"`
	InteractionID string `json:"interaction_id"`
	Location      string `json:"location"` // Request URL, Request Header, Request Body, Response Body, etc.
	SecretType    string `json:"secret_type"`
	Snippet       string `json:"snippet"`
}

// LintReport summarizes the results of a security scan across cassettes.
type LintReport struct {
	ScannedFiles int         `json:"scanned_files"`
	Clean        bool        `json:"clean"`
	Violations   []Violation `json:"violations"`
}

var (
	privateKeyRegex = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)
	stripeLiveRegex = regexp.MustCompile(`\b(sk|rk)_live_[0-9a-zA-Z]{20,}\b`)
	githubPatRegex  = regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[0-9a-zA-Z]{30,40}\b`)
	awsKeyRegex     = regexp.MustCompile(`\b(AKIA|ASIA)[0-9A-Z]{16}\b`)
	bearerRawRegex  = regexp.MustCompile(`(?i)\bBearer\s+([A-Za-z0-9_\-\.]{15,})\b`)
	creditCardRegex = regexp.MustCompile(`\b(?:\d{4}[ -]?){3}\d{4}\b`)
)

// LintPath recursively scans a directory or single file for unredacted secrets.
func LintPath(targetPath string) (*LintReport, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access path %q: %w", targetPath, err)
	}

	report := &LintReport{
		Clean:      true,
		Violations: make([]Violation, 0),
	}

	if !info.IsDir() {
		return LintFile(targetPath)
	}

	err = filepath.Walk(targetPath, func(path string, f os.FileInfo, err error) error {
		if err != nil || f.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" || ext == ".json" {
			fileReport, lintErr := LintFile(path)
			if lintErr == nil {
				report.ScannedFiles++
				if !fileReport.Clean {
					report.Clean = false
					report.Violations = append(report.Violations, fileReport.Violations...)
				}
			}
		}
		return nil
	})

	return report, err
}

// LintFile inspects a single cassette file for secret leaks.
func LintFile(filePath string) (*LintReport, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cassette file: %w", err)
	}

	var cass *Cassette
	if strings.HasSuffix(filePath, ".json") {
		cass, err = FromJSON(data)
	} else {
		cass, err = FromYAML(data)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse cassette: %w", err)
	}

	report := &LintReport{
		ScannedFiles: 1,
		Clean:        true,
		Violations:   make([]Violation, 0),
	}

	for _, inter := range cass.Interactions {
		// 1. Check Request URL
		checkStringForSecrets(filePath, inter.ID, "Request URL", inter.Request.URL, report)

		// 2. Check Request Headers
		for hKey, hVal := range inter.Request.Headers {
			if strings.EqualFold(hKey, "Authorization") && !strings.Contains(hVal, "[REDACTED") {
				report.addViolation(filePath, inter.ID, "Request Header: Authorization", "Unmasked Authorization Header", maskSnippet(hVal))
			} else {
				checkStringForSecrets(filePath, inter.ID, "Request Header: "+hKey, hVal, report)
			}
		}

		// 3. Check Request Body
		checkStringForSecrets(filePath, inter.ID, "Request Body", inter.Request.Body, report)
		checkJSONKeysForSecrets(filePath, inter.ID, "Request Body (JSON)", inter.Request.Body, report)

		// 4. Check Response Headers
		for hKey, hVal := range inter.Response.Headers {
			if strings.EqualFold(hKey, "Set-Cookie") && !strings.Contains(hVal, "[REDACTED") {
				report.addViolation(filePath, inter.ID, "Response Header: Set-Cookie", "Unmasked Set-Cookie Header", maskSnippet(hVal))
			} else {
				checkStringForSecrets(filePath, inter.ID, "Response Header: "+hKey, hVal, report)
			}
		}

		// 5. Check Response Body
		checkStringForSecrets(filePath, inter.ID, "Response Body", inter.Response.Body, report)
		checkJSONKeysForSecrets(filePath, inter.ID, "Response Body (JSON)", inter.Response.Body, report)
	}

	return report, nil
}

func (r *LintReport) addViolation(file, id, loc, secType, snippet string) {
	r.Clean = false
	r.Violations = append(r.Violations, Violation{
		File:          file,
		InteractionID: id,
		Location:      loc,
		SecretType:    secType,
		Snippet:       snippet,
	})
}

func checkStringForSecrets(file, id, loc, content string, report *LintReport) {
	if content == "" {
		return
	}

	if privateKeyRegex.MatchString(content) {
		report.addViolation(file, id, loc, "Private Key PEM", "Found private key block")
	}
	if match := stripeLiveRegex.FindString(content); match != "" {
		report.addViolation(file, id, loc, "Live Stripe Key", maskSnippet(match))
	}
	if match := githubPatRegex.FindString(content); match != "" {
		report.addViolation(file, id, loc, "GitHub Access Token", maskSnippet(match))
	}
	if match := awsKeyRegex.FindString(content); match != "" {
		report.addViolation(file, id, loc, "AWS Access Key", maskSnippet(match))
	}
	if match := bearerRawRegex.FindString(content); match != "" {
		if !strings.Contains(match, "[REDACTED") {
			report.addViolation(file, id, loc, "Raw Bearer Token", maskSnippet(match))
		}
	}
	if match := creditCardRegex.FindString(content); match != "" {
		// Ignore if already redacted
		if !strings.Contains(content, "[REDACTED_CARD_NUMBER]") {
			report.addViolation(file, id, loc, "Credit Card Number", maskSnippet(match))
		}
	}
}

func checkJSONKeysForSecrets(file, id, loc, body string, report *LintReport) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return
	}

	sensitiveKeys := []string{"password", "client_secret", "private_key", "secret_key"}
	for _, key := range sensitiveKeys {
		if val, exists := parsed[key]; exists {
			valStr := fmt.Sprintf("%v", val)
			if valStr != "[REDACTED]" && !strings.Contains(valStr, "[REDACTED") {
				report.addViolation(file, id, loc, "Plain JSON Secret Field: "+key, maskSnippet(valStr))
			}
		}
	}
}

func maskSnippet(text string) string {
	if len(text) <= 8 {
		return "******"
	}
	return text[:4] + "..." + text[len(text)-4:]
}
