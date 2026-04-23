package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Bacchusgift/go-tunnel/internal/client"
)

type tunnel struct {
	prefix string
	port   int
	domain string
	client *client.Client
}

func main() {
	serverURL := os.Getenv("TUNNEL_SERVER")
	if serverURL == "" {
		fmt.Print("🔐 服务端地址: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		serverURL = strings.TrimSpace(line)
		if serverURL == "" {
			fmt.Println("❌ 服务端地址不能为空")
			os.Exit(1)
		}
	}

	fmt.Println()
	fmt.Println("🔒 go-tunnel 交互模式")
	fmt.Println("  connect [prefix] <port>  创建隧道")
	fmt.Println("  list                    查看隧道")
	fmt.Println("  close <prefix>           关闭隧道")
	fmt.Println("  quit                    退出")
	fmt.Println()

	tunnels := make(map[string]*tunnel)
	var mu sync.Mutex
	nextID := 0

	// Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\n👋 再见")
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "connect", "c":
			if len(parts) < 2 {
				fmt.Println("❌ 用法: connect [prefix] <port>")
				continue
			}

			var prefix string
			var port int
			switch len(parts) {
			case 2:
				// connect <port>
				fmt.Sscanf(parts[1], "%d", &port)
			case 3:
				// connect <prefix> <port>
				prefix = parts[1]
				fmt.Sscanf(parts[2], "%d", &port)
			default:
				fmt.Println("❌ 用法: connect [prefix] <port>")
				continue
			}

			if port <= 0 || port > 65535 {
				fmt.Println("❌ 端口无效 (1-65535)")
				continue
			}

			key := prefix
			if key == "" {
				key = fmt.Sprintf("_pending_%d", nextID)
				nextID++
			}

			c := client.New(serverURL, port, prefix)
			t := &tunnel{
				prefix: prefix,
				port:   port,
				client: c,
			}

			c.OnRegistered(func(domain string) {
				mu.Lock()
				defer mu.Unlock()
				t.domain = domain
				if idx := strings.Index(domain, "."); idx > 0 {
					t.prefix = domain[:idx]
				}
				tunnels[t.prefix] = t
				// remove pending key
				delete(tunnels, key)
			})

			mu.Lock()
			tunnels[key] = t
			mu.Unlock()

			// wait for registered or timeout
			connected := make(chan error, 1)
			go func() {
				connected <- c.Connect()
			}()

			select {
			case <-c.Registered():
				fmt.Printf("✅ %s → localhost:%d\n", t.domain, t.port)
			case err := <-connected:
				fmt.Printf("❌ 连接失败: %v\n", err)
				mu.Lock()
				delete(tunnels, key)
				mu.Unlock()
			case <-time.After(5 * time.Second):
				fmt.Println("❌ 连接超时")
				c.Close()
				mu.Lock()
				delete(tunnels, key)
				mu.Unlock()
			}

		case "list", "ls":
			mu.Lock()
			if len(tunnels) == 0 {
				fmt.Println("  (暂无隧道)")
			}
			for _, t := range tunnels {
				if t.domain != "" {
					fmt.Printf("  ✅ %s → localhost:%d\n", t.domain, t.port)
				}
			}
			mu.Unlock()

		case "close", "d":
			if len(parts) < 2 {
				fmt.Println("❌ 用法: close <prefix>")
				continue
			}
			prefix := parts[1]
			mu.Lock()
			if t, ok := tunnels[prefix]; ok {
				t.client.Close()
				fmt.Printf("❌ 已关闭: %s\n", t.domain)
				delete(tunnels, prefix)
			} else {
				fmt.Printf("❌ 未找到: %s\n", prefix)
			}
			mu.Unlock()

		case "quit", "exit", "q":
			mu.Lock()
			for _, t := range tunnels {
				t.client.Close()
			}
			mu.Unlock()
			fmt.Println("👋 再见")
			return

		default:
			fmt.Println("❓ 未知命令")
		}
	}
}
