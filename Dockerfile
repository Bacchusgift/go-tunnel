FROM golang:1.23-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

# 替换 Alpine 国内源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /go-tunnel-server ./cmd/server

FROM alpine:3.19

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates tzdata

COPY --from=builder /go-tunnel-server /usr/local/bin/go-tunnel-server

EXPOSE 8080

ENTRYPOINT ["go-tunnel-server"]
CMD ["-addr", ":8080"]
