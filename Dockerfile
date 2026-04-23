# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /go-tunnel-server ./cmd/server

# Run stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /go-tunnel-server /usr/local/bin/go-tunnel-server

EXPOSE 8080

ENTRYPOINT ["go-tunnel-server"]
CMD ["-addr", ":8080", "-domain", "autowired.cn"]
