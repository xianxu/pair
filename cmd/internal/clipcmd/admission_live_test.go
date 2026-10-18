package clipcmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifytransport"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

type nativeClipboardQuery struct {
	ctx       context.Context
	elapsed   time.Duration
	remaining time.Duration
}

// Clipboard data is a private literal, never the operator clipboard. Native
// query, focus, paste trigger and source color use the production OSRuntime.
// Only the cosmetic reset timer is recorded rather than spawning a child that
// could outlive this owned fixture; its delayed scheduling is not this oracle.
type nativeClipboardRuntime struct {
	OSRuntime
	queries []nativeClipboardQuery
	resets  []string
}

func (*nativeClipboardRuntime) ClipboardPaste() (string, bool) {
	return "private native clipboard fixture", true
}
func (r *nativeClipboardRuntime) ResetPaneColorAfter(id string, _ time.Duration) error {
	r.resets = append(r.resets, id)
	return nil
}
func (r *nativeClipboardRuntime) ListClientsContext(ctx context.Context) ([]byte, error) {
	start := time.Now()
	deadline, _ := ctx.Deadline()
	remaining := time.Until(deadline)
	raw, err := r.OSRuntime.ListClientsContext(ctx)
	r.queries = append(r.queries, nativeClipboardQuery{ctx, time.Since(start), remaining})
	return raw, err
}
func (r *nativeClipboardRuntime) ListPanesContext(ctx context.Context) ([]byte, error) {
	start := time.Now()
	deadline, _ := ctx.Deadline()
	remaining := time.Until(deadline)
	raw, err := r.OSRuntime.ListPanesContext(ctx)
	r.queries = append(r.queries, nativeClipboardQuery{ctx, time.Since(start), remaining})
	return raw, err
}

