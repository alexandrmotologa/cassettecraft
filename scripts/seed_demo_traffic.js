const http = require('http');

const sampleInteractions = [
  {
    id: "live_stripe_charge",
    timestamp: new Date(Date.now() - 42000).toISOString(),
    mode: "replay",
    status: 200,
    method: "POST",
    url: "https://api.stripe.com/v1/payment_intents",
    host: "api.stripe.com",
    path: "/v1/payment_intents",
    req_headers: {
      "Authorization": "Bearer [REDACTED_STRIPE_SECRET]",
      "Content-Type": "application/x-www-form-urlencoded",
      "User-Agent": "Stripe/v1 GoBindings/v74.0.0"
    },
    req_body: "amount=4999&currency=usd&customer=cus_Nx9K2L91&automatic_payment_methods%5Benabled%5D=true",
    res_headers: {
      "Content-Type": "application/json",
      "Request-Id": "req_str_98124019284",
      "X-CassetteCraft": "REPLAY"
    },
    res_body: JSON.stringify({
      id: "pi_3NqE4mLkdF2",
      object: "payment_intent",
      amount: 4999,
      amount_received: 4999,
      currency: "usd",
      status: "succeeded",
      customer: "cus_Nx9K2L91",
      payment_method: "pm_1NqE4mLkdF28",
      charges: {
        total_count: 1,
        data: [{ id: "ch_3NqE4m", paid: true, status: "succeeded" }]
      }
    }, null, 2),
    latency_ms: 0,
    matched: true,
    cassette: "stripe_checkout.yaml"
  },
  {
    id: "live_openai_stream",
    timestamp: new Date(Date.now() - 36000).toISOString(),
    mode: "replay",
    status: 200,
    method: "POST",
    url: "https://api.openai.com/v1/chat/completions",
    host: "api.openai.com",
    path: "/v1/chat/completions",
    req_headers: {
      "Authorization": "Bearer [REDACTED_OPENAI_KEY]",
      "Content-Type": "application/json"
    },
    req_body: JSON.stringify({
      model: "gpt-4o",
      messages: [{ role: "user", content: "Explain microsecond deterministic replay" }],
      stream: true
    }, null, 2),
    res_headers: {
      "Content-Type": "text/event-stream; charset=utf-8",
      "X-CassetteCraft": "REPLAY_STREAMING"
    },
    res_body: 'data: {"choices":[{"delta":{"content":"CassetteCraft replaces flaky network calls with deterministic local YAML cassettes."}}]}\n\ndata: [DONE]\n\n',
    latency_ms: 1,
    matched: true,
    cassette: "openai_streaming.yaml"
  },
  {
    id: "live_github_repos",
    timestamp: new Date(Date.now() - 30000).toISOString(),
    mode: "replay",
    status: 200,
    method: "GET",
    url: "https://api.github.com/user/repos?per_page=5&sort=updated",
    host: "api.github.com",
    path: "/user/repos",
    req_headers: {
      "Accept": "application/vnd.github.v3+json",
      "Authorization": "token [REDACTED_GITHUB_PAT]",
      "User-Agent": "CassetteCraft/1.0"
    },
    req_body: "",
    res_headers: {
      "Content-Type": "application/json; charset=utf-8",
      "X-RateLimit-Limit": "5000",
      "X-RateLimit-Remaining": "4999",
      "X-CassetteCraft": "REPLAY"
    },
    res_body: JSON.stringify([
      { id: 9812451, name: "cassettecraft", full_name: "alexandrmotologa/cassettecraft", private: false },
      { id: 8941203, name: "idemgate", full_name: "alexandrmotologa/idemgate", private: false }
    ], null, 2),
    latency_ms: 0,
    matched: true,
    cassette: "github_sync.yaml"
  },
  {
    id: "live_github_ratelimit",
    timestamp: new Date(Date.now() - 25000).toISOString(),
    mode: "replay",
    status: 200,
    method: "GET",
    url: "https://api.github.com/rate_limit",
    host: "api.github.com",
    path: "/rate_limit",
    req_headers: {
      "Authorization": "token [REDACTED_GITHUB_PAT]",
      "User-Agent": "Octokit.js"
    },
    req_body: "",
    res_headers: {
      "Content-Type": "application/json",
      "X-CassetteCraft": "REPLAY"
    },
    res_body: JSON.stringify({
      resources: {
        core: { limit: 5000, remaining: 4999, reset: 1731260800 },
        search: { limit: 30, remaining: 30, reset: 1731257260 }
      }
    }, null, 2),
    latency_ms: 0,
    matched: true,
    cassette: "github_sync.yaml"
  },
  {
    id: "live_auth0_token",
    timestamp: new Date(Date.now() - 20000).toISOString(),
    mode: "record",
    status: 200,
    method: "POST",
    url: "https://auth.acme.corp/oauth/token",
    host: "auth.acme.corp",
    path: "/oauth/token",
    req_headers: {
      "Content-Type": "application/json"
    },
    req_body: JSON.stringify({
      client_id: "client_9182371",
      client_secret: "[REDACTED_SECRET]",
      audience: "https://api.acme.corp/v1",
      grant_type: "client_credentials"
    }, null, 2),
    res_headers: {
      "Content-Type": "application/json",
      "X-CassetteCraft": "RECORD"
    },
    res_body: JSON.stringify({
      access_token: "[REDACTED_JWT_TOKEN]",
      token_type: "Bearer",
      expires_in: 86400,
      scope: "read:records write:cassettes"
    }, null, 2),
    latency_ms: 142,
    matched: false,
    cassette: "auth_suite.yaml"
  },
  {
    id: "live_slack_webhook",
    timestamp: new Date(Date.now() - 15000).toISOString(),
    mode: "replay",
    status: 200,
    method: "POST",
    url: "https://hooks.slack.com/services/T00/B00/X00",
    host: "hooks.slack.com",
    path: "/services/T00/B00/X00",
    req_headers: {
      "Content-Type": "application/json"
    },
    req_body: JSON.stringify({
      channel: "#ci-alerts",
      username: "CassetteCraft Bot",
      text: "Integration test suite finished: 84 tests passed in 1.4s (100% offline replay)"
    }, null, 2),
    res_headers: {
      "Content-Type": "text/html",
      "X-CassetteCraft": "REPLAY"
    },
    res_body: "ok",
    latency_ms: 0,
    matched: true,
    cassette: "slack_notifications.yaml"
  },
  {
    id: "live_stripe_404",
    timestamp: new Date(Date.now() - 10000).toISOString(),
    mode: "replay",
    status: 404,
    method: "GET",
    url: "https://api.stripe.com/v1/customers/cus_non_existent",
    host: "api.stripe.com",
    path: "/v1/customers/cus_non_existent",
    req_headers: {
      "Authorization": "Bearer [REDACTED_STRIPE_SECRET]"
    },
    req_body: "",
    res_headers: {
      "Content-Type": "application/json",
      "X-CassetteCraft": "REPLAY"
    },
    res_body: JSON.stringify({
      error: {
        code: "resource_missing",
        doc_url: "https://stripe.com/docs/error-codes/resource-missing",
        message: "No such customer: 'cus_non_existent'",
        type: "invalid_request_error"
      }
    }, null, 2),
    latency_ms: 0,
    matched: true,
    cassette: "stripe_checkout.yaml"
  },
  {
    id: "live_weather_api",
    timestamp: new Date(Date.now() - 5000).toISOString(),
    mode: "auto",
    status: 200,
    method: "GET",
    url: "https://api.weather.com/v3/wx/conditions/current?apiKey=[REDACTED]&format=json",
    host: "api.weather.com",
    path: "/v3/wx/conditions/current",
    req_headers: {
      "Accept": "application/json"
    },
    req_body: "",
    res_headers: {
      "Content-Type": "application/json",
      "X-CassetteCraft": "RECORD_AUTO"
    },
    res_body: JSON.stringify({
      temperature: 18,
      condition: "Clear",
      humidity: 55,
      windSpeed: 12
    }, null, 2),
    latency_ms: 88,
    matched: true,
    cassette: "default.yaml"
  }
];

const postData = JSON.stringify(sampleInteractions);
const req = http.request({
  hostname: 'localhost',
  port: 8585,
  path: '/api/v1/interactions/seed',
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Content-Length': Buffer.byteLength(postData)
  }
}, (res) => {
  let body = '';
  res.on('data', chunk => body += chunk);
  res.on('end', () => {
    console.log('Seed response:', body);
  });
});

req.on('error', (e) => {
  console.error('Seed error:', e.message);
});

req.write(postData);
req.end();
