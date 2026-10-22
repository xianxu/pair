package mockserve

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFrontClosesUpgradedConnectionAndJoins(t *testing.T) {
	upstreamDone := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			close(upstreamDone)
			return
		}
		defer close(upstreamDone)
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
		_ = rw.Flush()
		_, _ = io.Copy(conn, conn)
	}))
	defer upstream.Close()
	p, _ := frontProject(t)
	s, err := newHybridServer(t.Context(), p, "upgrade")
	if err != nil {
		t.Fatal(err)
	}
	f, err := newDevelopmentFront(t.Context(), s, upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	server := httptest.NewServer(f)
	defer server.Close()
	conn, err := net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	_, _ = fmt.Fprintf(conn, "GET /framework/hmr HTTP/1.1\r\nHost: front\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
	reader := bufio.NewReader(conn)
	res, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 101 {
		t.Fatal(res.Status)
	}
	_, _ = conn.Write([]byte("x"))
	b, err := reader.ReadByte()
	if err != nil || b != 'x' {
		t.Fatalf("upgraded exchange: %c %v", b, err)
	}
	done := make(chan struct{})
	go func() { _ = f.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("front did not join upgraded request")
	}
	select {
	case <-upstreamDone:
	case <-time.After(10 * time.Second):
		t.Fatal("upstream upgrade survived front")
	}
	if _, err := reader.ReadByte(); err == nil {
		t.Fatal("downstream survived close")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.active != 0 || len(f.connections) != 0 {
		t.Fatalf("retained active=%d connections=%d", f.active, len(f.connections))
	}
}

func TestFrontCancelsStreamAndCapacity(t *testing.T) {
	started := make(chan struct{}, maxFrontRequests)
	ended := make(chan struct{}, maxFrontRequests)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: ready\n\n")
		w.(http.Flusher).Flush()
		started <- struct{}{}
		<-r.Context().Done()
		ended <- struct{}{}
	}))
	defer upstream.Close()
	p, _ := frontProject(t)
	s, err := newHybridServer(t.Context(), p, "stream")
	if err != nil {
		t.Fatal(err)
	}
	f, err := newDevelopmentFront(t.Context(), s, upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	server := httptest.NewServer(f)
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	for range maxFrontRequests {
		req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/events", nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { res.Body.Close() })
		select {
		case <-started:
		case <-time.After(10 * time.Second):
			t.Fatal("stream not ready")
		}
	}
	res, err := http.Get(server.URL + "/overflow")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 503 {
		t.Fatalf("overload: %d", res.StatusCode)
	}
	done := make(chan struct{})
	go func() { _ = f.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("stream requests did not join")
	}
	for range maxFrontRequests {
		select {
		case <-ended:
		case <-time.After(10 * time.Second):
			t.Fatal("upstream stream survived")
		}
	}
}

func TestFrontCancellationJoinsIncompleteUpload(t *testing.T) {
	ready := make(chan struct{})
	ended := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(ended)
		close(ready)
		_, _ = io.Copy(io.Discard, r.Body)
	}))
	defer upstream.Close()
	p, _ := frontProject(t)
	s, err := newHybridServer(t.Context(), p, "upload")
	if err != nil {
		t.Fatal(err)
	}
	f, err := newDevelopmentFront(t.Context(), s, upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	server := httptest.NewServer(f)
	defer server.Close()
	conn, err := net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, _ = io.WriteString(conn, "POST /api/upload HTTP/1.1\r\nHost: front\r\nTransfer-Encoding: chunked\r\n\r\n1\r\nx\r\n")
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		t.Fatal("upstream did not see upload")
	}
	done := make(chan struct{})
	go func() { _ = f.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("incomplete client upload blocked front cancellation")
	}
	select {
	case <-ended:
	case <-time.After(3 * time.Second):
		t.Fatal("upstream upload not joined")
	}
}
