package port

import "sync"

type Pool struct {
    mu    sync.Mutex
    start  int
    end    int
    used   map[int]bool
}

func NewPool(start, end int) *Pool {
    return &Pool{
        start: start,
        end:   end,
        used:  make(map[int]bool),
    }
}

// 分配端口
func (p *Pool) Allocate() (int, bool) {
    p.mu.Lock()
    defer p.mu.Unlock()

    for port := p.start; port <= p.end; port++ {
        if !p.used[port] {
            p.used[port] = true
            return port, true
        }
    }
    return 0, false
}

// 释放端口
func (p *Pool) Release(port int) {
    p.mu.Lock()
    defer p.mu.Unlock()
    delete(p.used, port)
}
