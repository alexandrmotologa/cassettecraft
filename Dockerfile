# Build stage for web studio
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Build stage for Go binary
FROM golang:1.23-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /cassettecraft ./cmd/cassettecraft

# Final minimal scratch runtime (< 20MB)
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /cassettecraft /usr/local/bin/cassettecraft

EXPOSE 8585
VOLUME ["/cassettes"]
ENTRYPOINT ["cassettecraft"]
CMD ["start", "--port", "8585", "--mode", "replay", "--cassette-dir", "/cassettes"]