// Run only in the coordinated native window:
// PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/clipcmd -run '^TestClipboardAdmissionInstalledZellijLive$' -count=1 -v
// This qualifies observed healthy latency and exact native input routing. It
// does not assert that a copy hook has historical source provenance, nor that
// OS clipboard copying occurred; the separate detached CLI test owns that proof.
func TestClipboardAdmissionInstalledZellijLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_ZELLIJ") != "1" {
		t.Skip("set PAIR_LIVE_ZELLIJ=1 for disposable installed-native clipboard conformance")
	}
	native, err := exec.LookPath("zellij")
	if err != nil {
		t.Fatal(err)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal(err)
	}
	//lint:allow tmp-literal — the native session and notification digest/PID sockets share this root and require short AF_UNIX addresses
	root, err := os.MkdirTemp("/tmp", "pcl-live-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	for _, name := range []string{"home", "cache", "runtime", "config", "data", "notify", "socket", "nvim"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	// The shared empty-environment owner prevents ambient Couch/Pair bindings,
	// credentials, shell startup files and operator native sockets from crossing.
	testpair.ApplyEnvironment(t, filepath.Join(root, "home"),
		"PATH="+filepath.Dir(native)+string(os.PathListSeparator)+filepath.Dir(python)+":/usr/bin:/bin",
		"TMPDIR=/tmp", "SHELL=/bin/sh", "ENV=", "PS1=",
		"XDG_CONFIG_HOME="+filepath.Join(root, "config"), "XDG_DATA_HOME="+filepath.Join(root, "data"),
		"XDG_CACHE_HOME="+filepath.Join(root, "cache"), "XDG_RUNTIME_DIR="+filepath.Join(root, "runtime"),
		"ZELLIJ_SOCKET_DIR="+filepath.Join(root, "socket"), "PAIR_DATA_DIR="+filepath.Join(root, "data"), "PAIR_TAG=native",
		"PAIR_NOTIFY_SOCKET_DIR="+filepath.Join(root, "notify"))
	binding := filepath.Join(root, "wrapper-pid")
	broker, err := notifytransport.Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := broker.Close(); err != nil {
			t.Error(err)
		}
	})
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", binding)

	stream := newClipboardNativeStream(t, filepath.Join(root, "receiver.sock"))
	receiver := []byte("import os,socket,sys,tty\ntty.setraw(0)\nout=socket.socket(socket.AF_UNIX,socket.SOCK_STREAM)\nout.connect(sys.argv[1])\nout.sendall(sys.argv[2].encode())\nwhile True:\n data=os.read(0,4096)\n if not data: break\n out.sendall(data)\n")
	sourceScript := filepath.Join(root, "receiver.py")
	draftScript := filepath.Join(root, "nvim", "init.lua")
	for _, path := range []string{sourceScript, draftScript} {
		if err := os.WriteFile(path, receiver, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// These are tiny raw-byte receiver processes. Their original layout argv
	// explicitly supplies the Pair role labels; no editor, agent or Pair build is
	// involved. Native pane replies, not a rewritten query, carry these templates.
	pane := func(name, script, marker, role string) string {
		return fmt.Sprintf("pane name=%s command=%s { args %s %s %s %s; }\n", strconv.Quote(name), strconv.Quote(python), strconv.Quote(script), strconv.Quote(stream.listener.Addr().String()), strconv.Quote(marker), strconv.Quote(role))
	}
	layout := filepath.Join(root, "layout.kdl")
	body := "layout {\n pane split_direction=\"vertical\" {\n" + pane("copy-source", sourceScript, "a", "pair wrap") + pane("copy-draft", draftScript, "d", "nvim") + " }\n}\n"
	if err := os.WriteFile(layout, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "config.kdl")
	if err := os.WriteFile(config, []byte("pane_frames false\nshow_startup_tips false\nshow_release_notes false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var entropy [8]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal(err)
	}
	// Match the established live fixture birth bound; admission itself remains
	// the unchanged production five seconds, never a larger native-query budget.
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	session := "pair-clip-" + hex.EncodeToString(entropy[:])
	fixture, err := pairlifecycletest.StartControlledZellijWithOptions(ctx, session, pairlifecycletest.ZellijOptions{ConfigFile: config, LayoutFile: layout})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Errorf("owned native fixture close: %v", err)
		}
	})
	t.Setenv("ZELLIJ_SESSION_NAME", session)
	clipboardNativeReady(t, ctx, stream)
	raw, err := (OSRuntime{}).ListPanesContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	panes, err := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{Command: true, Tab: true})
	if err != nil {
		t.Fatal(err)
	}
	var source, draft zellijpane.Pane
	for _, p := range panes {
		if !p.IsPlugin && p.Title == "copy-source" {
			source = p
		}
		if !p.IsPlugin && p.Title == "copy-draft" {
			draft = p
		}
	}
	if source.ID == "" || draft.ID == "" {
		t.Fatalf("receiver pane identities missing: %+v", panes)
	}
	t.Setenv("ZELLIJ_PANE_ID", source.ID)
	if err := (OSRuntime{}).FocusPaneContext(ctx, source.ID); err != nil {
		t.Fatal(err)
	}
	// Actual client input establishes its last-active identity and independently
	// proves readiness/focus before the source-context observation. No CLI retry.
	if _, err := fixture.WriteInput([]byte("A")); err != nil {
		t.Fatal(err)
	}
	if got := clipboardNativeByte(t, ctx, stream, 'a'); got != 'A' {
		t.Fatalf("source client focus marker=%q", got)
	}

	opts := ClipboardToPaneOptions{DataDir: filepath.Join(root, "data"), Tag: "native"}
	for _, operation := range []string{"auto-quote", "standalone-paste"} {
		rt := &nativeClipboardRuntime{}
		var stderr bytes.Buffer
		start := time.Now()
		var code int
		if operation == "auto-quote" {
			code = RunCopyOnSelectOrchestrate(CopyOnSelectOptions{Clipboard: opts}, rt, &stderr)
		} else {
			code = RunClipboardToPane(opts, rt, &stderr)
		}
		elapsed := time.Since(start)
		if code != 0 {
			t.Fatalf("%s refused after %s: %s", operation, elapsed, &stderr)
		}
		wantQueries := 2
		if operation == "auto-quote" {
			wantQueries = 4
		}
		if len(rt.queries) != wantQueries {
			t.Fatalf("%s query count=%d want=%d", operation, len(rt.queries), wantQueries)
		}
		var observationTime time.Duration
		for _, query := range rt.queries {
			if query.ctx != rt.queries[0].ctx {
				t.Fatalf("%s replaced admission context", operation)
			}
			if query.remaining <= 0 || query.remaining > 5*time.Second {
				t.Fatalf("%s query remaining budget=%s", operation, query.remaining)
			}
			observationTime += query.elapsed
		}
		t.Logf("%s source/destination native confirmation=%s full production helper=%s queries=%d", operation, observationTime, elapsed, len(rt.queries))
		if got := clipboardNativeByte(t, ctx, stream, 'd'); got != 31 {
			t.Fatalf("%s draft input=%q want Ctrl31", operation, got)
		}
		select {
		case event, ok := <-stream.events:
			if !ok {
				t.Fatal("receiver stream closed unexpectedly")
			}
			t.Fatalf("%s received extra receiver event: %+v", operation, event)
		default:
		}
		quote, err := os.ReadFile(filepath.Join(opts.DataDir, "quote-native"))
		if err != nil || string(quote) != "private native clipboard fixture" {
			t.Fatalf("%s staging=%q,%v", operation, quote, err)
		}
		if operation == "auto-quote" && (len(rt.resets) != 1 || rt.resets[0] != source.ID) {
			t.Fatalf("source flash reset request=%v", rt.resets)
		}
		select {
		case message := <-broker.Messages():
			t.Fatalf("healthy %s emitted refusal %q", operation, message)
		default:
		}
	}
	// Independently prove final presenting-client focus, after both actual native
	// focus/trigger operations, using its PTY input rather than another query.
	if _, err := fixture.WriteInput([]byte("D")); err != nil {
		t.Fatal(err)
	}
	if got := clipboardNativeByte(t, ctx, stream, 'd'); got != 'D' {
		t.Fatalf("final client focus reached %q", got)
	}
}

