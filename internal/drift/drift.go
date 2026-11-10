package drift

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
)

// DriftItem represents the structural difference detected for a single interaction.
type DriftItem struct {
	InteractionID  string   `json:"interaction_id"`
	Method         string   `json:"method"`
	URL            string   `json:"url"`
	RecordedStatus int      `json:"recorded_status"`
	LiveStatus     int      `json:"live_status"`
	StatusChanged  bool     `json:"status_changed"`
	Differences    []string `json:"differences"`
}

// DriftReport holds the full drift audit result for a cassette.
type DriftReport struct {
	Cassette          string      `json:"cassette"`
	TotalInteractions int         `json:"total_interactions"`
	DriftedCount      int         `json:"drifted_count"`
	HasDrift          bool        `json:"has_drift"`
	Items             []DriftItem `json:"items"`
}

// CheckCassetteDrift replays requests from a cassette against real upstreams and compares schemas.
func CheckCassetteDrift(cass *cassette.Cassette, cassName string, client *http.Client) (*DriftReport, error) {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	report := &DriftReport{
		Cassette:          cassName,
		TotalInteractions: len(cass.Interactions),
		Items:             make([]DriftItem, 0),
	}

	for _, inter := range cass.Interactions {
		item := compareInteractionWithUpstream(inter, client)
		if len(item.Differences) > 0 || item.StatusChanged {
			report.DriftedCount++
			report.HasDrift = true
		}
		report.Items = append(report.Items, item)
	}

	return report, nil
}

func compareInteractionWithUpstream(inter cassette.Interaction, client *http.Client) DriftItem {
	item := DriftItem{
		InteractionID:  inter.ID,
		Method:         inter.Request.Method,
		URL:            inter.Request.URL,
		RecordedStatus: inter.Response.Status,
		Differences:    make([]string, 0),
	}

	req, err := http.NewRequest(inter.Request.Method, inter.Request.URL, strings.NewReader(inter.Request.Body))
	if err != nil {
		item.Differences = append(item.Differences, fmt.Sprintf("Failed to construct request: %v", err))
		return item
	}

	for k, v := range inter.Request.Headers {
		// Skip pseudo-headers or host
		if !strings.EqualFold(k, "Host") && !strings.EqualFold(k, "Content-Length") {
			req.Header.Set(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		item.Differences = append(item.Differences, fmt.Sprintf("Upstream connection error: %v", err))
		return item
	}
	defer resp.Body.Close()

	item.LiveStatus = resp.StatusCode
	if resp.StatusCode != inter.Response.Status {
		item.StatusChanged = true
		item.Differences = append(item.Differences, fmt.Sprintf("Status code changed: recorded %d, live %d", inter.Response.Status, resp.StatusCode))
	}

	liveBody, _ := io.ReadAll(resp.Body)

	// Compare JSON structure if both are JSON
	var recJSON, liveJSON interface{}
	recIsJSON := json.Unmarshal([]byte(inter.Response.Body), &recJSON) == nil
	liveIsJSON := json.Unmarshal(liveBody, &liveJSON) == nil

	if recIsJSON && liveIsJSON {
		diffs := compareJSONStructures("", recJSON, liveJSON)
		item.Differences = append(item.Differences, diffs...)
	} else if len(inter.Response.Body) > 0 && len(liveBody) > 0 && !bytes.Equal([]byte(inter.Response.Body), liveBody) {
		if !recIsJSON {
			item.Differences = append(item.Differences, "Non-JSON response body content changed")
		}
	}

	return item
}

func compareJSONStructures(path string, rec, live interface{}) []string {
	var diffs []string

	if reflect.TypeOf(rec) != reflect.TypeOf(live) && rec != nil && live != nil {
		diffs = append(diffs, fmt.Sprintf("Type mismatch at %q: recorded %T, live %T", path, rec, live))
		return diffs
	}

	switch recVal := rec.(type) {
	case map[string]interface{}:
		liveVal, ok := live.(map[string]interface{})
		if !ok {
			diffs = append(diffs, fmt.Sprintf("Structure mismatch at %q: expected object", path))
			return diffs
		}

		// Check for missing keys in live response
		for k, v := range recVal {
			subPath := k
			if path != "" {
				subPath = path + "." + k
			}
			if liveSubVal, exists := liveVal[k]; !exists {
				diffs = append(diffs, fmt.Sprintf("Key missing in live response: %q", subPath))
			} else {
				diffs = append(diffs, compareJSONStructures(subPath, v, liveSubVal)...)
			}
		}

		// Check for new keys in live response
		for k := range liveVal {
			subPath := k
			if path != "" {
				subPath = path + "." + k
			}
			if _, exists := recVal[k]; !exists {
				diffs = append(diffs, fmt.Sprintf("New key added by upstream API: %q", subPath))
			}
		}

	case []interface{}:
		liveSlice, ok := live.([]interface{})
		if !ok {
			diffs = append(diffs, fmt.Sprintf("Structure mismatch at %q: expected array", path))
			return diffs
		}
		if len(recVal) > 0 && len(liveSlice) > 0 {
			diffs = append(diffs, compareJSONStructures(path+"[0]", recVal[0], liveSlice[0])...)
		}
	}

	return diffs
}
