package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

//
// =====================
// 端口池（内存版）
// =====================
//

type Pool struct {
	start int
	end   int
	used  map[int]bool
	mu    sync.Mutex
}

func NewPool(start, end int) *Pool {
	return &Pool{
		start: start,
		end:   end,
		used:  make(map[int]bool),
	}
}

func (p *Pool) Allocate() (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := p.start; i <= p.end; i++ {
		if !p.used[i] {
			p.used[i] = true
			return i, true
		}
	}
	return 0, false
}

func (p *Pool) AllocateSingle(port int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.used[port] {
		return false
	}
	p.used[port] = true
	return true
}

func (p *Pool) Release(port int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.used, port)
}

//
// =====================
// frpc 管理器
// =====================
//

type Manager struct {
	cmd     *exec.Cmd
	running bool
	mu      sync.Mutex
}

func (m *Manager) Start(configPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		fmt.Println("frpc 已在运行")
		return nil
	}

	m.stopLocked()

	m.cmd = exec.Command("./frpc", "-c", configPath)

	err := m.cmd.Start()
	if err != nil {
		return err
	}

	m.running = true

	go func() {
		_ = m.cmd.Wait()
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
}

func (m *Manager) stopLocked() {
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
	}
	m.cmd = nil
	m.running = false
}

//
// =====================
// 写 frpc 配置
// =====================
//

func generateConfig(serverAddr string, serverPort int, token string, name string, localPort int, remotePort int) error {
	content := fmt.Sprintf(`
serverAddr = "%s"
serverPort = %d

auth.method = "token"
auth.token = "%s"

[[proxies]]
name = "%s"
type = "tcp"
localIP = "127.0.0.1"
localPort = %d
remotePort = %d
`, serverAddr, serverPort, token, name, localPort, remotePort)

	return os.WriteFile("frpc.toml", []byte(content), 0644)
}

//
// =====================
// JSON 配置文件结构
// =====================
//

