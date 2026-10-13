package couchmessage

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestResponseAfterWriteWaitsForFrameAndReportsFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "written", true: "closed"}[fail], func(t *testing.T) {
			socket := transportSocket(t)
			entered := make(chan struct{})
			release := make(chan struct{})
			done := make(chan error, 2)
			server, err := StartResponseServer(context.Background(), socket, func(context.Context, []byte) Response {
				close(entered)
				<-release
				return Response{Code: "accepted", AfterWrite: func(err error) { done <- err }}
			})
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			conn, err := net.Dial("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if err := transportWriteFrame(conn, []byte(`{}`)); err != nil {
				t.Fatal(err)
			}
			<-entered
			select {
			case <-done:
				t.Fatal("hook before handler finished")
			default:
			}
			if fail {
				conn.Close()
			}
			close(release)
			if !fail {
				if _, err := transportReadFrame(conn); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case err := <-done:
				if (err != nil) != fail {
					t.Fatalf("failure %v: %v", fail, err)
				}
			case <-time.After(time.Second):
				t.Fatal("missing callback")
			}
			server.Close()
			select {
			case <-done:
				t.Fatal("duplicate callback")
			default:
			}
		})
	}
}

// net.Pipe holds the response write until the client consumes it, proving that
// the completion cannot run merely because a handler produced the response.
type observedResponseConn struct {
	net.Conn
	writing chan struct{}
}

func (c observedResponseConn) Write(body []byte) (int, error) {
	close(c.writing)
	return c.Conn.Write(body)
}
func TestResponseAfterWriteCannotPrecedeCompletedWrite(t *testing.T) {
	serverConn, client := net.Pipe()
	defer client.Close()
	writing := make(chan struct{})
	done := make(chan error, 1)
	conn := observedResponseConn{Conn: serverConn, writing: writing}
	s := &Server{ctx: context.Background(), connections: map[net.Conn]struct{}{conn: {}}, handler: func(context.Context, []byte) ([]byte, func(error), error) {
		return []byte(`{"Code":"accepted"}`), func(err error) { done <- err }, nil
	}}
	s.handlers.Add(1)
	go s.serve(conn)
	if err := transportWriteFrame(client, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	<-writing
	select {
	case <-done:
		t.Fatal("completion preceded the write")
	default:
	}
	if _, err := transportReadFrame(client); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("missing completion")
	}
	s.handlers.Wait()
}
