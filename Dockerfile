# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /7relay ./cmd/7relay

# Final stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /7relay /usr/local/bin/7relay

# Config directory
RUN mkdir -p /root/.7relay

EXPOSE 20128

ENTRYPOINT ["7relay"]
