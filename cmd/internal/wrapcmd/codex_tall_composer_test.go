package wrapcmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

const codexTallTestFooter = "GPT-6.1-Sol medium · /private/tmp/project"

func codexTallPaint(text, footer string) string {
	lines := strings.Split(text, "\n")
	var out strings.Builder
	out.WriteString("\x1b[2J\x1b[?2004h\x1b[3;1H\x1b[1m›\x1b[22m ")
	for i, line := range lines {
		fmt.Fprintf(&out, "\x1b[%d;3H%s", 3+i, line)
	}
	if text == "" {
		out.WriteString("\x1b[3;3H\x1b[2mAsk Codex to do anything\x1b[22m")
	}
	if footer != "" {
		fmt.Fprintf(&out, "\x1b[%d;3H%s", 4+len(lines), footer)
	}
	fmt.Fprintf(&out, "\x1b[?25h\x1b[%d;%dH", 2+len(lines), 3+xansi.StringWidth(lines[len(lines)-1]))
	return out.String()
}

func codexTallText(lines int) string {
	rows := make([]string, lines)
	for i := range rows {
		if i%7 != 6 {
			rows[i] = fmt.Sprintf("report %02d 界", i)
		}
	}
	return strings.Join(rows, "\n")
}

func codexTallSnapshot(t *testing.T, text, footer string) terminalSnapshot {
	return codexTallSnapshotSize(t, text, footer, 160, 80)
}

func codexTallSnapshotSize(t *testing.T, text, footer string, width, height int) terminalSnapshot {
	t.Helper()
	m := newTerminalModelForTest(t, width, height)
	if err := m.Feed([]byte(codexTallPaint(text, footer))); err != nil {
		t.Fatal(err)
	}
	return m.Snapshot()
}

func TestCodexTallComposerViewportSizes(t *testing.T) {
	for _, width := range []int{80, 100, 160} {
		for _, height := range []int{40, 60, 80} {
			t.Run(fmt.Sprintf("%dx%d", width, height), func(t *testing.T) {
				body := codexTallText(26)
				s := codexTallSnapshotSize(t, body, codexTallTestFooter, width, height)
				if !codexComposerActive(s) {
					t.Fatal("qualified tall composer declined at a supported viewport size")
				}
				if got, ok := qualifiedComposerText("codex", s); !ok || got != body {
					t.Fatalf("text=%q, valid=%v", got, ok)
				}
			})
		}
	}
}

func TestCodexTallComposerGeometryAndText(t *testing.T) {
	for _, lines := range []int{1, 19, 20, 21, 26, 32, 60} {
		for _, blankFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("rows-%d/blank-first-%v", lines, blankFirst), func(t *testing.T) {
				body := codexTallText(lines)
				if blankFirst && lines > 20 {
					body = "\n" + strings.Join(strings.Split(body, "\n")[1:], "\n")
				}
				s := codexTallSnapshot(t, body, codexTallTestFooter)
				if !codexComposerActive(s) {
					t.Fatal("complete native-shaped composer not recognized")
				}
				if got, ok := qualifiedComposerText("codex", s); !ok || got != body {
					t.Fatalf("body=%q, valid=%v; want exact %q without footer", got, ok, body)
				}
			})
		}
	}
}

