package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("X-Accel-Buffering", "no")
        f, ok := w.(http.Flusher)
        if !ok { http.Error(w, "streaming unsupported", http.StatusInternalServerError); return }
        for _, token := range []string{"Hello", " from", " the", " mock", " backend"} {
            select {
            case <-r.Context().Done(): return
            case <-time.After(250 * time.Millisecond):
            }
            data, _ := json.Marshal(map[string]string{"token":token})
            if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil { return }
            f.Flush()
        }
        _, _ = fmt.Fprint(w, "data: [DONE]\n\n")
        f.Flush()
    })
    addr := os.Getenv("MOCK_ADDR")
    if addr == "" { addr = ":8081" }
    log.Printf("mock backend listening on %s", addr)
    log.Fatal((&http.Server{Addr:addr, Handler:mux, ReadHeaderTimeout:5*time.Second}).ListenAndServe())
}
