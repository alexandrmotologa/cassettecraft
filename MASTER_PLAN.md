# cassettecraft: Architecture and Engineering Master Plan

Self-hosted HTTP recording and replay proxy for deterministic integration testing and CI/CD pipelines.

---

## 1. Overview and Problem Space

### Background
Automated integration testing against external APIs (such as Stripe, GitHub, Twilio, OpenAI, Resend, and AWS) presents recurring operational challenges:
1. Test flakiness from network jitter, rate limiting (`HTTP 429`), and upstream downtime causes unexpected pipeline failures.
2. Paid API consumption adds operational cost when test suites run on every commit or pull request.
3. Language-specific recording libraries (Ruby VCR, Python VCR.py/Betamax, Node.js Polly.js) require per-language SDKs and runtime monkey-patching of HTTP clients. In polyglot codebases or multi-service repositories, managing disparate mock setups increases maintenance overhead.

### Purpose of cassettecraft
`cassettecraft` is a language-agnostic HTTP and HTTPS recording and replay proxy written in Go with an embedded React 19 visual cassette studio. It operates transparently through standard `HTTP_PROXY` and `HTTPS_PROXY` environment variables.

In record mode, `cassettecraft` intercepts outbound requests, scrubs configured credentials and sensitive data, and saves clean YAML or JSON cassettes to disk. In replay mode, `cassettecraft` serves matching responses locally with sub-millisecond response times, without making external network calls.

---

## 2. System Architecture

```
[ Application Test Suite (Go, Python, Jest, Rust, Java, PHP, cURL) ]
                             |
                             |  HTTP_PROXY / HTTPS_PROXY = http://localhost:8585
                             v
+------------------------------------------------------------------------+
|                      cassettecraft Proxy Daemon                        |
|                                                                        |
|  +------------------------------------------------------------------+  |
|  | Ingress HTTP / HTTPS Connect Listener (:8585)                    |  |
|  | - TLS MITM Interceptor with dynamic certificate generation        |  |
|  | - Request Normalizer (Method, Path, Query Params, Headers, Body) |  |
|  +----------------------------------+-------------------------------+  |
|                                     |                                  |
|         +---------------------------+--------------------------+       |
|         | Mode: RECORD                                         | Mode: REPLAY
|         v                                                      v       |
|  +-------------------------------+           +----------------------+  |
|  | Forward to Upstream API       |           | Cassette Matcher     |  |
|  | (e.g. https://api.stripe.com) |           | - Exact / Body JSON  |  |
|  +--------------+----------------+           | - FIFO Sequence Mode |  |
|                 | Response                   +----------+-----------+  |
|                 v                                       | Match Found  |
|  +-------------------------------+                      v              |
|  | Secret Sanitizer and Redactor |           +----------------------+  |
|  | - Bearer tokens, API keys     |           | Local Replay Stream  |  |
|  | - PII, credit cards, JWTs     |           | (< 1ms latency)      |  |
|  +--------------+----------------+           +----------+-----------+  |
|                 v                                       |              |
|  +-------------------------------+                      |              |
|  | Serialize to Deterministic    |                      |              |
|  | Cassette (YAML/JSON)          |                      |              |
|  +--------------+----------------+                      |              |
|                 v                                       v              |
|  [ Deliver HTTP Response Back to Test Suite: Status, Headers, Body ]   |
|                                                                        |
|  +------------------------------------------------------------------+  |
|  | Embedded React 19 Cassette Studio                                |  |
|  | - Intercepted request waterfall and inspection tabs              |  |
|  | - Cassette manager, inline response editor, diff comparison       |  |
|  | - Interactive secret redaction rules tester                      |  |
|  | - Synthetic latency and chaos injection simulator                |  |
|  +------------------------------------------------------------------+  |
+------------------------------------------------------------------------+
```

---

## 3. Technology Stack

| Layer | Component | Specification |
| :--- | :--- | :--- |
| Core Language | Go 1.23+ | Low memory consumption (< 25MB RAM under load), native concurrency, zero CGo |
| Proxy Engine | HTTP / HTTPS MITM Proxy | Handles plain HTTP forwarding and HTTPS `CONNECT` tunnels with dynamic TLS leaf certificate generation |
| Cryptography | `crypto/tls`, `crypto/x509` | Ephemeral or persistent Root CA; on-demand SAN leaf certificates with LRU cache |
| Serialization | Deterministic YAML and JSON | Sorted headers, normalized JSON payloads, SHA-256 body hashes for git diff stability |
| Secret Redactor | Regex and JSON Tree Traversal | Built-in rules for Bearer, Stripe, GitHub, OpenAI, AWS, JWT, credit cards, and custom user rules |
| Matching Engine | Multi-strategy matcher | Exact, method and path, normalized JSON body, header exclusion list, and sequential FIFO playback |
| Web UI | React 19, TypeScript, Vite | Embedded into the Go binary via `//go:embed` |
| UI Styling | Tailwind CSS, Lucide Icons | Clean dark layout designed for developer productivity |

