# CassetteCraft Architecture

This document describes the internal architecture of cassettecraft, covering network interception, TLS man in the middle processing, request matching, and memory efficiency.

---

## 1. Network Interception Model

`cassettecraft` operates as an HTTP/1.1 forward proxy listening on a single TCP port (default `:8585`). Applications route network requests through the proxy using standard environment variables:

- `HTTP_PROXY` and `http_proxy`: Routes plain HTTP traffic.
- `HTTPS_PROXY` and `https_proxy`: Routes encrypted HTTPS traffic via the HTTP `CONNECT` tunnel method.
- `ALL_PROXY`: Universal proxy fallback used by tools such as cURL and gRPC.

### Request Flow for Plain HTTP
For unencrypted requests (e.g. `GET http://example.com/api/v1/users`), the proxy receives the target URL in the HTTP request line. The proxy normalizes headers, inspects the current operating mode, and either fetches upstream data or retrieves a local cassette recording.

### Request Flow for HTTPS CONNECT Tunnels
For TLS encrypted requests (e.g. `CONNECT api.stripe.com:443`), the client requests an opaque TCP tunnel through the proxy:

1. Client sends `CONNECT api.stripe.com:443 HTTP/1.1` to the proxy.
2. The proxy responds with `HTTP/1.1 200 Connection Established`.
3. Instead of blindly piping bytes to the remote host, the proxy hijacks the connection and initiates a TLS server handshake with the client.
4. The proxy inspects the Server Name Indication (SNI) or uses the host from the CONNECT line.
5. The proxy dynamically generates or retrieves a cached X.509 leaf certificate signed by its internal Root Certificate Authority.
6. The client completes the TLS handshake, trusting the connection because the Root CA is registered in the client trust store or environment.
7. Decrypted HTTP requests are read, processed by the proxy pipeline, and re-encrypted back to the client.

```
Client                  cassettecraft                  Upstream (Record Mode)
  |                           |                                  |
  |--- CONNECT host:443 ----->|                                  |
  |<-- 200 Connection Est. ---|                                  |
  |                           |                                  |
  |--- TLS Client Hello ----->|                                  |
  |<-- TLS Handshake (CA) ----|                                  |
  |                           |                                  |
  |--- Decrypted Request ---->|                                  |
  |    (GET /v1/charges)      |--- Forward Decrypted Request --->|
  |                           |<-- Real Response Body -----------|
  |                           |    [Sanitize & Save Cassette]    |
  |<-- Encrypted Response ----|                                  |
```

---

## 2. Dynamic TLS Certificate Authority

The module `internal/certs/ca.go` generates and manages a dedicated Root CA:

- **Key Algorithm**: RSA 2048 or ECDSA P-256 with SHA-256 signatures.
- **Persistence**: Stored in `~/.cassettecraft/ca/` by default, or generated ephemerally in memory during disposable test runs.
- **Leaf Certificate Caching**: Certificates generated for specific hosts (e.g. `api.github.com`, `*.stripe.com`) are cached in a thread-safe LRU cache with an expiration window matching the test run. This keeps per-request TLS negotiation overhead under 1 millisecond.
- **Subject Alternative Names (SAN)**: Every generated leaf certificate includes both DNS names and IP addresses matching the target host, ensuring strict TLS validation libraries do not reject the connection.

---

## 3. Cassette Matching Pipeline

When operating in `replay` mode, incoming requests pass through `internal/proxy/matcher.go`:

1. **Filtering Ignored Headers**: Ephemeral headers that change across test runs (such as `Date`, `User-Agent`, `X-Request-Id`, `Traceparent`, and `Accept-Encoding`) are stripped prior to comparison.
2. **Method and Path Matching**: Verifies the HTTP method and URI path.
3. **Query Parameter Normalization**: Query parameters are sorted alphabetically to prevent ordering mismatches from breaking tests.
4. **Body Comparison**:
   - For `application/json` payloads, the matcher parses both recorded and incoming bodies into semantic JSON trees. Key order and whitespace variations do not cause match failures.
   - For other content types (form-urlencoded, multipart, binary), byte-level or string comparison is applied.
5. **Sequential Mode (FIFO)**: When enabled, multiple requests to the same endpoint return recorded responses in sequential order, supporting polling workflows where the same endpoint transitions from pending to succeeded.

### Match Failure Diagnostics
If a request cannot be matched in replay mode, `cassettecraft` does not silently fail. It computes the closest matching interaction in the active cassette and returns an HTTP 502 response with a structured JSON body identifying the exact fields that differed (e.g. missing query parameter or altered payload attribute).

---

## 4. Memory and Concurrency Management

`cassettecraft` is designed to run in resource-constrained environments such as containerized CI jobs:

- **Target Footprint**: Under 25MB resident set size (RSS) under sustained load.
- **Buffer Recycling**: Uses `sync.Pool` for byte buffers during request copying and body sanitization to minimize GC pressure.
- **Streaming Response Piping**: Responses from upstream are streamed to the client while simultaneously filling the cassette buffer, preventing large file downloads from consuming excessive memory.
- **Zero CGo**: Compiled purely in native Go, simplifying cross-compilation across Linux, macOS, and Windows.
