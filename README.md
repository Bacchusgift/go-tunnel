# go-tunnel

极简内网穿透工具，基于 WebSocket 的反向代理。

## 架构

```
外部用户 → {prefix}.autowired.cn → Nginx(:80) → 服务端(:8080) → WebSocket隧道 → 客户端 → localhost:{port}
```

## 快速开始

### 服务端部署（Docker Compose）

在服务器上执行：

```bash
curl -fsSL git@github.com:Bacchusgift/go-tunnel/raw/main/deploy | bash
```

或手动：

```bash
git clone git@github.com:Bacchusgift/go-tunnel.git /opt/go-tunnel
cd /opt/go-tunnel
docker compose up -d
```

服务监听 `:8080`，Nginx 反代配置参考：

```nginx
server {
    listen 80;
    server_name *.autowired.cn;

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

**Docker 方式：**
```bash
# 自动分配随机短码
docker run --rm -p 3000:3000 ghcr.io/bacchusgift/go-tunnel-client -port 3000

# 指定自定义前缀
docker run --rm ghcr.io/bacchusgift/go-tunnel-client -port 3000 -prefix myapp
```

**本地编译：**
```bash
make build
./bin/go-tunnel-client -port 3000
# ✅ 隧道已建立: a3xk9m.autowired.cn → localhost:3000
```

## CLI 参数

### 服务端

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-addr` | `:8080` | 监听地址 |
| `-domain` | `autowired.cn` | 基础域名（用于子域名路由） |

### 客户端

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-port` | - | 本地反代端口（必填） |
| `-prefix` | 随机6位短码 | 域名前缀 |
| `-server` | `http://proxy.autowired.cn/_tunnel/ws` | 服务端 WebSocket 地址 |

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
- 自动断线重连（5s 间隔）
- 心跳保活（15s 间隔，60s 超时清理）
- 请求超时（30s）
- 请求体 Base64 编码传输
- 无鉴权，适合自用

## License

MIT
