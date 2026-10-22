package wrapcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

type peerCaptureWriter struct {
	bytes.Buffer
	before func()
	fail   bool
}

func (w *peerCaptureWriter) Write(b []byte) (int, error) {
	if w.before != nil {
		w.before()
	}
	if w.fail {
		return 0, io.ErrClosedPipe
	}
	return w.Buffer.Write(b)
}

func TestPeerWaitCaptureGeneratedSequences(t *testing.T) {
	rng := rand.New(rand.NewSource(791))
	var state PeerWaitCaptureState
	now := time.Unix(1000, 0)
	lastID, captures := "", 0
	for range 10000 {
		id := fmt.Sprint(rng.Intn(4))
		waiting := rng.Intn(5) != 0
		now = now.Add(time.Duration(rng.Intn(25)-3) * time.Second)
		if id != lastID {
			captures = 0
			lastID = id
		}
		previous := state
		var capture bool
		state, capture = advancePeerWaitCapture(state, id, waiting, now)
		if capture {
			captures++
			if !waiting || captures > 1 || previous.id != id || !previous.waiting || now.Sub(previous.since) < 15*time.Second {
				t.Fatalf("unsafe capture: before=%+v after=%+v captures=%d", previous, state, captures)
			}
		}
	}
	// An uninterrupted head must eventually capture; condition changes don't reset it.
	state = PeerWaitCaptureState{}
	state, _ = advancePeerWaitCapture(state, "head", true, now)
	_, capture := advancePeerWaitCapture(state, "head", true, now.Add(15*time.Second))
	if !capture {
		t.Fatal("eligible head never captured")
	}
}

func TestPeerWaitCaptureEncodedScreenAndCap(t *testing.T) {
	s := peerSnapshot(t, claudeBox(5, "❯", "136;136;136", "\x1b[2msuggestion\x1b[22m")+"\x1b[?25l\x1b[7;3H")
	e := PeerWaitEvidence{At: time.Unix(1, 0), Binding: peerTestBinding(), Condition: "waiting for recognized composer: cursor is hidden", ScreenAvailable: true, snapshot: s}
	raw := encodePeerWaitCapture(e, 1)
	var record peerWaitRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	if record.Truncated || record.CursorVisible || !record.Complete || len(record.Rows) != s.Height {
		t.Fatalf("lost screen flags: %+v", record)
	}
	m, err := newTerminalModel(s.Width, s.Height)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	for _, row := range record.Rows {
		if err := m.Feed([]byte(fmt.Sprintf("\x1b[%d;1H%s", row.Y+1, row.ANSI))); err != nil {
			t.Fatal(err)
		}
	}
	replayed := m.Snapshot()
	for i, c := range s.Cells {
		got := replayed.Cells[i]
		if strings.TrimSpace(c.Content) != strings.TrimSpace(got.Content) || !c.Style.Equal(&got.Style) {
			t.Fatalf("cell %d changed: %#v -> %#v", i, c, got)
		}
	}
	huge := strings.Repeat("x", 70000)
	for i := range e.snapshot.Cells {
		e.snapshot.Cells[i].Content = huge
	}
	raw = encodePeerWaitCapture(e, 2)
	if len(raw) == 0 || len(raw) > diagnosticlog.BatchMaxRecordBytes || json.Unmarshal(raw, &record) != nil || !record.Truncated {
		t.Fatalf("missing bounded truncation: %d", len(raw))
	}
}

