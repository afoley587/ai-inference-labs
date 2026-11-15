package gateway

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
	if err != nil {
		t.Fatal(err)
	}
	frontend := httptest.NewServer(h)
	defer frontend.Close()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(frontend.URL+"/v1/chat/completions", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if line != "data: first\n" {
		t.Fatalf("first line = %q", line)
	}
}

func TestUpstreamFailureReturnsBadGateway(t *testing.T) {
	h, err := New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRejectsInvalidOrigin(t *testing.T) {
	for _, input := range []string{"file:///etc/passwd", "http://a/path", "http://user:pw@host", "http://host?x=1", "http://host?", "http://:8081"} {
		if _, err := New(input); err == nil {
			t.Errorf("accepted invalid backend %q", input)
		}
	}
}

func TestMethodAndHealth(t *testing.T) {
	h, err := New("http://localhost:8081")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{"GET", "/healthz", 200},
		{"GET", "/v1/chat/completions", 405},
		{"GET", "/missing", 404},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != tc.want {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}
}

// A real HTTP server is important here: ResponseRecorder cannot expose buffering
// or the difference between an aborted response and a clean end of body.
func newTestGateway(t *testing.T, backendURL string) *httptest.Server {
	t.Helper()
	h, err := New(backendURL)
	if err != nil {
		t.Fatal(err)
	}
	frontend := httptest.NewServer(h)
	t.Cleanup(frontend.Close)
	t.Cleanup(h.client.CloseIdleConnections)
	return frontend
}

func TestDoesNotFollowBackendRedirect(t *testing.T) {
	var redirected atomic.Bool
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected.Store(true)
	}))
	defer destination.Close()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer backend.Close()
	frontend := newTestGateway(t, backend.URL)
	client := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Post(frontend.URL+"/v1/chat/completions", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTemporaryRedirect || resp.Header.Get("Location") != destination.URL {
		t.Fatalf("redirect not preserved: status %d, location %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if redirected.Load() {
		t.Fatal("gateway followed backend redirect")
	}
}

func TestForwardsRequestAndUpstreamError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if r.Method != http.MethodPost || r.URL.RequestURI() != "/v1/chat/completions?model=test" || string(body) != `{"stream":true}` {
			t.Errorf("unexpected upstream request: %s %s %q", r.Method, r.URL, body)
		}
		if r.Header.Get("Authorization") != "Bearer test" || r.Header.Get("Accept") != "text/event-stream" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("forwarded headers: %v", r.Header)
		}
		w.Header().Set("Retry-After", "5")
		w.Header().Set("X-Request-Id", "test-request")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "busy")
	}))
	defer backend.Close()
	frontend := newTestGateway(t, backend.URL)
	req, err := http.NewRequest(http.MethodPost, frontend.URL+"/v1/chat/completions?model=test", strings.NewReader(`{"stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer test")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusTooManyRequests || string(body) != "busy" || resp.Header.Get("Retry-After") != "5" || resp.Header.Get("X-Request-Id") != "test-request" {
		t.Fatalf("upstream error not preserved: status %d, headers %v, body %q", resp.StatusCode, resp.Header, body)
	}
}

func TestClientCancellationReachesBackend(t *testing.T) {
	canceled := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(canceled)
	}))
	defer backend.Close()
	frontend := newTestGateway(t, backend.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, frontend.URL+"/v1/chat/completions", nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := bufio.NewReader(resp.Body).ReadString('\n'); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("backend did not observe cancellation")
	}
}

func TestAbortsTruncatedUpstreamStream(t *testing.T) {
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: partial\n\n")
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	}))
	backend.Start()
	defer backend.Close()
	frontend := newTestGateway(t, backend.URL)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(frontend.URL+"/v1/chat/completions", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Fatal("truncated upstream stream appeared to end cleanly")
	}
}

func TestPreservesSplitEventsAndCompletion(t *testing.T) {
	const want = "data: {\"token\":\"Hello\"}\n\ndata: [DONE]\n\n"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, part := range []string{"data: {\"tok", "en\":\"Hello\"}\n", "\ndata: [DONE]\n\n"} {
			fmt.Fprint(w, part)
			w.(http.Flusher).Flush()
		}
	}))
	defer backend.Close()
	frontend := newTestGateway(t, backend.URL)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(frontend.URL+"/v1/chat/completions", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
	if resp.Header.Get("Content-Type") != "text/event-stream" || resp.Header.Get("X-Accel-Buffering") != "no" {
		t.Fatalf("streaming headers = %v", resp.Header)
	}
}
