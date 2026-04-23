# go-tunnel

极简内网穿透工具，基于 WebSocket 的反向代理。

## 架构

```
外部用户 → {prefix}.autowired.cn → Nginx(:80) → 服务端(:8080) → WebSocket隧道 → 客户端 → localhost:{port}
```

## 快速开始

### 编译

```bash
make          # 构建 server 和 client（当前平台）
make linux    # 交叉编译 linux/amd64
make darwin   # 交叉编译 darwin/amd64 + darwin/arm64
```

### 服务端部署

```bash
./server -addr :8080 -domain autowired.cn
```

Nginx 配置参考：

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

```bash
# 自动分配随机短码
./client -port 3000
# ✅ 隧道已建立: a3xk9m.autowired.cn → localhost:3000

# 指定自定义前缀
./client -prefix myapp -port 8080
# ✅ 隧道已建立: myapp.autowired.cn → localhost:8080

# 指定自定义服务端地址
./client -port 3000 -server http://your-server.com/_tunnel/ws
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

## 特性

- 单端口设计：控制通道和代理流量共用一个端口
- 自动断线重连（5s 间隔）
- 心跳保活（15s 间隔，60s 超时清理）
- 请求超时（30s）
- 请求体 Base64 编码传输
- 无鉴权，适合自用

## License

MIT
