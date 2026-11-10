<p align="center">
  <img src="docs/images/logo.png?raw=true" alt="CassetteCraft Logo" width="130" style="border-radius: 24px;" />
</p>

<h1 align="center">CassetteCraft</h1>

<p align="center">
  <a href="https://github.com/alexandrmotologa/cassettecraft/actions"><img src="https://img.shields.io/badge/build-passing-brightgreen.svg" alt="Build Status" /></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.23%2B-blue.svg" alt="Go Version" /></a>
  <a href="https://react.dev"><img src="https://img.shields.io/badge/studio-React%2019-61dafb.svg" alt="React 19" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License" /></a>
  <a href="#minimal-memory-footprint"><img src="https://img.shields.io/badge/memory-%3C%2025MB%20RAM-purple.svg" alt="Memory" /></a>
</p>

<p align="center">
  <b>Self-hosted HTTP and HTTPS recording and replay proxy for flaky-free integration tests and CI/CD pipelines.</b>
</p>

---

`cassettecraft` intercepts outbound HTTP and HTTPS network traffic from your test suite through standard `HTTP_PROXY` and `HTTPS_PROXY` environment variables. During recording, it forwards requests to real upstream services, scrubs credentials, and writes deterministic YAML or JSON cassettes. During replay, it answers matching requests locally with sub-millisecond latency and zero external network calls.

---

## Demo Walkthrough

<p align="center">
  <img src="docs/images/cassettecraft_demo.gif?raw=true" alt="CassetteCraft Studio Animated Walkthrough" width="100%" style="border-radius: 8px; border: 1px solid #1e293b;" />
</p>

---

## Visual Cassette Studio

CassetteCraft includes an embedded web cockpit built with React 19, accessible at `http://localhost:8585` during test recording or standalone inspection.

### 1. Real-Time Interaction Waterfall & Inspection

Inspect live HTTP transactions, filter by status codes (2XX, 4XX, 5XX) or methods, view sub-millisecond replay latencies, and debug response payloads.

<p align="center">
  <img src="docs/images/cassettecraft_waterfall.png?raw=true" alt="CassetteCraft Live Waterfall" width="100%" style="border-radius: 8px; border: 1px solid #1e293b;" />
</p>

### 2. Cassette Manager & Interactive Mock Editor

Browse stored YAML cassettes, inspect recorded interactions, and modify mock responses or HTTP status codes directly in the visual editor.

<p align="center">
  <img src="docs/images/cassettecraft_manager.png?raw=true" alt="CassetteCraft Cassette Manager" width="100%" style="border-radius: 8px; border: 1px solid #1e293b;" />
</p>

### 3. HTTP Request Scratchpad

Compose arbitrary HTTP requests against upstream services or local cassettes with automatic cURL snippet generation.

<p align="center">
  <img src="docs/images/cassettecraft_scratchpad.png?raw=true" alt="CassetteCraft HTTP Scratchpad" width="100%" style="border-radius: 8px; border: 1px solid #1e293b;" />
</p>

### 4. Third-Party API Drift Detector

Compare recorded interactions with live upstream APIs to detect breaking schema changes and deprecated endpoints before deploying to production.

<p align="center">
  <img src="docs/images/cassettecraft_drift.png?raw=true" alt="CassetteCraft Drift Detector" width="100%" style="border-radius: 8px; border: 1px solid #1e293b;" />
</p>

### 5. Multi-Device Responsive Design

Cassette Studio adapts across all viewport sizes, from high-density desktop monitors down to mobile devices.

<p align="center">
  <img src="docs/images/cassettecraft_mobile.png?raw=true" alt="CassetteCraft Mobile Responsive Layout" width="340" style="border-radius: 12px; border: 1px solid #1e293b;" />
</p>

---

## Capabilities

