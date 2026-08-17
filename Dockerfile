FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /node ./cmd/node

FROM alpine:latest

# Create non-root user and group (finding #22)
RUN addgroup -S cw && adduser -S -G cw -u 10001 cw

WORKDIR /app

COPY --from=builder /node /app/node

RUN mkdir -p /data && chown -R cw:cw /data /app

USER cw

EXPOSE 9000

ENTRYPOINT ["/app/node"]
