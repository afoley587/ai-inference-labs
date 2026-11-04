// Package gateway forwards one inference HTTP endpoint to a configured backend.
// This lab deliberately does not validate or parse OpenAI-compatible payloads.
package gateway

import (
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"
)

type Handler struct {
    target *url.URL
    client *http.Client
}

func New(rawURL string) (*Handler, error) {
    u, err := url.Parse(rawURL)
    if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
        return nil, fmt.Errorf("invalid backend URL: use an http(s) origin without credentials, query or fragment")
    }
    if u.Path != "" && u.Path != "/" {
        return nil, fmt.Errorf("backend URL must not include a path")
    }
    transport := http.DefaultTransport.(*http.Transport).Clone()
    transport.ResponseHeaderTimeout = 10 * time.Second
    return &Handler{target: u, client: &http.Client{Transport: transport}}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/healthz" && r.Method == http.MethodGet {
        w.Header().Set("Content-Type", "text/plain; charset=utf-8")
        _, _ = io.WriteString(w, "ok\n")
        return
    }
    if r.URL.Path != "/v1/chat/completions" {
        http.NotFound(w, r)
        return
    }
    if r.Method != http.MethodPost {
        w.Header().Set("Allow", "POST")
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    target := *h.target
    target.Path = r.URL.Path
    target.RawQuery = r.URL.RawQuery
    upstream, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target.String(), r.Body)
    if err != nil {
        http.Error(w, "failed to construct upstream request", http.StatusBadRequest)
        return
    }
    for _, name := range []string{"Content-Type", "Authorization", "Accept"} {
        if value := r.Header.Get(name); value != "" {
            upstream.Header.Set(name, value)
        }
    }

    resp, err := h.client.Do(upstream)
    if err != nil {
        http.Error(w, "upstream unavailable", http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()

    for _, name := range []string{"Content-Type", "Cache-Control", "X-Request-Id"} {
        if value := resp.Header.Get(name); value != "" {
            w.Header().Set(name, value)
        }
    }
    w.Header().Set("X-Content-Type-Options", "nosniff")
    // Streaming through reverse proxies is deployment-specific; this hint is
    // understood by some proxies, including nginx.
    if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("X-Accel-Buffering", "no")
    }
    w.WriteHeader(resp.StatusCode)
    flusher, canFlush := w.(http.Flusher)
    buf := make([]byte, 4096)
    for {
        n, readErr := resp.Body.Read(buf)
        if n > 0 {
            if _, err := w.Write(buf[:n]); err != nil {
                return
            }
            if canFlush {
                flusher.Flush()
            }
        }
        if readErr != nil {
            return
        }
    }
}
