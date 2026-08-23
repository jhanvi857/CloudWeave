FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /node ./cmd/node

FROM alpine:latest

# Install ca-certificates for TLS and tzdata for logging timezones
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user and group
RUN addgroup -S cw -g 10001 && adduser -S -G cw -u 10001 -h /home/cw cw

WORKDIR /app

COPY --from=builder /node /app/node

RUN mkdir -p /data && chown -R cw:cw /data /app

# Metadata labels (OCI Standard)
LABEL org.opencontainers.image.title="CloudWeave" \
      org.opencontainers.image.description="S3-compatible distributed object store for local development and distributed systems learning" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.licenses="MIT"

USER cw

VOLUME ["/data"]

EXPOSE 9000

# Built-in healthcheck for Docker and container orchestrators
HEALTHCHECK --interval=15s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:9000/health || exit 1

ENTRYPOINT ["/app/node"]