---

## 4. Repository Structure

```
cassettecraft/
├── cmd/
│   └── cassettecraft/
│       └── main.go                 # CLI entrypoint (start, record, replay, exec, ca, studio)
├── internal/
│   ├── config/
│   │   └── config.go               # Configuration loading, CLI flags, environment variables
│   ├── proxy/
│   │   ├── server.go               # Ingress HTTP/HTTPS proxy server and CONNECT handler
│   │   ├── mitm.go                 # TLS dynamic certificate generator with SAN support
│   │   ├── roundtripper.go         # Upstream forwarding and response capture
│   │   └── matcher.go              # Request matching engine and diff diagnostics
│   ├── cassette/
│   │   ├── format.go               # Cassette data models (YAML/JSON) and metadata
│   │   ├── storage.go              # Cassette filesystem storage, loading, and indexing
│   │   └── sanitizer.go            # Multi-layer secret redaction engine
│   ├── chaos/
│   │   └── chaos.go                # Latency injection, error rate simulation, rate limiting
│   ├── certs/
│   │   └── ca.go                   # Root CA generator, storage, and certificate export
│   ├── runner/
│   │   └── exec.go                 # Child process runner with automatic proxy environment injection
│   └── api/
│       ├── router.go               # REST API for proxy control and studio backend
│       ├── cassettes.go            # Cassette inspection, editing, and deletion endpoints
│       └── interactions.go         # Live interaction ring buffer and streaming
├── web/
│   ├── src/
│   │   ├── components/
│   │   │   ├── Navbar.tsx          # Status indicators, mode switcher, and navigation
│   │   │   ├── Waterfall.tsx       # Live waterfall timeline of intercepted requests
│   │   │   ├── Inspector.tsx       # Request and response headers, query, and payload viewer
│   │   │   ├── CassetteManager.tsx # Cassette file browser, interaction list, inline editor
│   │   │   ├── DiffViewer.tsx      # Side-by-side diff between recorded and live responses
│   │   │   ├── RedactionTester.tsx # Live regex and field masking testing sandbox
│   │   │   └── ChaosControls.tsx   # Synthetic latency and error simulation controls
│   │   ├── types/
│   │   │   └── api.ts              # TypeScript interfaces matching backend models
│   │   ├── App.tsx                 # Root studio interface
│   │   ├── main.tsx                # React 19 mount point
│   │   └── index.css               # Tailwind CSS declarations
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── docs/
│   ├── ARCHITECTURE.md             # In-depth architectural details and proxy flow
│   ├── CASSETTE_SPEC.md            # Cassette schema, determinism rules, and serialization
│   ├── SECRET_REDACTION.md         # Redaction guide, default patterns, and custom rules
│   ├── CI_CD_GUIDE.md              # Recipes for GitHub Actions, GitLab CI, and Docker
│   └── EXAMPLES.md                 # Real examples for Go, Python, Node.js, and cURL
├── tests/
│   ├── integration_test.go         # End-to-end integration tests (Record -> Replay cycle)
│   ├── mitm_test.go                # HTTPS CONNECT interception tests
│   └── fixtures/                   # Sample test fixtures and upstream mocks
├── Makefile                        # Multi-platform build, test, and release targets
├── Dockerfile                      # Standalone container build
├── README.md                       # Main documentation with quick start and CLI reference
└── MASTER_PLAN.md                  # This implementation plan
```

---

## 5. Detailed Implementation Tasks

### Phase 1: Repository Setup and Documentation
- [x] Create public GitHub repository `alexandrmotologa/cassettecraft` with coupled description and 14 topics.
- [x] Register project in active development in `mtlg-site` and `mtlglabs-space`.
- [x] Author core documentation adhering to humanizer guidelines (`README.md`, `ARCHITECTURE.md`, `CASSETTE_SPEC.md`, `SECRET_REDACTION.md`, `CI_CD_GUIDE.md`, `EXAMPLES.md`).

### Phase 2: Core Certificate Authority and TLS MITM Interception
- [ ] Implement `internal/certs/ca.go` for persistent and ephemeral Root CA generation (`ECDSA` / `RSA 2048`, `x509` extensions, validity period).
- [ ] Implement `internal/proxy/mitm.go` for dynamic leaf certificate generation per SNI hostname with thread-safe LRU caching.
- [ ] Implement CA export helpers for system certificates (`ca.crt`, `ca.key`) and CLI command `cassettecraft ca export`.

