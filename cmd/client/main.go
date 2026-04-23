package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Bacchusgift/go-tunnel/internal/client"
)

const configDir = ".go-tunnel"
const configFile = "config.json"

type config struct {
	ServerURL string `json:"server_url"`
}

type tunnelItem struct {
	prefix string
	port   int
	domain string
	client *client.Client
}

// normalizeServerURL appends /_tunnel/ws if not already present
func normalizeServerURL(raw string) string {
	raw = strings.TrimRight(raw, "/")
	if !strings.HasSuffix(raw, "/_tunnel/ws") {
		raw += "/_tunnel/ws"
	}
	return raw
}

func main() {
	// Load config
	cfg := loadConfig()

	// If no server URL, ask once
	if cfg.ServerURL == "" {
		fmt.Println("🔧 首次使用，请配置服务端地址")
		fmt.Println()
		raw := inputString("🌐 服务器地址 (如 http://proxy.autowired.cn): ")
		if raw == "" {
			fmt.Println("❌ 地址不能为空")
			os.Exit(1)
		}
		cfg.ServerURL = normalizeServerURL(raw)
		saveConfig(cfg)
	}

	tunnels := make(map[string]*tunnelItem)
	var mu sync.Mutex

	// Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\n👋 再见")
		os.Exit(0)
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println()
		mu.Lock()
		fmt.Println("━━━━━━━━━━ go-tunnel ━━━━━━━━━━")
		fmt.Printf("🔧 服务端: %s\n", cfg.ServerURL)
		fmt.Println("──────────────────────────────")
		if len(tunnels) == 0 {
			fmt.Println("  (暂无活跃隧道)")
		}
		for _, t := range tunnels {
			fmt.Printf("  ✅ %s → localhost:%d\n", t.domain, t.port)
		}
		fmt.Println("──────────────────────────────")
		fmt.Println("  [1] 创建隧道")
		fmt.Println("  [2] 关闭隧道")
		fmt.Println("  [3] 修改服务端地址")
		fmt.Println("  [0] 退出")
		mu.Unlock()

		fmt.Print("\n请选择: ")
		line, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(line)

		switch choice {
		case "1":
			// Create tunnel
			fmt.Println()
			fmt.Print("📌 输入域名前缀 (直接回车=随机生成): ")
			prefix, _ := reader.ReadString('\n')
			prefix = strings.TrimSpace(prefix)

			fmt.Print("🔌 本地端口号: ")
			portStr, _ := reader.ReadString('\n')
			portStr = strings.TrimSpace(portStr)
			port, err := strconv.Atoi(portStr)
			if err != nil || port <= 0 || port > 65535 {
				fmt.Println("❌ 端口无效")
				continue
			}

			c := client.New(cfg.ServerURL, port, prefix)
			t := &tunnelItem{
				prefix: prefix,
				port:   port,
				client: c,
			}

			if prefix != "" {
				tunnels[prefix] = t
			}

			c.OnRegistered(func(domain string) {
				mu.Lock()
				defer mu.Unlock()
				t.domain = domain
				if idx := strings.Index(domain, "."); idx > 0 {
					t.prefix = domain[:idx]
					if prefix == "" {
						tunnels[t.prefix] = t
					}
				}
			})

			// Connect in background
			go func() {
				for {
					err := c.Connect()
					if err != nil {
						mu.Lock()
						if t.domain != "" {
							fmt.Printf("\n❌ 隧道断开: %s (%v)\n", t.domain, err)
							delete(tunnels, t.prefix)
						} else {
							fmt.Printf("\n❌ 连接失败: %v\n", err)
						}
						mu.Unlock()
					}
					// Auto reconnect after 5s
					time.Sleep(5 * time.Second)
				}
			}()

			// Wait for registration
			select {
			case <-c.Registered():
				fmt.Printf("✅ 已建立: %s → localhost:%d\n", t.domain, t.port)
			case <-time.After(5 * time.Second):
				fmt.Println("❌ 连接超时，请检查服务端地址")
			}

		case "2":
			mu.Lock()
			if len(tunnels) == 0 {
				fmt.Println("  (暂无隧道)")
				mu.Unlock()
				continue
			}
			fmt.Println()
			i := 1
			keys := make([]string, 0, len(tunnels))
			for _, t := range tunnels {
				fmt.Printf("  [%d] %s → localhost:%d\n", i, t.domain, t.port)
				keys = append(keys, t.prefix)
				i++
			}
			mu.Unlock()

			fmt.Print("\n选择要关闭的隧道: ")
			line, _ := reader.ReadString('\n')
			idx, err := strconv.Atoi(strings.TrimSpace(line))
			if err != nil || idx < 1 || idx > len(keys) {
				fmt.Println("❌ 无效选择")
				continue
			}

			key := keys[idx-1]
			mu.Lock()
			if t, ok := tunnels[key]; ok {
				t.client.Close()
				fmt.Printf("✅ 已关闭: %s\n", t.domain)
				delete(tunnels, key)
			}
			mu.Unlock()

		case "3":
			fmt.Println()
			// 显示用户输入的原始地址（去掉路径）
			displayURL := strings.TrimSuffix(cfg.ServerURL, "/_tunnel/ws")
			fmt.Printf("当前地址: %s\n", displayURL)
			fmt.Print("新地址 (直接回车=不修改): ")
			newURL, _ := reader.ReadString('\n')
			newURL = strings.TrimSpace(newURL)
			if newURL != "" {
				normalized := normalizeServerURL(newURL)
				if normalized != cfg.ServerURL {
					cfg.ServerURL = normalized
					saveConfig(cfg)
					fmt.Println("✅ 已保存")
				}
			}

		case "0", "q", "quit":
			mu.Lock()
			for _, t := range tunnels {
				t.client.Close()
			}
			mu.Unlock()
			fmt.Println("👋 再见")
			return

		default:
			fmt.Println("❓ 无效选择")
		}
	}
}

func inputString(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return configDir
	}
	return home + "/" + configDir
}

func loadConfig() config {
	path := getConfigPath() + "/" + configFile
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}
	}
	return cfg
}

func saveConfig(cfg config) {
	dir := getConfigPath()
	os.MkdirAll(dir, 0755)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(dir+"/"+configFile, data, 0644)
}
