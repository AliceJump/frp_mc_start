package server

import (
    "encoding/json"
    "net/http"
)

type Proxy struct {
    Name       string `json:"name"`
    RemotePort int    `json:"remote_port"`
}

func GetProxies(api string) ([]Proxy, error) {
    resp, err := http.Get(api + "/api/proxies")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Proxies []Proxy `json:"proxies"`
    }

    json.NewDecoder(resp.Body).Decode(&result)
    return result.Proxies, nil
}
