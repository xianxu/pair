package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

// Serve owns its listener and joins graceful shutdown before returning.
func Serve(ctx context.Context, address string, screens []Screen) error {
	h, e := New(screens)
	if e != nil {
		return e
	}
	listener, e := net.Listen("tcp", address)
	if e != nil {
		return e
	}
	server := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second}
	fmt.Fprintf(os.Stdout, "listening http://%s\n", listener.Addr())
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if server.Shutdown(shutdown) != nil {
				_ = server.Close()
			}
		case <-finished:
		}
	}()
	e = server.Serve(listener)
	if e == http.ErrServerClosed {
		e = nil
	} // cancellation signals normal return
	// Closing the server also wakes the owner if Serve failed independently.
	_ = server.Close()
	close(finished)
	<-done
	return e
}
