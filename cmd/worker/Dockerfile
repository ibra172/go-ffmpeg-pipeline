FROM golang:1.25.13-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/worker ./cmd/worker

FROM alpine:3.23 AS runner
WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata ffmpeg

COPY --from=builder /app/worker .
CMD ["./worker"]
