# ============================================================
# 方式1（推荐）：宿主机已装 Go，deploy 脚本直接编译二进制
#   使用 Dockerfile（单阶段，只 COPY 二进制进去）
#   ./deploy
#
# 方式2（备选）：宿主机没有 Go，用 Docker 内编译
#   使用 Dockerfile.build（多阶段，需要能拉 golang 镜像）
#   docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
# ============================================================

# Pre-compile: GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/go-tunnel-server ./cmd/server
# Then: docker compose up -d

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY bin/go-tunnel-server /usr/local/bin/go-tunnel-server

EXPOSE 8080

ENTRYPOINT ["go-tunnel-server"]
CMD ["-addr", ":8080"]
