FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /node ./cmd/node

FROM alpine:latest

WORKDIR /

COPY --from=builder /node /node

EXPOSE 8080

ENTRYPOINT ["/node"]