func TestPeerWaitCaptureConditions(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		setup      func(*harnessSessionFake, *peerDelivery, time.Time)
	}{
		{"picker", "picker", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) { f.proxy.pickerActive.Store(true) }},
		{"automatic", "previous automatic input", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) {
			f.proxy.automaticInput.phase = automaticInputDrive
		}},
		{"forwarded", "forwarded input", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) { d.replies.inFlight = 1 }},
		{"partial", "incomplete terminal input", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) { d.replies.input = []byte("\x1b[") }},
		{"output", "rendered output", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) {
			f.output("\x1b[?2026h")
			d.outputPending = 1
		}},
		{"buffered", "buffered input", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) { d.inputBuffered = true }},
		{"image", "image input", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) { d.image = true }},
		{"no-terminal", "terminal state", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) { f.proxy.terminal = nil }},
		{"recent", "input to settle", func(f *harnessSessionFake, d *peerDelivery, now time.Time) { d.lastInput = now }},
		{"held", "held operator input", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) { d.holding = true }},
		{"paste", "paste mode", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) { f.output("\x1b[?2004l") }},
		{"draft", "empty composer", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) {
			f.output(peerFilledComposer("operator draft"))
		}},
		{"hidden", "cursor is hidden", func(f *harnessSessionFake, d *peerDelivery, _ time.Time) { f.output("\x1b[?25l") }},
		{"view", "composer to be in view", func(f *harnessSessionFake, d *peerDelivery, now time.Time) {
			d.viewProbe = func() bool { return false }
			d.view = peerView{at: now, visible: false}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			terminal := f.proxy.terminal
			t.Cleanup(func() { f.proxy.terminal = terminal })
			now := time.Unix(100, 0)
			d.now = func() time.Time { return now }
			peerIntegrationEnqueue(t, d)
			var sink peerCaptureWriter
			var input bytes.Buffer
			for i := range 2 {
				if i > 0 {
					now = now.Add(15 * time.Second)
				}
				tc.setup(f, d, now)
				sink.Reset()
				f.proxy.wrapEventsFD = &sink
				f.proxy.dispatchPeer(&input)
			}
			var record peerWaitRecord
			if err := json.Unmarshal(bytes.TrimSpace(sink.Bytes()), &record); err != nil {
				t.Fatalf("no capture for %s: %v %q", tc.name, err, sink.String())
			}
			if !strings.Contains(record.Evidence.Condition, tc.want) || input.Len() != 0 || d.receipt().Status.Terminal() {
				t.Fatalf("changed refusal: %+v input=%q", record.Evidence, input.String())
			}
		})
	}
}

func TestPeerWaitCaptureCountedHeadsAndPolls(t *testing.T) {
	for _, heads := range []int{8, 16} {
		t.Run(fmt.Sprint(heads), func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			now := time.Unix(100, 0)
			d.now = func() time.Time { return now }
			f.output("\x1b[?25l")
			var sink peerCaptureWriter
			f.proxy.wrapEventsFD = &sink
			var input bytes.Buffer
			for head := range heads {
				m := peerTestMessage(now)
				m.ID = fmt.Sprintf("capture-%d", head)
				d.current = couchmessage.Receipt{Message: m, Status: couchmessage.Queued}
				f.proxy.dispatchPeer(&input)
				now = now.Add(15 * time.Second)
				f.proxy.dispatchPeer(&input)
				for range heads {
					now = now.Add(time.Second)
					f.proxy.dispatchPeer(&input)
				}
			}
			if got := bytes.Count(sink.Bytes(), []byte(`"label":"PEER-wait"`)); got != heads {
				t.Fatalf("captures=%d heads=%d", got, heads)
			}
			if input.Len() != 0 || sink.Len() > heads*diagnosticlog.BatchMaxRecordBytes {
				t.Fatal("unbounded diagnostic or input effects")
			}
		})
	}
}

func FuzzEncodePeerWaitCapture(f *testing.F) {
	f.Add([]byte("screen\x00\"\\\n"), uint8(40), uint8(20))
	f.Add([]byte("你好👩‍💻"), uint8(120), uint8(38))
	f.Fuzz(func(t *testing.T, data []byte, w, h uint8) {
		width, height := int(w)+1, int(h)+1
		if len(data) > 4096 {
			data = data[:4096]
		}
		s := terminalSnapshot{Width: width, Height: height, Cells: make([]uv.Cell, width*height)}
		content := strings.ToValidUTF8(string(data), "�")
		for i := range s.Cells {
			s.Cells[i] = uv.Cell{Content: content, Width: 1, Style: uv.Style{Attrs: uint8(i)}}
		}
		e := PeerWaitEvidence{At: time.Unix(1, 0), Binding: peerTestBinding(), Condition: string(data), snapshot: s, ScreenAvailable: true}
		raw := encodePeerWaitCapture(e, 1)
		var r peerWaitRecord
		if len(raw) == 0 || len(raw) > diagnosticlog.BatchMaxRecordBytes || json.Unmarshal(raw, &r) != nil {
			t.Fatalf("invalid bounded record: %d", len(raw))
		}
		if len(r.Rows) != height && !r.Truncated {
			t.Fatal("silent truncation")
		}
	})
}

