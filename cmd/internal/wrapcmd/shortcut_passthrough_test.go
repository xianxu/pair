package wrapcmd

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

type shortcutChunkReader struct{ chunks [][]byte }

func (r *shortcutChunkReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.chunks[0])
	r.chunks[0] = r.chunks[0][n:]
	if len(r.chunks[0]) == 0 {
		r.chunks = r.chunks[1:]
	}
	return n, nil
}

// Transport actual output from the outer input owner into the production
// wrapper. The fake is the attached terminal, not a second chord encoder.
func TestConsoleWrapperShortcutPassthrough(t *testing.T) {
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	reader, writer := io.Pipe()
	console := couchtty.New(host, reader)
	child := ptychild.NewFakeChild([]byte("\x1b[>1u\x1b[?2004h"))
	console.Attach("agent", "agent", child)
	done := make(chan int, 1)
	go func() { done <- console.Run() }()
	t.Cleanup(func() {
		console.Stop()
		_ = writer.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("Console did not stop")
		}
	})
	// Pasted navigation must reach the agent literally. An unpasted tab key
	// reaches the wrapper but performs its reserved action there. Alt+n is
	// Couch's own from every pane (#284), so it appears only inside the paste,
	// where it is content for both layers.
	literal := "\x1b[100;3u\x1bx\x1b[1;3A\x1bh\x1bl\x1b[200~\x00\x1b[84;4u\x1b[110;3u\r\x1b[201~"
	input := []byte(literal + "\x1b[84;4u" + "END245")
	if _, err := writer.Write(input); err != nil {
		t.Fatal(err)
	}
	var delivered []byte
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		delivered = bytes.Join(child.Writes(), nil)
		if bytes.HasSuffix(delivered, []byte("END245")) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	// Endpoint re-encodes semantic input for the child's negotiated protocol.
	encodedLiteral := strings.NewReplacer("\x1bx", "\x1b[120;3u", "\x1bh", "\x1b[104;3u", "\x1bl", "\x1b[108;3u").Replace(literal)
	wantDelivered := []byte(encodedLiteral + "\x1b[84;4uEND245")
	if !bytes.Equal(delivered, wantDelivered) {
		t.Fatalf("Couch delivered %q; want %q", delivered, wantDelivered)
	}
	var actions []string
	p := &proxy{workbenchShortcutHandler: func(chord string) bool { actions = append(actions, chord); return true }}
	var out bytes.Buffer
	p.translateStdinFrom(bytes.NewReader(delivered), &out, time.Second)
	if out.String() != encodedLiteral+"END245" || strings.Join(actions, ",") != "Alt+Shift+T" {
		t.Fatalf("agent=%q actions=%v", out.String(), actions)
	}
}

func TestWrapperShortcutStreamPartitions(t *testing.T) {
	reserved := "\x1b[84;4u"
	// Include a real reserved chord after the paste closes. Literal pasted
	// keys and CRs must survive in both adaptation configurations.
	literal := "prefix\x1b[200~" + reserved + "\r\x1b[1;3A\x1b[201~\x1b[1;3B\x1bx"
	stream := []byte(literal + reserved + "tail")
	for _, agent := range []string{"", "claude", "codex", "muse"} {
		for split := 0; split <= len(stream)+1; split++ {
			p := &proxy{lifecycleEvents: make(chan TurnObservation, 20)}
			if agent != "" {
				// Claude's profile supports Return adaptation without an output
				// composer emulator; every pasted CR remains literal.
				profile, ok := profileForHarness(agent, true)
				if !ok {
					t.Fatalf("missing %s profile", agent)
				}
				p.ttyProfile = &profile
			}
			var actions []string
			p.workbenchShortcutHandler = func(chord string) bool { actions = append(actions, chord); return true }
			var out bytes.Buffer
			var chunks [][]byte
			if split > len(stream) {
				for i := range stream {
					chunks = append(chunks, stream[i:i+1])
				}
			} else {
				chunks = [][]byte{stream[:split], stream[split:]}
			}
			reader := &shortcutChunkReader{chunks: chunks}
			p.translateStdinFrom(reader, &out, time.Second)
			if len(p.lifecycleEvents) != 0 {
				t.Fatalf("pasted CR produced submission: %v", <-p.lifecycleEvents)
			}
			if out.String() != literal+"tail" || strings.Join(actions, ",") != "Alt+Shift+T" {
				t.Fatalf("agent=%q split=%d output=%q actions=%v", agent, split, out.String(), actions)
			}
		}
	}
}