- **Language independent**: Works with any language or test framework (Go, Node.js, Python, Rust, Java, PHP, cURL) through standard HTTP proxy configuration.
- **Transparent TLS MITM**: Handles HTTPS connections with on-the-fly leaf certificate generation and an in-memory or persisted local certificate authority.
- **AI Streaming and SSE support**: Intercepts Server-Sent Events (`text/event-stream`) and chunked responses, recording timing intervals and replaying streams with high fidelity.
- **Dynamic response templating**: Resolves runtime variables (`{{uuid}}`, `{{now_utc}}`, `{{request.body.field}}`) during replay to keep responses in sync with dynamic tests.
- **Multi-worker test isolation**: Supports concurrent parallel testing (`pytest -n 4`, `go test -parallel 4`) using isolated session contexts via the `X-Cassette-Session` header.
- **Security linter**: Scans repositories for unredacted secrets with `cassettecraft lint` before committing cassettes to version control.
- **API drift detector**: Compares recorded cassettes against live external endpoints with `cassettecraft drift` to detect breaking API changes or schema modifications.
- **HAR import and cURL generator**: Imports browser and DevTools `.har` archives into sanitized cassettes and generates executable cURL snippets.
- **Deterministic cassettes**: Sorts headers, normalizes JSON payloads, and calculates SHA-256 body hashes to keep git diffs minimal and stable.
- **Automated secret scrubbing**: Replaces API keys, Bearer tokens, basic authentication headers, credit card patterns, and sensitive JSON fields before writing to disk.
- **Flexible request matching**: Supports exact matching, path and method matching, normalized JSON body comparison, and sequential FIFO replay for stateful workflows.
- **Process execution wrapper**: The `cassettecraft exec` command runs test commands with proxy and CA environment variables pre-configured.
- **Embedded web studio**: Provides a visual cockpit built with React 19 to inspect captured transactions, edit mock responses, test requests in a scratchpad, and simulate latency.
- **Minimal memory footprint**: Consumes less than 25MB of RAM under load with zero CGo dependencies.

---

## Architecture Overview

```
[ Application Test Suite (Go, Python, Jest, Rust, Java, PHP) ]
                             |
                             |  HTTP_PROXY / HTTPS_PROXY = http://localhost:8585
                             v
+------------------------------------------------------------------------+
|                      cassettecraft Proxy Daemon                        |
|                                                                        |
|  +------------------------------------------------------------------+  |
|  | Ingress Listener (:8585) & TLS MITM Engine                       |  |
|  +----------------------------------+-------------------------------+  |
|                                     |                                  |
|         +---------------------------+--------------------------+       |
|         | Mode: RECORD                                         | Mode: REPLAY
|         v                                                      v       |
|  +-------------------------------+           +----------------------+  |
|  | Forward to Real Upstream API  |           | Match in Cassette    |  |
|  +--------------+----------------+           +----------+-----------+  |
|                 | Response                              | Found        |
|                 v                                       v              |
|  +-------------------------------+           +----------------------+  |
|  | Redact Secrets & Credentials  |           | Return Local Match   |  |
|  +--------------+----------------+           | (< 1ms, 0 net calls) |  |
|                 v                            +----------+-----------+  |
|  +-------------------------------+                      |              |
|  | Save Deterministic YAML/JSON  |                      |              |
|  +--------------+----------------+                      |              |
|                 v                                       v              |
|  [ Deliver Response Back to Test Suite: Status, Headers, Body ]        |
+------------------------------------------------------------------------+
```

---

## Quick Start

### 1. Installation

Download precompiled binaries or install from source using Go:

```bash
go install github.com/alexandrmotologa/cassettecraft/cmd/cassettecraft@latest
```

### 2. Record API Interactions

Start `cassettecraft` in record mode and execute your test suite:

```bash
cassettecraft exec --mode record --cassette tests/cassettes/stripe.yaml -- pytest tests/test_billing.py
```

`cassettecraft exec` sets `HTTP_PROXY`, `HTTPS_PROXY`, and the necessary certificate environment variables (`SSL_CERT_FILE`, `NODE_EXTRA_CA_CERTS`, `REQUESTS_CA_BUNDLE`) in the child process.

### 3. Replay in CI/CD

Run your test suite in replay mode without network access:

```bash
cassettecraft exec --mode replay --cassette tests/cassettes/stripe.yaml -- pytest tests/test_billing.py
```