// One private listener accepts exactly the two owned receivers. Native query
// transport is unchanged; this stream only witnesses fixture readiness/input.
type clipboardNativeEvent struct {
	role  byte
	value byte
	ready bool
	err   error
}
type clipboardNativeStream struct {
	listener   *net.UnixListener
	events     chan clipboardNativeEvent
	stop       chan struct{}
	acceptDone chan struct{}
	conns      []net.Conn // written only by acceptor; read after acceptDone is joined
	readers    sync.WaitGroup
	once       sync.Once
}

func newClipboardNativeStream(t *testing.T, path string) *clipboardNativeStream {
	t.Helper()
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	stream := &clipboardNativeStream{listener: listener, events: make(chan clipboardNativeEvent, 64), stop: make(chan struct{}), acceptDone: make(chan struct{})}
	go func() {
		defer close(stream.acceptDone)
		for i := 0; i < 2; i++ {
			conn, err := listener.AcceptUnix()
			if err != nil {
				stream.publish(clipboardNativeEvent{err: fmt.Errorf("accept receiver: %w", err)})
				return
			}
			stream.conns = append(stream.conns, conn)
			stream.readers.Add(1)
			go stream.read(conn)
		}
	}()
	t.Cleanup(stream.close)
	return stream
}
func (s *clipboardNativeStream) publish(event clipboardNativeEvent) bool {
	select {
	case s.events <- event:
		return true
	case <-s.stop:
		return false
	}
}
func (s *clipboardNativeStream) read(conn net.Conn) {
	defer s.readers.Done()
	var b [1]byte
	if _, err := io.ReadFull(conn, b[:]); err != nil {
		s.publish(clipboardNativeEvent{err: fmt.Errorf("receiver readiness: %w", err)})
		return
	}
	role := b[0]
	if role != 'a' && role != 'd' {
		s.publish(clipboardNativeEvent{role: role, err: fmt.Errorf("invalid receiver role %q", role)})
		return
	}
	if !s.publish(clipboardNativeEvent{role: role, ready: true}) {
		return
	}
	for {
		if _, err := io.ReadFull(conn, b[:]); err != nil {
			s.publish(clipboardNativeEvent{role: role, err: fmt.Errorf("receiver input: %w", err)})
			return
		}
		if !s.publish(clipboardNativeEvent{role: role, value: b[0]}) {
			return
		}
	}
}
func (s *clipboardNativeStream) close() {
	s.once.Do(func() {
		// Listener close releases Accept; stop releases event publication. Join the
		// acceptor before accessing its connection list or waiting for its readers.
		s.listener.Close()
		close(s.stop)
		<-s.acceptDone
		for _, conn := range s.conns {
			conn.Close()
		}
		s.readers.Wait()
		close(s.events)
	})
}
func clipboardNativeEventFrom(t *testing.T, ctx context.Context, s *clipboardNativeStream) clipboardNativeEvent {
	t.Helper()
	select {
	case event, ok := <-s.events:
		if !ok {
			t.Fatal("owned receiver stream closed")
		}
		if event.err != nil {
			t.Fatal(event.err)
		}
		return event
	case <-ctx.Done():
		t.Fatalf("receiver event: %v", ctx.Err())
		return clipboardNativeEvent{}
	}
}
func clipboardNativeReady(t *testing.T, ctx context.Context, s *clipboardNativeStream) {
	t.Helper()
	seen := map[byte]bool{}
	for i := 0; i < 2; i++ {
		event := clipboardNativeEventFrom(t, ctx, s)
		if !event.ready || seen[event.role] {
			t.Fatalf("unexpected receiver readiness: %+v", event)
		}
		seen[event.role] = true
	}
}
func clipboardNativeByte(t *testing.T, ctx context.Context, s *clipboardNativeStream, role byte) byte {
	t.Helper()
	event := clipboardNativeEventFrom(t, ctx, s)
	if event.ready || event.role != role {
		t.Fatalf("expected input from receiver %q, got %+v", role, event)
	}
	return event.value
}

