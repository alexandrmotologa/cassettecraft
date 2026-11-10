package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/certs"
	"github.com/alexandrmotologa/cassettecraft/internal/chaos"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
	"github.com/alexandrmotologa/cassettecraft/internal/drift"
	"github.com/alexandrmotologa/cassettecraft/internal/proxy"
)

// ServerDependencies interfaces the proxy server capabilities needed by the API.
type ServerDependencies interface {
	GetMode() config.Mode
	SetMode(config.Mode) error
	GetBuffer() *proxy.InteractionBuffer
	GetStorage() *cassette.Storage
	GetSanitizer() *cassette.Sanitizer
	GetChaos() *chaos.Chaos
}

// Handler handles API routes.
type Handler struct {
	server  ServerDependencies
	ca      *certs.CA
	webFS   http.FileSystem
	mux     *http.ServeMux
}

// NewHandler creates a new API and static asset handler.
func NewHandler(server ServerDependencies, ca *certs.CA, webFS http.FileSystem) *Handler {
	h := &Handler{
		server: server,
		ca:     ca,
		webFS:  webFS,
		mux:    http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}

// ServeHTTP delegates to the configured mux.
func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Enable CORS for development
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Cassette-Mode, X-Cassette-Name")

	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	h.mux.ServeHTTP(w, req)
}

