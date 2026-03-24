FROM golang:1.26-alpine AS builder
WORKDIR /app

# Cache dependency download layer separately from source
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server .

FROM alpine:3.21
RUN apk add --no-cache ffmpeg ca-certificates tzdata

COPY --from=builder /app/server /usr/local/bin/server

# Non-root user for runtime
RUN addgroup -S screenspace && adduser -S screenspace -G screenspace
USER screenspace

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["server"]