func TestAgentAllUnreservedChordEncodingsReachInput(t *testing.T) {
	for chord := workbenchshortcut.Chord(1); chord < workbenchshortcut.ChordMax(); chord++ {
		if chord == workbenchshortcut.ChordAltShiftT || chord == workbenchshortcut.ChordAltShiftLeft || chord == workbenchshortcut.ChordAltShiftRight {
			continue
		}
		for _, raw := range workbenchshortcut.ChordEncodings(chord) {
			p := &proxy{workbenchShortcutHandler: func(string) bool { t.Errorf("unreserved %q intercepted", raw); return true }}
			var out bytes.Buffer
			p.translateStdinFrom(bytes.NewReader(raw), &out, time.Second)
			if !bytes.Equal(out.Bytes(), raw) {
				t.Errorf("chord %v input=%q output=%q", chord, raw, out.Bytes())
			}
		}
	}
}

func TestReservedShortcutAfterIncompletePrefixEOF(t *testing.T) {
	for _, remap := range []bool{false, true} {
		for _, binding := range workbenchshortcut.GlobalBindings() {
			if !binding.AgentReserved {
				continue
			}
			for _, chord := range workbenchshortcut.ChordEncodings(binding.Chord) {
				for _, prefix := range []string{"\x1b", "\x1b[", "\x1b[1;", "\x1b[20"} {
					stream := append(append([]byte("text"+prefix), chord...), []byte("tail\x1b[")...)
					for split := 0; split <= len(stream); split++ {
						p := &proxy{}
						if remap {
							profile, _ := profileForHarness("claude", true)
							p.ttyProfile = &profile
						}
						actions := 0
						p.workbenchShortcutHandler = func(string) bool { actions++; return true }
						var out bytes.Buffer
						p.translateStdinFrom(&shortcutChunkReader{chunks: [][]byte{stream[:split], stream[split:]}}, &out, time.Second)
						if actions != 1 || out.String() != "text"+prefix+"tail\x1b[" {
							t.Fatalf("remap=%v chord=%q prefix=%q split=%d actions=%d output=%q", remap, chord, prefix, split, actions, out.String())
						}
					}
				}
			}
		}
	}
}

func TestReservedShortcutPrefixProgressBeforeTimeout(t *testing.T) {
	for _, remap := range []bool{false, true} {
		for _, binding := range workbenchshortcut.GlobalBindings() {
			if !binding.AgentReserved {
				continue
			}
			for _, prefix := range []string{"\x1b", "\x1b[1;"} {
				p := &proxy{}
				if remap {
					profile, _ := profileForHarness("claude", true)
					p.ttyProfile = &profile
				}
				actions := make(chan struct{}, 16)
				p.workbenchShortcutHandler = func(string) bool { actions <- struct{}{}; return true }
				reader, writer := io.Pipe()
				out := &drainBuffer{}
				done := make(chan struct{})
				go func() { p.translateStdinFrom(reader, out, 40*time.Millisecond); close(done) }()
				t.Cleanup(func() {
					_ = writer.Close()
					_ = reader.Close()
					select {
					case <-done:
					case <-time.After(time.Second):
						t.Error("prefix fixture did not stop")
					}
				})
				// Every read must execute its completed action. Waiting for the action
				// also proves previous input cannot accumulate in pending across reads.
				chord := workbenchshortcut.ChordEncodings(binding.Chord)[0]
				var want []byte
				for n := 0; n < 8; n++ {
					chunk := append([]byte("text"+prefix), chord...)
					_, err := writer.Write(chunk)
					if err != nil {
						t.Fatal(err)
					}
					select {
					case <-actions:
					case <-time.After(time.Second):
						_ = writer.Close()
						<-done
						t.Fatalf("complete chord retained: remap=%v prefix=%q chord=%q iteration=%d", remap, prefix, chord, n)
					}
					want = append(want, []byte("text"+prefix)...)
					if !bytes.Equal(out.Bytes(), want) {
						t.Fatalf("prefix was not delivered before action: got=%q want=%q", out.Bytes(), want)
					}
				}
				// A genuinely incomplete trailing CSI still waits for the timeout, then
				// flushes once. EOF must not duplicate it.
				_, _ = writer.Write([]byte("\x1b["))
				want = append(want, []byte("\x1b[")...)
				deadline := time.Now().Add(time.Second)
				for !bytes.Equal(out.Bytes(), want) && time.Now().Before(deadline) {
					time.Sleep(time.Millisecond)
				}
				_ = writer.Close()
				<-done
				if !bytes.Equal(out.Bytes(), want) {
					t.Fatalf("timeout/EOF bytes=%q want=%q", out.Bytes(), want)
				}
			}
		}
	}
}
