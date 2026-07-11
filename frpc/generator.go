package frpc

import (
    "fmt"
    "os"
)

type Config struct {
    ServerAddr string
    ServerPort int
    Token      string
    LocalPort  int
    RemotePort int
}

func Generate(cfg Config, path string) error {
    name := fmt.Sprintf("mc_%d", cfg.RemotePort)

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
`, cfg.ServerAddr, cfg.ServerPort, cfg.Token, name, cfg.LocalPort, cfg.RemotePort)

    return os.WriteFile(path, []byte(content), 0644)
}
