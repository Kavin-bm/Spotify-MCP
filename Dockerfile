FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o spotify-mcp .

# ── runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/spotify-mcp .

EXPOSE 8080

ENTRYPOINT ["./spotify-mcp"]