In replay mode, matching requests return recorded responses in less than 1ms. If an unrecorded request is encountered, `cassettecraft` returns HTTP 502 with diagnostic details explaining why matching failed.

---

## CLI Reference

### `cassettecraft start`
Starts the proxy daemon in standalone mode:

```bash
cassettecraft start --port 8585 --mode record --cassette-dir ./cassettes --studio
```

Flags:
- `--port`, `-p`: Proxy listen port (default: `8585`).
- `--mode`, `-m`: Operating mode: `record`, `replay`, `auto`, or `passthrough` (default: `record`).
- `--cassette`, `-c`: Path to the default cassette file or name.
- `--cassette-dir`, `-d`: Directory containing cassette files (default: `./cassettes`).
- `--studio`: Enable embedded React 19 Cassette Studio on port 8585 (default: `true`).
- `--latency`: Inject synthetic latency into replayed responses, e.g. `50ms`.
- `--config`: Path to optional configuration file (`cassettecraft.yaml`).

### `cassettecraft exec`
Spawns a child process with proxy and certificate environment variables set automatically:

```bash
cassettecraft exec --mode replay --cassette tests/cassettes/auth.yaml -- npm test
```

### `cassettecraft lint`
Scans cassette files for unredacted credentials before git commit:

```bash
cassettecraft lint ./cassettes
```

### `cassettecraft drift`
Audits recorded cassettes against live external APIs to detect schema drift:

```bash
cassettecraft drift --cassette stripe.yaml
```

### `cassettecraft import`
Converts HTTP Archive (`.har`) files exported from browsers or DevTools into clean cassettes:

```bash
cassettecraft import --format har --in network.har --out ./cassettes/browser_flow.yaml
```

### `cassettecraft ca`
Manages the local Root Certificate Authority:

```bash
# Export the root certificate for manual trust store installation
cassettecraft ca export --out ./ca.crt
```

---

## Operating Modes

1. **`record`**: Forwards requests to real servers, scrubs credentials, and writes responses to the target cassette.
2. **`replay`**: Answers requests strictly from loaded cassettes. Returns HTTP 502 if an unrecorded request is made.
3. **`auto`**: Replays matching interactions if they exist in the cassette; forwards to upstream and records new interactions when unseen.
4. **`passthrough`**: Functions as a standard HTTP/HTTPS forward proxy without recording or replaying.

---

## Secret Redaction

`cassettecraft` scrubs sensitive values before saving cassettes to disk:

- **Headers**: `Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key`, `Stripe-Signature`.
- **Query parameters**: `api_key`, `token`, `secret`, `access_token`.
- **Known credential patterns**: Stripe API keys (`sk_live_...`, `rk_live_...`), GitHub tokens (`ghp_...`, `gho_...`), OpenAI API keys (`sk-...`), AWS access keys (`AKIA...`), and JSON Web Tokens (`ey...`).
- **Structured JSON fields**: Any key named `password`, `secret`, `token`, `apiKey`, `clientSecret`, or `card_number`.

Custom redaction rules can be defined in `cassettecraft.yaml`:

```yaml
redaction:
  headers:
    - "X-Custom-Auth"
  json_keys:
    - "private_seed"
    - "tax_identifier"
  patterns:
    - name: "custom_token"
      regex: "cust_[a-zA-Z0-9]{32}"
      replacement: "[REDACTED_CUSTOM_TOKEN]"
```

---

## CI/CD Pipeline Integration

### GitHub Actions

```yaml
name: Integration Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Install cassettecraft
        run: go install github.com/alexandrmotologa/cassettecraft/cmd/cassettecraft@latest

      - name: Run Tests with Cassette Replay
        run: |
          cassettecraft exec --mode replay --cassette-dir ./tests/cassettes -- go test -v ./...
```

---

## Documentation

- [Architecture Details](docs/ARCHITECTURE.md)
- [Cassette Specification](docs/CASSETTE_SPEC.md)
- [Secret Redaction Guide](docs/SECRET_REDACTION.md)
- [CI/CD Integration Guide](docs/CI_CD_GUIDE.md)
- [Code Examples](docs/EXAMPLES.md)

---

## License

This project is licensed under the Apache License 2.0. See LICENSE for details.