func TestPeerWaitCapturePersonalitiesAndBatch(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "qoder", "claude-batch"} {
		t.Run(agent, func(t *testing.T) {
			var f *harnessSessionFake
			var d *peerDelivery
			now := time.Unix(100, 0)
			if agent == "claude-batch" {
				var clock *time.Time
				f, d, _, clock = peerBatchFixture(t)
				now = *clock
				d.now = func() time.Time { return now }
			} else {
				f = newHarnessSessionFake(t, agent, true)
				t.Cleanup(f.close)
				paint := claudeBox(5, "❯", "136;136;136", "") + "\x1b[7;3H"
				if agent == "codex" {
					paint = "\x1b[20;1H\x1b[1m›\x1b[22m \x1b[20;3H"
				}
				if agent == "qoder" {
					paint = peerQoderPaint(">", "operator draft")
				}
				f.output("\x1b[?2004h" + paint + "\x1b[?25h")
				binding := peerTestBinding()
				binding.Agent = agent
				d = newPeerDelivery(binding, func() time.Time { return now })
				f.proxy.peer = d
				m := peerTestMessage(now)
				m.To = binding
				if err := d.reserve(m.ID, d.sequence); err != nil {
					t.Fatal(err)
				}
				if err := d.enqueue(m); err != nil {
					t.Fatal(err)
				}
			}
			if agent != "qoder" {
				f.output("\x1b[?25l")
			}
			var sink peerCaptureWriter
			f.proxy.wrapEventsFD = &sink
			var input bytes.Buffer
			f.proxy.dispatchPeer(&input)
			now = now.Add(15 * time.Second)
			f.proxy.dispatchPeer(&input)
			var r peerWaitRecord
			if err := json.Unmarshal(sink.Bytes(), &r); err != nil {
				t.Fatal(err)
			}
			if r.Evidence.Condition != d.receipt().Detail || input.Len() != 0 {
				t.Fatalf("capture changed admission: %+v", r.Evidence)
			}
			if agent == "claude-batch" && d.batchReady() {
				t.Fatal("unrecognized composer made batch ready")
			}
		})
	}
}

func TestPeerWaitCaptureFailureDisabledAndNewHead(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(fmt.Sprint(disabled), func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			now := time.Unix(100, 0)
			d.now = func() time.Time { return now }
			f.output("\x1b[?25l")
			calls := 0
			var sink peerCaptureWriter
			sink.fail = true
			sink.before = func() { calls++ }
			if !disabled {
				f.proxy.wrapEventsFD = &sink
			}
			peerIntegrationEnqueue(t, d)
			var input bytes.Buffer
			f.proxy.dispatchPeer(&input)
			now = now.Add(15 * time.Second)
			f.proxy.dispatchPeer(&input)
			for range 8 {
				now = now.Add(time.Second)
				f.proxy.dispatchPeer(&input)
			}
			want := 1
			if disabled {
				want = 0
			}
			if calls != want || input.Len() != 0 || d.receipt().Status.Terminal() {
				t.Fatalf("failure changed safety: calls=%d receipt=%+v", calls, d.receipt())
			}
			d.released = true
			f.proxy.dispatchPeer(&input)
			m := peerTestMessage(now)
			m.ID = "new-head"
			if err := d.reserve(m.ID, d.sequence); err != nil {
				t.Fatal(err)
			}
			if err := d.enqueue(m); err != nil {
				t.Fatal(err)
			}
			f.proxy.dispatchPeer(&input)
			now = now.Add(15 * time.Second)
			f.proxy.dispatchPeer(&input)
			if calls != 2*want {
				t.Fatalf("new head did not reset capture: calls=%d", calls)
			}
		})
	}
}