type Config struct {
	ServerAddr string `json:"serverAddr"`
	ServerPort int    `json:"serverPort"`
	Token      string `json:"token"`
	LocalPort  int    `json:"localPort"`
	PortStart  int    `json:"portStart"`
	PortEnd    int    `json:"portEnd"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func generateSampleConfig(path string) error {
	sample := Config{
		ServerAddr: "请填写服务器地址",
		ServerPort: 7000,
		Token:      "请填写token",
		LocalPort:  25565,
		PortStart:  25565,
		PortEnd:    25575,
	}
	data, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

//
// =====================
// main
// =====================
//

func main() {

	// 命令行参数
	cfgPath := flag.String("config", "config.json", "配置文件路径")
	cliServerAddr := flag.String("server-addr", "", "覆盖 serverAddr")
	cliServerPort := flag.Int("server-port", 0, "覆盖 serverPort")
	cliToken := flag.String("token", "", "覆盖 token")
	cliLocalPort := flag.Int("local-port", 0, "覆盖 localPort")
	cliPortStart := flag.Int("port-start", 0, "覆盖端口池起始")
	cliPortEnd := flag.Int("port-end", 0, "覆盖端口池结束")
	flag.Parse()

	// 从 JSON 文件加载
	fileCfg, err := loadConfig(*cfgPath)
	if err != nil {
		fmt.Printf("未找到配置文件 %s，已自动创建示例文件，请编辑后重新运行\n", *cfgPath)
		if err := generateSampleConfig(*cfgPath); err != nil {
			fmt.Printf("创建示例文件失败: %v\n", err)
		}
		os.Exit(1)
	}

	// 使用配置文件的值，命令行参数优先覆盖
	cfg := fileCfg
	if *cliServerAddr != "" {
		cfg.ServerAddr = *cliServerAddr
	}
	if *cliServerPort != 0 {
		cfg.ServerPort = *cliServerPort
	}
	if *cliToken != "" {
		cfg.Token = *cliToken
	}
	if *cliLocalPort != 0 {
		cfg.LocalPort = *cliLocalPort
	}
	if *cliPortStart != 0 {
		cfg.PortStart = *cliPortStart
	}
	if *cliPortEnd != 0 {
		cfg.PortEnd = *cliPortEnd
	}

	// 验证必要字段
	if cfg.ServerAddr == "" || cfg.ServerPort == 0 || cfg.Token == "" {
		fmt.Println("错误: serverAddr、serverPort、token 必须在 config.json 中填写，或通过命令行参数传入")
		os.Exit(1)
	}
	if cfg.LocalPort == 0 {
		cfg.LocalPort = 25565
	}
	if cfg.PortStart == 0 {
		cfg.PortStart = 25565
	}
	if cfg.PortEnd == 0 {
		cfg.PortEnd = 25575
	}

	pool := NewPool(cfg.PortStart, cfg.PortEnd)

	tryPorts := make([]int, 0, cfg.PortEnd-cfg.PortStart+1)
	for p := cfg.PortStart; p <= cfg.PortEnd; p++ {
		tryPorts = append(tryPorts, p)
	}
	startWithRetry(cfg.ServerAddr, cfg.ServerPort, cfg.Token, cfg.LocalPort, pool, tryPorts, 0)

	// startWithRetry 返回说明尝试了所有端口都失败
	fmt.Println("所有端口尝试失败，程序退出")
	os.Exit(1)
}

func startWithRetry(serverAddr string, serverPort int, token string, localPort int, pool *Pool, tryPorts []int, idx int) {
	if idx >= len(tryPorts) {
		fmt.Println("所有端口都已尝试，均不可用")
		return
	}

	remotePort := tryPorts[idx]
	fmt.Printf("[%d/%d] 尝试端口 %d ...\n", idx+1, len(tryPorts), remotePort)
	if !pool.AllocateSingle(remotePort) {
		fmt.Printf("端口 %d 已被占用，尝试下一个...\n", remotePort)
		startWithRetry(serverAddr, serverPort, token, localPort, pool, tryPorts, idx+1)
		return
	}

	name := "mc_" + strconv.Itoa(remotePort)

	err := generateConfig(serverAddr, serverPort, token, name, localPort, remotePort)
	if err != nil {
		fmt.Println("生成配置失败:", err)
		fmt.Printf("尝试下一个端口...\n")
		pool.Release(remotePort)
		startWithRetry(serverAddr, serverPort, token, localPort, pool, tryPorts, idx+1)
		return
	}

	// 启动 frpc 并实时读取输出
	cmd := exec.Command("./frpc", "-c", "frpc.toml")

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		fmt.Printf("启动 frpc 失败: %v\n", err)
		pool.Release(remotePort)
		startWithRetry(serverAddr, serverPort, token, localPort, pool, tryPorts, idx+1)
		return
	}

	// 实时读取输出，检测 "already exists"
	detected := make(chan string, 1)
	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		scanner.Buffer(make([]byte, 64*1024), 256*1024)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			if strings.Contains(line, "already exists") {
				detected <- "already exists"
				return
			}
		}
		done <- scanner.Err()
	}()

	// 等待：检测到错误、进程退出、或超时（成功）
	var failReason string

	select {
	case <-detected:
		// 检测到 "already exists"
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		fmt.Printf("端口 %d 的代理已存在\n", remotePort)
		failReason = "代理已存在"

	case err := <-done:
		// frpc 自行退出（如登录失败）
		waitErr := cmd.Wait()
		if waitErr != nil {
			fmt.Printf("端口 %d 启动失败: %v\n", remotePort, waitErr)
		} else if err != nil {
			fmt.Printf("端口 %d 输出读取错误: %v\n", remotePort, err)
		}
		failReason = "进程退出"

	case <-time.After(5 * time.Second):
		// 5 秒内无问题，认为启动成功
		fmt.Println("启动成功！")
		fmt.Println("公网端口:", remotePort)
		fmt.Println("名称:", name)
		// 阻塞等待 frpc 退出（保持程序存活）
		_ = cmd.Wait()
		fmt.Println("frpc 已停止，程序退出")
		os.Exit(0)
	}

	// 只有失败才会走到这里
	fmt.Printf("  → 原因: %s\n", failReason)
	pool.Release(remotePort)
	startWithRetry(serverAddr, serverPort, token, localPort, pool, tryPorts, idx+1)
}