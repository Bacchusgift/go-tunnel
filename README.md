# go-tunnel

极简内网穿透工具，基于 WebSocket 的反向代理。

## 架构

```
外部用户 → {prefix}.yourdomain.com → Nginx(:80) → 服务端(:8080) → WebSocket隧道 → 客户端 → localhost:{port}
```

## 快速开始

### 服务端部署（Docker Compose）

```bash
git clone git@github.com:Bacchusgift/go-tunnel.git /opt/go-tunnel
cd /opt/go-tunnel

# 配置域名
cp .env.example .env
# 编辑 .env，设置 TUNNEL_DOMAIN=yourdomain.com

# 启动
docker compose up -d
```

或使用部署脚本：

```bash
./deploy
```

脚本会自动引导你配置 `.env` 文件。

**Nginx 反代配置：**

```nginx
server {
    listen 80;
    server_name *.yourdomain.com;

    location /_tunnel/ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 客户端使用

```bash
# 编译
go build -o tunnel-client ./cmd/client

# 自动分配随机短码
./tunnel-client -port 3000 -server http://yourdomain.com/_tunnel/ws
# ✅ 隧道已建立: a3xk9m.yourdomain.com → localhost:3000

# 指定自定义前缀
./tunnel-client -port 8080 -prefix myapp -server http://yourdomain.com/_tunnel/ws
# ✅ 隧道已建立: myapp.yourdomain.com → localhost:8080

# 也可以通过环境变量配置服务端地址
export TUNNEL_SERVER=http://yourdomain.com/_tunnel/ws
./tunnel-client -port 3000
```

## 环境变量

### 服务端

| 环境变量 | CLI 参数 | 默认值 | 说明 |
|----------|----------|--------|------|
| `TUNNEL_ADDR` | `-addr` | `:8080` | 监听地址 |
| `TUNNEL_DOMAIN` | `-domain` | （必填） | 基础域名，用于子域名路由 |

### 客户端

| 环境变量 | CLI 参数 | 默认值 | 说明 |
|----------|----------|--------|------|
| `TUNNEL_SERVER` | `-server` | （必填） | 服务端 WebSocket 地址 |

### CLI 参数

**服务端：** `-addr`、`-domain`

**客户端：** `-port`（必填）、`-prefix`（可选，随机6位短码）、`-server`（必填）

## 常用命令

```bash
make              # 编译 server 和 client
make docker-build # Docker 构建
make docker-up    # Docker 启动
make docker-down  # Docker 停止
make docker-logs  # 查看日志
make clean        # 清理编译产物
```

## 特性

- 单端口设计：控制通道和代理流量共用一个端口
- Docker Compose 一键部署
- 环境变量配置，无硬编码域名
- 自动断线重连（5s 间隔）
- 心跳保活（15s 间隔，60s 超时清理）
- 请求超时（30s）

## License

MIT