func TestClipboardNativeStreamDeliveryAndCleanup(t *testing.T) {
	for _, scenario := range []string{"delivery", "no-client", "idle-reader", "unread-buffer", "read-error"} {
		t.Run(scenario, func(t *testing.T) {
			root, err := os.MkdirTemp("", "pci-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.RemoveAll(root) })
			stream := newClipboardNativeStream(t, filepath.Join(root, "receiver.sock"))
			if scenario == "no-client" {
				stream.close()
				return
			}
			if scenario == "unread-buffer" {
				// Fill publication capacity before readers start; cleanup must release a
				// publication whether it carries readiness, data or a closed-read error.
				for i := 0; i < cap(stream.events); i++ {
					stream.events <- clipboardNativeEvent{value: 'x'}
				}
			}
			connect := func() net.Conn {
				conn, err := net.Dial("unix", stream.listener.Addr().String())
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { conn.Close() })
				return conn
			}
			source := connect()
			if scenario == "read-error" {
				source.Close()
				select {
				case event := <-stream.events:
					if !errors.Is(event.err, io.EOF) {
						t.Fatalf("read failure lost: %+v", event)
					}
				case <-t.Context().Done():
					t.Fatal(t.Context().Err())
				}
				stream.close()
				return
			}
			draft := connect()
			<-stream.acceptDone // both accepted connections are now tracked and owned
			if scenario == "idle-reader" {
				stream.close()
				return
			}
			if _, err := source.Write([]byte{'a'}); err != nil {
				t.Fatal(err)
			}
			if _, err := draft.Write([]byte{'d'}); err != nil {
				t.Fatal(err)
			}
			if scenario == "unread-buffer" {
				stream.close()
				return
			}
			clipboardNativeReady(t, t.Context(), stream)
			if _, err := source.Write([]byte{'A'}); err != nil {
				t.Fatal(err)
			}
			if got := clipboardNativeByte(t, t.Context(), stream, 'a'); got != 'A' {
				t.Fatalf("source data=%q", got)
			}
			if _, err := draft.Write([]byte{31}); err != nil {
				t.Fatal(err)
			}
			if got := clipboardNativeByte(t, t.Context(), stream, 'd'); got != 31 {
				t.Fatalf("draft data=%q", got)
			}
			stream.close()
		})
	}
}