### Phase 3: Proxy Core Engine (HTTP & HTTPS CONNECT)
- [ ] Implement `internal/proxy/server.go`:
  - Handle standard HTTP forward proxy requests (`GET http://api.stripe.com/v1/...`).
  - Handle HTTPS `CONNECT` tunnels (`CONNECT api.stripe.com:443`).
  - Hijack TLS connection, perform handshake with dynamic leaf certificate, read decrypted HTTP request, forward upstream or resolve from cassette, and return encrypted response to client.
- [ ] Implement proxy modes:
  - `record`: forwards to upstream, intercepts response, applies sanitizer, appends to active cassette.
  - `replay`: looks up request in active cassette, returns matching response. Fails with diagnostic 502/404 if no match found.
  - `auto`: replays if recorded; forwards and records if unseen.
  - `passthrough`: transparent forward proxy without recording or replaying.
- [ ] Implement dynamic cassette assignment via header (`X-Cassette-Name`) or API.

### Phase 4: Cassette Format, Storage, and Secret Redaction
- [ ] Implement `internal/cassette/format.go` with deterministic YAML/JSON structs.
- [ ] Implement `internal/cassette/storage.go`:
  - Thread-safe cassette file loading, indexing, and writing.
  - Ordered interactions list with unique IDs.
- [ ] Implement `internal/cassette/sanitizer.go`:
  - Redacts sensitive headers (`Authorization`, `Cookie`, `X-Api-Key`, `Stripe-Signature`).
  - Redacts sensitive query parameters (`api_key`, `token`, `secret`).
  - Redacts structured JSON fields (`password`, `client_secret`, `access_token`, `refresh_token`, `card_number`, `cvv`).
  - Regex rules for Stripe keys, GitHub tokens, OpenAI keys, AWS access keys, Bearer tokens, and credit cards.
  - Custom user redaction regexes from configuration.
- [ ] Implement `internal/proxy/matcher.go`:
  - Exact match, Path and Method match, Normalized JSON body match.
  - Configurable ignored headers (`Date`, `User-Agent`, `X-Request-Id`, `Traceparent`).
  - FIFO sequential playback option for polling and stateful endpoints.
  - Match failure diagnostics showing side-by-side mismatch reasons.

### Phase 5: Chaos Simulation and Process Runner
- [ ] Implement `internal/chaos/chaos.go`:
  - Artificial latency injection (fixed or jittered).
  - Error rate injection (`HTTP 500`, `HTTP 503`, `HTTP 429`).
- [ ] Implement `internal/runner/exec.go`:
  - Spawn child test processes with injected proxy environment variables (`HTTP_PROXY`, `HTTPS_PROXY`, `SSL_CERT_FILE`, `NODE_EXTRA_CA_CERTS`, `REQUESTS_CA_BUNDLE`).
  - Propagate signals and child exit codes.

### Phase 6: REST Control API and Live Interaction Buffer
- [ ] Implement `internal/api/router.go` and handlers:
  - `GET /api/v1/health`
  - `GET /api/v1/status`
  - `POST /api/v1/mode`
  - `GET /api/v1/cassettes`
  - `GET /api/v1/cassettes/{name}`
  - `POST /api/v1/cassettes/{name}`
  - `DELETE /api/v1/cassettes/{name}`
  - `GET /api/v1/interactions/live`
  - `POST /api/v1/interactions/purge`
  - `GET /api/v1/ca.crt`
  - `POST /api/v1/chaos`
  - `POST /api/v1/redact/test`

### Phase 7: React 19 Cassette Studio Frontend
- [ ] Scaffold React 19 + TypeScript + Vite + Tailwind CSS in `web/`.
- [ ] Implement visual Waterfall timeline of all intercepted interactions.
- [ ] Implement Interaction Inspector with tabs for Headers, Query, Request Body, Response Body.
- [ ] Implement Cassette Browser with inline response editor.
- [ ] Implement Diff Viewer between live and recorded responses.
- [ ] Implement Live Mode Switcher and Chaos Controls.
- [ ] Implement Redaction Sandbox for testing regex rules.
- [ ] Embed frontend build directly into the Go binary using `//go:embed`.

### Phase 8: Testing, Benchmarking, and Verification
- [ ] Unit tests for CA generation, leaf certificate issuance, and TLS handshakes.
- [ ] Unit tests for secret sanitization and JSON field masking.
- [ ] Unit tests for matcher algorithms (exact, JSON body, sequential FIFO).
- [ ] End-to-end integration tests: record real upstream mock, replay offline, verify zero network calls and < 1ms response latency.
- [ ] Child runner integration tests (`cassettecraft exec`).
- [ ] Memory and performance benchmarks verifying < 25MB RAM under sustained load.