func (h *Handler) registerRoutes() {
	// Health & Status
	h.mux.HandleFunc("/api/v1/health", h.handleHealth)
	h.mux.HandleFunc("/api/v1/status", h.handleStatus)
	h.mux.HandleFunc("/api/v1/mode", h.handleMode)

	// Cassettes
	h.mux.HandleFunc("/api/v1/cassettes", h.handleCassettesList)
	h.mux.HandleFunc("/api/v1/cassettes/", h.handleCassetteDetail)
	h.mux.HandleFunc("/api/v1/active-cassette", h.handleActiveCassette)

	// Live Interactions
	h.mux.HandleFunc("/api/v1/interactions/live", h.handleLiveInteractions)
	h.mux.HandleFunc("/api/v1/interactions/purge", h.handlePurgeInteractions)
	h.mux.HandleFunc("/api/v1/interactions/seed", h.handleSeedInteractions)

	// Root CA Download
	h.mux.HandleFunc("/ca.crt", h.handleDownloadCA)
	h.mux.HandleFunc("/api/v1/ca.crt", h.handleDownloadCA)

	// Chaos Simulation Controls
	h.mux.HandleFunc("/api/v1/chaos", h.handleChaos)

	// Redaction Sandbox Tester
	h.mux.HandleFunc("/api/v1/redact/test", h.handleRedactTest)

	// Security Audit Linter
	h.mux.HandleFunc("/api/v1/lint", h.handleLint)

	// API Drift Detector
	h.mux.HandleFunc("/api/v1/drift", h.handleDrift)

	// Web Studio static assets
	if h.webFS != nil {
		fileServer := http.FileServer(h.webFS)
		h.mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			// Don't intercept API routes
			if strings.HasPrefix(req.URL.Path, "/api/") {
				http.NotFound(w, req)
				return
			}
			fileServer.ServeHTTP(w, req)
		})
	} else {
		h.mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/api/") {
				http.NotFound(w, req)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>CassetteCraft</title></head><body style="background:#0f172a;color:#f8fafc;font-family:sans-serif;padding:2rem;"><h1>CassetteCraft Proxy Daemon</h1><p>Status: Running</p><p><a href="/api/v1/status" style="color:#38bdf8;">API Status</a> | <a href="/ca.crt" style="color:#38bdf8;">Download CA Certificate</a></p></body></html>`))
		})
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, req *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) handleStatus(w http.ResponseWriter, req *http.Request) {
	storage := h.server.GetStorage()
	buffer := h.server.GetBuffer()
	chaosCfg := h.server.GetChaos().GetConfig()

	cassettes, _ := storage.ListCassettes()

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"mode":               h.server.GetMode(),
		"active_cassette":    storage.GetActiveName(),
		"cassettes_count":    len(cassettes),
		"interactions_count": len(buffer.GetAll()),
		"ca_fingerprint":     h.ca.FingerprintSHA256(),
		"chaos":              chaosCfg,
	})
}

func (h *Handler) handleMode(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"mode": h.server.GetMode()})
		return
	}

	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	m := config.Mode(strings.ToLower(payload.Mode))
	if err := h.server.SetMode(m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"mode":    h.server.GetMode(),
		"message": "Proxy mode updated successfully",
	})
}

func (h *Handler) handleCassettesList(w http.ResponseWriter, req *http.Request) {
	storage := h.server.GetStorage()
	names, err := storage.ListCassettes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type CassetteSummary struct {
		Name         string `json:"name"`
		Interactions int    `json:"interactions"`
		Active       bool   `json:"active"`
	}

	activeName := storage.GetActiveName()
	var summaries []CassetteSummary
	for _, n := range names {
		cass, err := storage.Load(n)
		count := 0
		if err == nil && cass != nil {
			count = len(cass.Interactions)
		}
		summaries = append(summaries, CassetteSummary{
			Name:         n,
			Interactions: count,
			Active:       n == activeName,
		})
	}

	jsonResponse(w, http.StatusOK, summaries)
}

func (h *Handler) handleCassetteDetail(w http.ResponseWriter, req *http.Request) {
	name := strings.TrimPrefix(req.URL.Path, "/api/v1/cassettes/")
	if name == "" {
		http.Error(w, "Cassette name required", http.StatusBadRequest)
		return
	}

	storage := h.server.GetStorage()

	switch req.Method {
	case http.MethodGet:
		cass, err := storage.Load(name)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load cassette: %v", err), http.StatusNotFound)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"name":     name,
			"cassette": cass,
		})

	case http.MethodPost, http.MethodPut:
		var cass cassette.Cassette
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(bodyBytes, &cass); err != nil {
			http.Error(w, fmt.Sprintf("Invalid cassette JSON: %v", err), http.StatusBadRequest)
			return
		}

		if err := storage.Save(name, &cass); err != nil {
			http.Error(w, fmt.Sprintf("Failed to save cassette: %v", err), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Cassette saved successfully",
			"name":    name,
		})

	case http.MethodDelete:
		if err := storage.DeleteCassette(name); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete cassette: %v", err), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Cassette deleted successfully",
			"name":    name,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleActiveCassette(w http.ResponseWriter, req *http.Request) {
	storage := h.server.GetStorage()

	if req.Method == http.MethodGet {
		jsonResponse(w, http.StatusOK, map[string]string{
			"active_cassette": storage.GetActiveName(),
		})
		return
	}

	if req.Method == http.MethodPost {
		var payload struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload.Name == "" {
			http.Error(w, "Name field required", http.StatusBadRequest)
			return
		}
		storage.SetActiveName(payload.Name)
		jsonResponse(w, http.StatusOK, map[string]string{
			"active_cassette": storage.GetActiveName(),
			"message":         "Active cassette updated",
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) handleLiveInteractions(w http.ResponseWriter, req *http.Request) {
	buffer := h.server.GetBuffer()
	interactions := buffer.GetAll()
	jsonResponse(w, http.StatusOK, interactions)
}

func (h *Handler) handlePurgeInteractions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	buffer := h.server.GetBuffer()
	buffer.Clear()
	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Live interactions buffer purged",
	})
}

func (h *Handler) handleSeedInteractions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var items []proxy.LiveInteraction
	if err := json.NewDecoder(req.Body).Decode(&items); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusBadRequest)
		return
	}
	buffer := h.server.GetBuffer()
	for _, it := range items {
		buffer.Add(it)
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Interactions seeded successfully",
		"count":   len(items),
	})
}

func (h *Handler) handleDownloadCA(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", "attachment; filename=\"cassettecraft-ca.crt\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.ca.CertPEM)
}

func (h *Handler) handleChaos(w http.ResponseWriter, req *http.Request) {
	chaosSim := h.server.GetChaos()

	if req.Method == http.MethodGet {
		jsonResponse(w, http.StatusOK, chaosSim.GetConfig())
		return
	}

	if req.Method == http.MethodPost {
		var cfg config.ChaosConfig
		if err := json.NewDecoder(req.Body).Decode(&cfg); err != nil {
			http.Error(w, "Invalid chaos config JSON", http.StatusBadRequest)
			return
		}
		chaosSim.UpdateConfig(cfg)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Chaos configuration updated",
			"chaos":   chaosSim.GetConfig(),
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) handleRedactTest(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	sanitizer := h.server.GetSanitizer()
	sanitized := sanitizer.SanitizeBody(payload.Text)

	jsonResponse(w, http.StatusOK, map[string]string{
		"original":  payload.Text,
		"sanitized": sanitized,
	})
}

func (h *Handler) handleLint(w http.ResponseWriter, req *http.Request) {
	storage := h.server.GetStorage()
	baseDir := storage.GetBaseDir()

	report, err := cassette.LintPath(baseDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to lint cassettes: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, report)
}

func (h *Handler) handleDrift(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Cassette string `json:"cassette"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload.Cassette == "" {
		http.Error(w, "Cassette field required in JSON payload", http.StatusBadRequest)
		return
	}

	storage := h.server.GetStorage()
	cass, err := storage.Load(payload.Cassette)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load cassette: %v", err), http.StatusNotFound)
		return
	}

	report, err := drift.CheckCassetteDrift(cass, payload.Cassette, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check drift: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, report)
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
