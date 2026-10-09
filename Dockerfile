# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=mod -ldflags="-w -s" -o /app/bin/bot ./cmd/bot

# Stage 2: Minimal runtime image (<25MB)
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/bot /app/bin/bot

ENV ENV=production

CMD ["/app/bin/bot"]