func TestCodexTallComposerRejectsBrokenAuthority(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*terminalSnapshot)
	}{
		{"hidden cursor", func(s *terminalSnapshot) { s.CursorVisible = false }},
		{"open frame", func(s *terminalSnapshot) { s.Complete = false }},
		{"cursor before text", func(s *terminalSnapshot) { s.Cursor.X = 1 }},
		{"cursor out of bounds", func(s *terminalSnapshot) { s.Cursor.Y = s.Height }},
		{"truncated cells", func(s *terminalSnapshot) { s.Cells = s.Cells[:10] }},
		{"unbold menu prompt", func(s *terminalSnapshot) { s.CellAt(0, 2).Style.Attrs &^= uv.AttrBold }},
		{"clipped historical prompt", func(s *terminalSnapshot) { s.CellAt(0, 2).Content = " " }},
		{"painted prompt spacing", func(s *terminalSnapshot) { s.CellAt(1, 2).Content = "x" }},
		{"nearer output left edge", func(s *terminalSnapshot) { s.CellAt(0, 18).Content = "•" }},
		{"replacement below cursor", func(s *terminalSnapshot) { s.CellAt(0, s.Cursor.Y+1).Content = "›" }},
		{"footer cursor", func(s *terminalSnapshot) { s.Cursor.Y = 35 }},
		{"footer at left edge", func(s *terminalSnapshot) { s.CellAt(0, 35).Content = "x" }},
		{"painted footer separator", func(s *terminalSnapshot) { s.CellAt(2, 34).Content = "x" }},
		{"painted row after footer", func(s *terminalSnapshot) { s.CellAt(2, 39).Content = "x" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := codexTallSnapshot(t, codexTallText(32), codexTallTestFooter)
			tc.edit(&s)
			if codexComposerActive(s) {
				t.Fatal("broken tall evidence granted composer authority")
			}
			if got := peerComposerState("codex", s); got != PeerComposerUnknown {
				t.Fatalf("broken tall evidence = %v, want unknown", got)
			}
			profile, _ := profileForHarness("codex", true)
			if got := decidePlainReturn(profile, false, &s).bytes; !bytes.Equal(got, []byte{'\r'}) {
				t.Fatalf("unknown tall surface remapped Return to %q", got)
			}
		})
	}
	for _, footer := range []string{"", "OpenAI Codex model: GPT-6.1-Sol · /private/tmp/project", "GPT-6.1-Sol medium — /private/tmp/project"} {
		t.Run("unrecognized-footer/"+footer, func(t *testing.T) {
			s := codexTallSnapshot(t, codexTallText(32), footer)
			if codexComposerActive(s) || peerComposerState("codex", s) != PeerComposerUnknown {
				t.Fatal("absent/changed footer wording qualified tall input")
			}
		})
	}
}

func TestEmitPlainCR_CodexTallComposerAndOverlay(t *testing.T) {
	f := newHarnessSessionFake(t, "codex", true)
	t.Cleanup(f.close)
	f.resize(160, 80)
	f.output(codexTallPaint(codexTallText(32), codexTallTestFooter))
	if got := f.enter(); !bytes.Equal(got, []byte{'\n'}) {
		t.Fatalf("tall composer ordinary Return=%q, want LF", got)
	}
	if got := f.altEnter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("tall composer explicit submit=%q, want CR", got)
	}
	f.proxy.pickerActive.Store(true)
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("picker did not override tall composer: %q", got)
	}
}

func TestPeerCodexTallComposerConfirmsDuringExistingTurn(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		busy bool
	}{
		{"short busy", "short", true},
		{"tall singleton busy", codexTallText(25), true},
		{"tall singleton idle", codexTallText(25), false},
		{"batch shaped busy", codexTallBatchText(), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			f := newHarnessSessionFake(t, "codex", true)
			t.Cleanup(f.close)
			f.resize(160, 80)
			m := peerTestMessage(now)
			m.Body = tc.body
			d := newPeerDelivery(m.To, func() time.Time { return now })
			f.proxy.peer = d
			peerRender(f, d, codexTallPaint("", codexTallTestFooter))
			if err := d.reserve(m.ID, d.sequence); err != nil {
				t.Fatal(err)
			}
			if err := d.enqueue(m); err != nil {
				t.Fatal(err)
			}
			f.proxy.turnActive.Store(tc.busy)
			var input bytes.Buffer
			f.proxy.dispatchPeer(&input)
			paste := input.String()
			if paste != "\x1b[200~"+peerEnvelope(m)+"\x1b[201~" {
				t.Fatalf("paste mismatch: %q", paste)
			}
			peerRender(f, d, codexTallPaint(peerEnvelope(m), codexTallTestFooter))
			now = now.Add(PeerSubmitDelay + time.Second)
			f.proxy.dispatchPeer(&input)
			if input.String() != paste+"\r" {
				t.Fatal("expected exactly one submit write")
			}
			peerRender(f, d, codexTallPaint("", codexTallTestFooter))
			f.proxy.turnActive.Store(true)
			f.proxy.dispatchPeer(&input)
			if got := d.receipt(); got.Status != couchmessage.Submitted {
				t.Fatalf("fresh clear after submit not confirmed: %+v", got)
			}
			f.proxy.dispatchPeer(&input)
			if input.String() != paste+"\r" {
				t.Fatal("terminal receipt replayed input")
			}
		})
	}
}

