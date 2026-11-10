# CI/CD Integration Guide

This guide describes how to run tests with cassettecraft in continuous integration environments such as GitHub Actions, GitLab CI, and Docker.

---

## 1. Running Tests in Replay Mode

In CI/CD, the goal is 100% deterministic test execution with zero external network dependencies:

1. Tests execute in `replay` mode against checked-in cassettes.
2. Any unexpected or unrecorded outbound API call fails the build immediately.
3. Tests run in parallel without triggering third-party rate limits.

---

## 2. GitHub Actions Integration

Use this workflow to run tests with `cassettecraft`:

```yaml
name: Test Suite

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true

      - name: Install cassettecraft
        run: |
          go install github.com/alexandrmotologa/cassettecraft/cmd/cassettecraft@latest

      - name: Run Test Suite with Cassette Replay
        run: |
          cassettecraft exec \
            --mode replay \
            --cassette-dir ./tests/cassettes \
            -- go test -v ./tests/...
```

---

## 3. GitLab CI Integration

In `.gitlab-ci.yml`:

```yaml
stages:
  - test

test_job:
  stage: test
  image: golang:1.23-alpine
  before_script:
    - apk add --no-cache git gcc musl-dev
    - go install github.com/alexandrmotologa/cassettecraft/cmd/cassettecraft@latest
  script:
    - cassettecraft exec --mode replay --cassette-dir ./tests/cassettes -- go test -v ./...
  variables:
    CGO_ENABLED: "0"
```

---

## 4. Docker Container Integration

Run `cassettecraft` alongside your test container via Docker Compose:

```yaml
services:
  cassettecraft:
    image: alexandrmotologa/cassettecraft:latest
    command: ["start", "--mode", "replay", "--cassette-dir", "/cassettes"]
    volumes:
      - ./tests/cassettes:/cassettes:ro
    ports:
      - "8585:8585"

  app-tests:
    build: .
    depends_on:
      - cassettecraft
    environment:
      - HTTP_PROXY=http://cassettecraft:8585
      - HTTPS_PROXY=http://cassettecraft:8585
    command: ["pytest", "tests/"]
```

---

## 5. Detecting Missing or Outdated Cassettes

When an unrecorded request is made in `replay` mode:
- The proxy returns `HTTP 502 Bad Gateway`.
- The response header `X-CassetteCraft-Status: NO_MATCH` is set.
- The response body contains JSON explaining which endpoint was called and why no existing cassette matched.

This ensures test failures pinpoint the exact missing recording rather than failing with confusing timeouts or generic connection errors.
