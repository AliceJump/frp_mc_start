package frpc

import (
    "fmt"
    "os/exec"
    "sync"
)

type Manager struct {
    cmd     *exec.Cmd
    running bool
    mu      sync.Mutex
}

// 启动（自动先停旧的）
func (m *Manager) Start(configPath string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.running {
        fmt.Println("frpc 已在运行，跳过")
        return nil
    }

    m.stopLocked()

    m.cmd = exec.Command("./frpc", "-c", configPath)

    if err := m.cmd.Start(); err != nil {
        return err
    }

    m.running = true

    go func() {
        m.cmd.Wait()

        m.mu.Lock()
        m.running = false
        m.mu.Unlock()
    }()

    return nil
}

// 停止（对外调用）
func (m *Manager) Stop() {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.stopLocked()
}

// 内部停止逻辑
func (m *Manager) stopLocked() {
    if m.cmd != nil && m.cmd.Process != nil {
        _ = m.cmd.Process.Kill()
        _, _ = m.cmd.Process.Wait()
    }
    m.cmd = nil
    m.running = false
}