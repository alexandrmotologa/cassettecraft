# Secret Redaction Guide

This document describes how cassettecraft detects, scrubs, and redacts sensitive credentials and personal data before saving interactions to disk.

---

## 1. Security Invariant

Test cassettes are frequently checked into version control repositories. If an API key or session token is recorded into a cassette, it risks leaking into public or shared git history. `cassettecraft` runs a multi-stage redaction pipeline on both the request and response before writing to disk.

---

## 2. Default Redaction Rules

### 2.1 Header Redaction
The following headers are automatically scrubbed:

| Header Name | Replacement Value | Reason |
| :--- | :--- | :--- |
| `Authorization` | `Bearer [REDACTED_TOKEN]` | Bearer tokens, OAuth access tokens, API tokens |
| `Proxy-Authorization` | `[REDACTED_PROXY_AUTH]` | Upstream proxy credentials |
| `Cookie` | `[REDACTED_COOKIE]` | Session cookies |
| `Set-Cookie` | `[REDACTED_SET_COOKIE]` | Session cookies set by response |
| `X-Api-Key` | `[REDACTED_API_KEY]` | Custom API key headers |
| `Stripe-Signature` | `[REDACTED_SIGNATURE]` | Webhook verification signatures |
| `X-Hub-Signature-256` | `[REDACTED_SIGNATURE]` | GitHub webhook signatures |

### 2.2 Query Parameter Redaction
URL query parameters matching sensitive keys are masked in recorded URLs:
- `api_key`
- `apikey`
- `token`
- `access_token`
- `secret`
- `signature`
- `auth`

### 2.3 Regex Token Scanners
Request bodies, URLs, and response bodies are scanned against regex patterns:

- **Stripe Keys**: `sk_live_[0-9a-zA-Z]{24,}` -> `[REDACTED_STRIPE_KEY]`
- **Stripe Restricted Keys**: `rk_live_[0-9a-zA-Z]{24,}` -> `[REDACTED_STRIPE_KEY]`
- **GitHub Personal Access Tokens**: `ghp_[0-9a-zA-Z]{36}` -> `[REDACTED_GITHUB_PAT]`
- **GitHub OAuth Tokens**: `gho_[0-9a-zA-Z]{36}` -> `[REDACTED_GITHUB_TOKEN]`
- **GitHub Fine-Grained Tokens**: `github_pat_[0-9a-zA-Z_]{80,}` -> `[REDACTED_GITHUB_PAT]`
- **OpenAI API Keys**: `sk-[a-zA-Z0-9]{20,}` -> `[REDACTED_OPENAI_KEY]`
- **AWS Access Key IDs**: `AKIA[0-9A-Z]{16}` -> `[REDACTED_AWS_KEY]`
- **JSON Web Tokens**: `ey[A-Za-z0-9_-]{10,}\.ey[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}` -> `[REDACTED_JWT]`
- **Credit Card Numbers**: Luhn-passing 13 to 19 digit sequences -> `[REDACTED_CARD_NUMBER]`

### 2.4 JSON Tree Traversal
When a payload is valid JSON, the redactor performs recursive traversal. Any key matching common credential terms (case-insensitive) has its value replaced with `"[REDACTED]"`:
- `password`
- `passphrase`
- `client_secret`
- `clientSecret`
- `private_key`
- `privateKey`
- `cvv`
- `cvc`
- `pin`

---

## 3. Custom Redaction Configuration

You can customize rules by placing a `cassettecraft.yaml` file in your repository root:

```yaml
redaction:
  # Additional headers to scrub
  headers:
    - "X-Internal-Token"
    - "X-Tenant-Secret"

  # Additional query parameters to mask in URLs
  query_params:
    - "session_id"
    - "temp_key"

  # Custom JSON keys to redact recursively
  json_keys:
    - "ssn"
    - "national_id"
    - "tax_identifier"

  # Custom regex patterns
  patterns:
    - name: "custom_company_token"
      regex: "corp_[0-9a-f]{32}"
      replacement: "[REDACTED_CORP_TOKEN]"
```
