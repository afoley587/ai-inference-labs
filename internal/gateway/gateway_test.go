package gateway

import (
    "bufio"
    "fmt"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"
)

func TestStreamsBeforeUpstreamCompletion(t *testing.T) {
    release := make(chan struct{})
    defer close(release)
    backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/event-stream")
        fmt.Fprint(w, "data: first\n\n")
        w.(http.Flusher).Flush()
        select {
        case <-release:
        case <-r.Context().Done():
        }
    }))
    defer backend.Close()
    h, err := New(backend.URL)
    if err != nil { t.Fatal(err) }
    frontend := httptest.NewServer(h)
    defer frontend.Close()
    client := &http.Client{Timeout: 2*time.Second}
    resp, err := client.Post(frontend.URL+"/v1/chat/completions", "application/json", strings.NewReader("{}"))
    if err != nil { t.Fatal(err) }
    defer resp.Body.Close()
    line, err := bufio.NewReader(resp.Body).ReadString('\n')
    if err != nil { t.Fatal(err) }
    if line != "data: first\n" { t.Fatalf("first line = %q", line) }
}

func TestUpstreamFailureReturnsBadGateway(t *testing.T) {
    h, err := New("http://127.0.0.1:1")
    if err != nil { t.Fatal(err) }
    req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{}"))
    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, req)
    if rec.Code != http.StatusBadGateway { t.Fatalf("status = %d", rec.Code) }
}

func TestRejectsInvalidOrigin(t *testing.T) {
    for _, input := range []string{"file:///etc/passwd", "http://a/path", "http://user:pw@host", "http://host?x=1"} {
        if _, err := New(input); err == nil { t.Errorf("accepted invalid backend %q", input) }
    }
}

func TestMethodAndHealth(t *testing.T) {
    h, err := New("http://localhost:8081")
    if err != nil { t.Fatal(err) }
    for _, tc := range []struct { method, path string; want int }{
        {"GET", "/healthz", 200},
        {"GET", "/v1/chat/completions", 405},
        {"GET", "/missing", 404},
    } {
        rec := httptest.NewRecorder()
        h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
        if rec.Code != tc.want { t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, rec.Code, tc.want) }
    }
}
