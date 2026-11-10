# Code Examples

This document demonstrates how to configure applications in Go, Python, Node.js, and cURL to record and replay HTTP and HTTPS traffic through cassettecraft.

---

## 1. Go Example

Standard Go `http.Client` instances automatically read `HTTP_PROXY` and `HTTPS_PROXY` from the environment when using `http.DefaultTransport`:

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	// Standard client respects HTTP_PROXY and HTTPS_PROXY
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://api.github.com/zen", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", "Bearer ghp_example1234567890abcdef")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\nResponse: %s\n", resp.StatusCode, string(body))
}
```

Run with `cassettecraft`:

```bash
# Record
cassettecraft exec --mode record --cassette ./cassettes/zen.yaml -- go run main.go

# Replay
cassettecraft exec --mode replay --cassette ./cassettes/zen.yaml -- go run main.go
```

---

## 2. Python Example (requests and httpx)

Python libraries like `requests` and `httpx` respect standard proxy variables and `SSL_CERT_FILE`:

```python
import os
import requests

def test_stripe_customer_creation():
    url = "https://api.stripe.com/v1/customers"
    headers = {
        "Authorization": "Bearer sk_test_51MockKeyForIntegrationTest123",
        "Content-Type": "application/x-www-form-urlencoded"
    }
    data = {
        "name": "Integration User",
        "email": "test@example.com"
    }

    response = requests.post(url, headers=headers, data=data)
    assert response.status_code == 200
    res_json = response.json()
    assert res_json["email"] == "test@example.com"
```

Execute via pytest:

```bash
cassettecraft exec --mode record --cassette tests/cassettes/stripe_cust.yaml -- pytest
```

---

## 3. Node.js Example (fetch and axios)

Node.js 18+ native `fetch` uses undici, which reads `HTTP_PROXY` and `HTTPS_PROXY` with the `undici.EnvHttpProxyAgent`:

```javascript
import { fetch } from 'undici';

async function fetchUserData() {
  const response = await fetch('https://api.github.com/users/octocat', {
    headers: {
      'User-Agent': 'CassetteCraft-Test-Runner'
    }
  });

  const data = await response.json();
  console.log('Login:', data.login);
}

fetchUserData();
```

Run with Node:

```bash
cassettecraft exec --mode replay --cassette cassettes/github_user.yaml -- node index.js
```

---

## 4. cURL and Shell Scripts

```bash
# Record request
cassettecraft exec --mode record --cassette ./curl_test.yaml -- \
  curl -s https://api.github.com/zen

# Replay request
cassettecraft exec --mode replay --cassette ./curl_test.yaml -- \
  curl -s https://api.github.com/zen
```
