package proxy

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
)

// MatchResult contains the matched interaction or diagnostic information.
type MatchResult struct {
	Matched     *cassette.Interaction
	Index       int
	Diagnostics *MismatchDiagnostic
}

// MismatchDiagnostic contains detailed information on why a request didn't match.
type MismatchDiagnostic struct {
	RequestedMethod string   `json:"requested_method"`
	RequestedURL    string   `json:"requested_url"`
	ClosestID       string   `json:"closest_id,omitempty"`
	Reasons         []string `json:"reasons"`
	AvailableRoutes []string `json:"available_routes,omitempty"`
}

// Matcher evaluates incoming HTTP requests against recorded cassette interactions.
type Matcher struct {
	cfg           config.MatchingConfig
	seqMu         sync.Mutex
	consumedIndex map[string]int // tracks consumed index for sequential mode
}

// NewMatcher creates a new request matcher with configuration.
func NewMatcher(cfg config.MatchingConfig) *Matcher {
	return &Matcher{
		cfg:           cfg,
		consumedIndex: make(map[string]int),
	}
}

// ResetSequential clears sequential playback progress.
func (m *Matcher) ResetSequential() {
	m.seqMu.Lock()
	defer m.seqMu.Unlock()
	m.consumedIndex = make(map[string]int)
}

// FindMatch looks for an interaction in the cassette that matches the incoming request.
func (m *Matcher) FindMatch(cass *cassette.Cassette, method, reqURL string, headers map[string]string, body []byte) *MatchResult {
	if cass == nil || len(cass.Interactions) == 0 {
		return &MatchResult{
			Diagnostics: &MismatchDiagnostic{
				RequestedMethod: method,
				RequestedURL:    reqURL,
				Reasons:         []string{"Cassette contains no recorded interactions."},
			},
		}
	}

	parsedReqURL, _ := url.Parse(reqURL)
	var bestDiagnostic *MismatchDiagnostic
	var bestScore int
	availableRoutes := make([]string, 0, len(cass.Interactions))

	m.seqMu.Lock()
	defer m.seqMu.Unlock()

	for i, inter := range cass.Interactions {
		routeKey := fmt.Sprintf("%s %s", inter.Request.Method, inter.Request.URL)
		availableRoutes = append(availableRoutes, routeKey)

		// Check if already consumed in sequential mode
		if m.cfg.Sequential {
			if consumed, exists := m.consumedIndex[routeKey]; exists && i < consumed {
				continue
			}
		}

		matched, score, reasons := m.evaluateInteraction(inter, method, parsedReqURL, reqURL, headers, body)
		if matched {
			if m.cfg.Sequential {
				m.consumedIndex[routeKey] = i + 1
			}
			return &MatchResult{
				Matched: &inter,
				Index:   i,
			}
		}

		if score > bestScore {
			bestScore = score
			bestDiagnostic = &MismatchDiagnostic{
				RequestedMethod: method,
				RequestedURL:    reqURL,
				ClosestID:       inter.ID,
				Reasons:         reasons,
			}
		}
	}

	if bestDiagnostic == nil {
		bestDiagnostic = &MismatchDiagnostic{
			RequestedMethod: method,
			RequestedURL:    reqURL,
			Reasons:         []string{"No interaction matched requested method and URL path."},
		}
	}
	bestDiagnostic.AvailableRoutes = availableRoutes

	return &MatchResult{
		Diagnostics: bestDiagnostic,
	}
}

func (m *Matcher) evaluateInteraction(inter cassette.Interaction, method string, reqURL *url.URL, rawReqURL string, headers map[string]string, body []byte) (bool, int, []string) {
	score := 0
	var reasons []string

	// 1. Method check
	if !strings.EqualFold(inter.Request.Method, method) {
		reasons = append(reasons, fmt.Sprintf("Method mismatch: expected %s, got %s", inter.Request.Method, method))
		return false, score, reasons
	}
	score += 10

	interURL, err := url.Parse(inter.Request.URL)
	if err != nil {
		reasons = append(reasons, fmt.Sprintf("Failed to parse recorded interaction URL: %v", err))
		return false, score, reasons
	}

	// 2. Host check
	if reqURL != nil && interURL.Host != "" && reqURL.Host != "" {
		if !strings.EqualFold(interURL.Host, reqURL.Host) {
			reasons = append(reasons, fmt.Sprintf("Host mismatch: expected %s, got %s", interURL.Host, reqURL.Host))
			return false, score, reasons
		}
	}
	score += 10

	// 3. Path check
	if reqURL != nil && interURL.Path != reqURL.Path {
		reasons = append(reasons, fmt.Sprintf("Path mismatch: expected %s, got %s", interURL.Path, reqURL.Path))
		return false, score, reasons
	}
	score += 20

	if m.cfg.Strategy == "method_url" {
		return true, score, nil
	}

	// 4. Query parameters check
	if reqURL != nil {
		interQuery := m.normalizeQuery(interURL.Query())
		reqQuery := m.normalizeQuery(reqURL.Query())

		if !reflect.DeepEqual(interQuery, reqQuery) {
			reasons = append(reasons, fmt.Sprintf("Query parameters mismatch: expected %v, got %v", interQuery, reqQuery))
			return false, score, reasons
		}
	}
	score += 20

	// 5. Body check
	interBodyTrimmed := strings.TrimSpace(inter.Request.Body)
	reqBodyTrimmed := strings.TrimSpace(string(body))

	if interBodyTrimmed == "" && reqBodyTrimmed == "" {
		score += 30
		return true, score, nil
	}

	if interBodyTrimmed != "" || reqBodyTrimmed != "" {
		if m.cfg.Strategy == "exact" {
			if interBodyTrimmed != reqBodyTrimmed {
				reasons = append(reasons, "Exact request body mismatch")
				return false, score, reasons
			}
		} else {
			// Try JSON semantic comparison
			var interJSON, reqJSON interface{}
			interIsJSON := json.Unmarshal([]byte(interBodyTrimmed), &interJSON) == nil
			reqIsJSON := json.Unmarshal([]byte(reqBodyTrimmed), &reqJSON) == nil

			if interIsJSON && reqIsJSON {
				if !reflect.DeepEqual(interJSON, reqJSON) {
					reasons = append(reasons, "Semantic JSON request body mismatch")
					return false, score, reasons
				}
			} else {
				if interBodyTrimmed != reqBodyTrimmed {
					reasons = append(reasons, "Request body content mismatch")
					return false, score, reasons
				}
			}
		}
	}
	score += 30

	return true, score, nil
}

func (m *Matcher) normalizeQuery(vals url.Values) map[string][]string {
	res := make(map[string][]string)
	ignoreMap := make(map[string]bool)
	for _, ign := range m.cfg.IgnoreQueries {
		ignoreMap[strings.ToLower(ign)] = true
	}

	for k, v := range vals {
		if ignoreMap[strings.ToLower(k)] {
			continue
		}
		sortedVals := make([]string, len(v))
		copy(sortedVals, v)
		sort.Strings(sortedVals)
		res[k] = sortedVals
	}
	return res
}