func TestPeerWaitCaptureManagedFile(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	now := time.Unix(100, 0)
	d.now = func() time.Time { return now }
	f.output("\x1b[?25l")
	peerIntegrationEnqueue(t, d)
	t.Setenv("PAIR_WRAP_EVENTS", "1")
	path := filepath.Join(t.TempDir(), "wait-events.jsonl")
	f.proxy.wrapEventsPath = path
	f.proxy.diagnosticOptions = func() diagnosticlog.Options {
		return diagnosticlog.Options{FlushTicks: make(chan time.Time), ObserveIO: func(effect diagnosticlog.IOEffect) {
			if !d.mu.TryLock() {
				t.Error("file IO under mailbox lock")
			} else {
				d.mu.Unlock()
			}
			if !f.proxy.inputAdmission.TryLock() {
				t.Error("file IO under admission lock")
			} else {
				f.proxy.inputAdmission.Unlock()
			}
		}}
	}
	f.proxy.openWrapEvents(nil)
	writer := f.proxy.wrapEventsFD.(*diagnosticlog.BufferedWriter)
	t.Cleanup(func() { _ = f.proxy.closeWrapEvents() })
	var input bytes.Buffer
	f.proxy.dispatchPeer(&input)
	now = now.Add(15 * time.Second)
	f.proxy.dispatchPeer(&input)
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(raw, []byte(`"label":"PEER-wait"`)) != 1 || input.Len() != 0 {
		t.Fatal("managed writer lost capture or changed input")
	}
}
func (w *peerCaptureWriter) Close() error { return nil }

func TestPeerWaitCaptureForcedRefusal(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	now := time.Unix(100, 0)
	d.now = func() time.Time { return now }
	var diagnostic peerCaptureWriter
	diagnostic.before = func() {
		if !d.mu.TryLock() {
			t.Error("diagnostic writer called with mailbox locked")
		} else {
			d.mu.Unlock()
		}
		if !f.proxy.inputAdmission.TryLock() {
			t.Error("diagnostic writer called with admission locked")
		} else {
			f.proxy.inputAdmission.Unlock()
		}
	}
	f.proxy.wrapEventsFD = &diagnostic
	f.output("\x1b[?25l")
	diagnostic.Reset()
	peerIntegrationEnqueue(t, d)
	var input bytes.Buffer
	f.proxy.dispatchPeer(&input)
	now = now.Add(15*time.Second - time.Nanosecond)
	f.proxy.dispatchPeer(&input)
	if diagnostic.Len() != 0 {
		t.Fatal("capture before threshold")
	}
	now = now.Add(time.Nanosecond)
	f.proxy.dispatchPeer(&input)
	if input.Len() != 0 || !strings.Contains(d.receipt().Detail, "cursor is hidden") {
		t.Fatalf("forced refusal lost: input=%q receipt=%+v", input.String(), d.receipt())
	}
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(diagnostic.Bytes()), &record); err != nil {
		t.Fatalf("missing capture at threshold: %v, bytes=%q", err, diagnostic.String())
	}
	if record["label"] != "PEER-wait" || !strings.Contains(diagnostic.String(), "cursor is hidden") {
		t.Fatalf("capture omitted condition: %s", diagnostic.String())
	}
	first := diagnostic.String()
	for range 16 {
		now = now.Add(time.Second)
		f.proxy.dispatchPeer(&input)
	}
	if diagnostic.String() != first {
		t.Fatal("repeated capture for same head")
	}
	f.output("\x1b[?25h")
	f.proxy.dispatchPeer(&input)
	if input.Len() == 0 {
		t.Fatal("diagnostics prevented normal recovery")
	}
}

func TestPeerWaitCaptureFittingStyledRow(t *testing.T) {
	s := terminalSnapshot{Width: 2000, Height: 1, Cells: make([]uv.Cell, 2000)}
	for i := range s.Cells {
		s.Cells[i] = uv.Cell{Content: "a", Width: 1}
		if i%2 == 1 {
			s.Cells[i].Style.Attrs = uv.AttrBold
		}
	}
	e := PeerWaitEvidence{Binding: peerTestBinding(), ScreenAvailable: true, snapshot: s}
	row, ok := peerWaitScreenRow(s, 0, diagnosticlog.BatchMaxRecordBytes)
	if !ok {
		t.Fatal("reference row unexpectedly over budget")
	}
	golden, _ := json.Marshal(row)
	if len(golden) > 60000 {
		t.Fatal("counterexample does not fit")
	}
	var r peerWaitRecord
	raw := encodePeerWaitCapture(e, 1)
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	if r.Truncated || len(r.Rows) != 1 || strings.Count(r.Rows[0].ANSI, "a") != 2000 {
		t.Fatalf("fitting %d-byte JSON row was discarded: rows=%d truncated=%v record=%d", len(golden), len(r.Rows), r.Truncated, len(raw))
	}
}