func codexTallBatchText() string {
	messages := make([]couchmessage.Message, 5)
	for i := range messages {
		messages[i] = peerTestMessage(time.Unix(1, 0))
		messages[i].ID = fmt.Sprintf("tall-report-%d", i)
		messages[i].Body = fmt.Sprintf("report %d 界 🦉\nsecond line\nthird line", i)
	}
	return couchmessage.BatchEnvelope(messages)
}

type codexTallPartialWriter struct{ out *bytes.Buffer }

func (w codexTallPartialWriter) Write(data []byte) (int, error) {
	n := len(data) / 2
	_, _ = w.out.Write(data[:n])
	return n, io.ErrShortWrite
}

func TestPeerCodexTallConfirmationRejectsUncertainSequences(t *testing.T) {
	for _, event := range []string{"no occupied repaint", "changed footer", "open frame", "stale clear", "partial paste", "failed submit", "operator input", "picker", "child exit"} {
		t.Run(event, func(t *testing.T) {
			now := time.Now()
			f := newHarnessSessionFake(t, "codex", true)
			t.Cleanup(f.close)
			f.resize(160, 80)
			m := peerTestMessage(now)
			m.Body = codexTallText(25)
			d := newPeerDelivery(m.To, func() time.Time { return now })
			f.proxy.peer = d
			peerRender(f, d, codexTallPaint("", codexTallTestFooter))
			if err := d.reserve(m.ID, d.sequence); err != nil {
				t.Fatal(err)
			}
			if err := d.enqueue(m); err != nil {
				t.Fatal(err)
			}
			f.proxy.turnActive.Store(true)
			var input bytes.Buffer
			var writer io.Writer = &input
			if event == "partial paste" {
				writer = codexTallPartialWriter{&input}
			}
			f.proxy.dispatchPeer(writer)
			if input.Len() == 0 {
				t.Fatal("fixture wrote no paste bytes")
			}
			if event != "no occupied repaint" && event != "partial paste" {
				paint := codexTallPaint(peerEnvelope(m), codexTallTestFooter)
				if event == "changed footer" {
					paint = strings.Replace(paint, codexTallTestFooter, "New Codex model footer", 1)
				}
				if event == "open frame" {
					paint += "\x1b[?2026h"
				}
				peerRender(f, d, paint)
				f.proxy.dispatchPeer(&input) // Observe, before the submit delay.
			}
			switch event {
			case "stale clear":
				peerRender(f, d, codexTallPaint("", codexTallTestFooter))
			case "failed submit":
				writer = peerIntegrationShortWriter{}
			case "operator input":
				d.admitInput([]byte("human"))
			case "picker":
				f.proxy.pickerActive.Store(true)
			case "child exit":
				d.mu.Lock()
				d.exited = true // The runtime's external child-exit observation.
				d.mu.Unlock()
			}
			now = now.Add(PeerSubmitDelay + time.Second)
			f.proxy.dispatchPeer(writer)
			written := input.String()
			if event != "stale clear" {
				peerRender(f, d, codexTallPaint("", codexTallTestFooter)+"\x1b[?2026l")
			}
			f.proxy.dispatchPeer(&input)
			if d.receipt().Status == couchmessage.Submitted {
				t.Fatal("uncertain sequence falsely confirmed submission")
			}
			now = now.Add(couchmessage.DeliveryTimeout)
			f.proxy.dispatchPeer(&input)
			if got := d.receipt(); !got.Status.Terminal() || got.Status == couchmessage.Submitted {
				t.Fatalf("uncertain sequence did not finish conservatively: %+v", got)
			}
			if input.String() != written {
				t.Fatal("uncertain input was replayed")
			}
		})
	}
}
