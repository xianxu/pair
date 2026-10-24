package couchtty

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

type submissionCastFake struct {
	feedback           *submissionPreviewFake // initialized only by explicit transaction preview setup
	gate               chan struct{}
	fail               error
	resolveFail        error
	empty              bool
	mu                 sync.Mutex
	prepared, resolved int
	path               string
	onPrepare          func(func(core.SubmissionProgress))
	onSelection        func(board.Selection)
}

func (f *submissionCastFake) RecordingPath() string { return f.path }
func (f *submissionCastFake) SubmissionSelection() (board.Selection, error) {
	if f.empty {
		return board.Selection{Document: 1, TabID: "tab"}, nil
	}
	return board.Selection{Document: 1, TabID: "tab", Notes: []board.SelectedNote{{ID: "note"}}}, nil
}
func (f *submissionCastFake) PrepareSubmission(ctx context.Context, id string, target core.Target, s board.Selection, base string, progress func(core.SubmissionProgress)) (core.Submission, map[string][]byte, error) {
	f.mu.Lock()
	f.prepared++
	f.mu.Unlock()
	if f.onSelection != nil {
		f.onSelection(s)
	}
	if f.onPrepare != nil {
		f.onPrepare(progress)
	}
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return core.Submission{}, nil, ctx.Err()
		}
	}
	if f.fail != nil {
		return core.Submission{}, nil, f.fail
	}
	return core.Submission{HowToRead: core.ReadingGuide(f.path), Version: 1, ID: id, Target: target, Source: core.Source{Kind: "dom", RecordingID: "rec", RecordingPath: f.path, TabID: "tab", Document: 1}, Delivery: core.Prepared, Notes: []core.Note{{ID: "note", Text: "fix", Crop: core.Crop{Unavailable: "page changed"}}}}, nil, nil
}
func (f *submissionCastFake) ResolveSubmission(ctx context.Context, s core.Submission) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resolved++
	return f.resolveFail
}

func TestSubmissionInlineSchedulerDoesNotHoldManagerLock(t *testing.T) {
	m, feedbackCast, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, feedbackCast)
	m.spawn = func(fn func()) {
		if !m.mu.TryLock() {
			t.Error("worker scheduled while manager lock held: inline scheduler deadlocks")
			return
		}
		m.mu.Unlock()
		fn()
	}
	if _, err := m.request(context.Background(), "prepare", "one", target); err != nil {
		t.Fatal(err)
	}
	if t.Failed() {
		return
	}
	submissionWait(t, m, "one", target, "prepared")
	if _, err := m.request(context.Background(), "begin", "one", target); err != nil {
		t.Fatal(err)
	}
	if _, err := m.request(context.Background(), "ack", "one", target); err != nil {
		t.Fatal(err)
	}
	if !t.Failed() {
		submissionWait(t, m, "one", target, "resolved")
	}
}

func TestSubmissionUnknownOperationCannotReadCachedReceipt(t *testing.T) {
	m, _, target := newSubmissionManagerTest(t)
	m.current = func() (submissionCast, error) { return nil, nil }
	if _, err := m.request(context.Background(), "prepare", "one", target); err != nil {
		t.Fatal(err)
	}
	if _, err := m.request(context.Background(), "unknown", "one", target); err == nil {
		t.Fatal("unknown operation accepted cached empty receipt")
	}
}
func newSubmissionManagerTest(t *testing.T) (*submissionManager, *submissionCastFake, core.Target) {
	t.Helper()
	st, e := core.NewStore(filepath.Join(t.TempDir(), "submissions"))
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	f := &submissionCastFake{path: filepath.Join(t.TempDir(), "rec")}
	target := core.Target{Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce"}
	m := &submissionManager{store: st, ctx: ctx, spawn: func(fn func()) { go fn() }, current: func() (submissionCast, error) { return f, nil }, guard: func(core.Target) error { return nil }, replay: func(context.Context, string) (string, error) { return "https://replay", nil }}
	return m, f, target
}
func submissionWait(t *testing.T, m *submissionManager, id string, target core.Target, state string) couchmessage.SubmissionStatus {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		r, e := m.request(context.Background(), "status", id, target)
		if e != nil {
			t.Fatal(e)
		}
		if r.State == state {
			return r
		}
		if r.State == "failed" {
			t.Fatal(r.Error)
		}
		select {
		case <-deadline:
			t.Fatalf("state %s wanted %s", r.State, state)
		case <-time.After(time.Millisecond):
		}
	}
}
func TestSubmissionSinglePreparationAndAckOnly(t *testing.T) {
	m, f, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, f)
	f.gate = make(chan struct{})
	r, e := m.request(context.Background(), "prepare", "one", target)
	if e != nil || r.State != "preparing" {
		t.Fatalf("%+v %v", r, e)
	}
	if _, e = m.request(context.Background(), "prepare", "two", target); e == nil {
		t.Fatal("parallel prep accepted")
	}
	if _, e = m.request(context.Background(), "prepare", "one", target); e != nil {
		t.Fatal(e)
	}
	close(f.gate)
	submissionWait(t, m, "one", target, "prepared")
	if _, e = m.request(context.Background(), "ack", "one", target); e == nil {
		t.Fatal("unsent acknowledgment accepted")
	}
	if _, e = m.request(context.Background(), "begin", "one", target); e != nil {
		t.Fatal(e)
	}
	if _, e = m.request(context.Background(), "ack", "one", target); e != nil {
		t.Fatal(e)
	}
	submissionWait(t, m, "one", target, "resolved")
	if _, e = m.request(context.Background(), "ack", "one", target); e != nil {
		t.Fatal(e)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.prepared != 1 || f.resolved != 1 {
		t.Fatalf("prep=%d resolve=%d", f.prepared, f.resolved)
	}
}
func TestSubmissionDurableUncertaintyAndTargetGuard(t *testing.T) {
	m, feedbackCast, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, feedbackCast)
	m.request(context.Background(), "prepare", "one", target)
	submissionWait(t, m, "one", target, "prepared")
	m.guard = func(core.Target) error { return errors.New("active agent changed") }
	if _, e := m.request(context.Background(), "begin", "one", target); e == nil {
		t.Fatal("changed view accepted")
	}
	m.guard = func(core.Target) error { return nil }
	m.request(context.Background(), "begin", "one", target)
	m.job = nil // nvim/Couch restart loses volatile receipt, not durable evidence.
	if _, e := m.request(context.Background(), "prepare", "two", target); e != nil {
		t.Fatal("uncertain delivery blocked fresh submission", e)
	}
	submissionWait(t, m, "two", target, "prepared")
	other := target
	other.Nonce = "replacement"
	if _, e := m.request(context.Background(), "ack", "one", other); e == nil {
		t.Fatal("replacement acknowledged old binding")
	}
}

func submissionWorkQueue(t *testing.T, m *submissionManager) func() {
	t.Helper()
	var queue []func()
	m.spawn = func(fn func()) { queue = append(queue, fn) }
	return func() {
		t.Helper()
		if len(queue) != 1 {
			t.Fatalf("queued workers=%d, want one", len(queue))
		}
		fn := queue[0]
		queue = nil
		fn()
	}
}

func TestSubmissionEmptyDoesNotRequestReplayOrCapture(t *testing.T) {
	for _, noCast := range []bool{false, true} {
		m, f, target := newSubmissionManagerTest(t)
		f.empty = true
		if noCast {
			m.current = func() (submissionCast, error) { return nil, nil }
		}
		m.replay = func(context.Context, string) (string, error) {
			t.Fatal("empty submission requested replay")
			return "", nil
		}
		r, err := m.request(context.Background(), "prepare", "one", target)
		if err != nil || r.State != "empty" || r.Target != target {
			t.Fatalf("empty: %+v %v", r, err)
		}
		if _, err := m.store.Read("one"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("empty submission wrote evidence: %v", err)
		}
		if f.prepared != 0 {
			t.Fatal("empty submission captured")
		}
	}
}

func TestSubmissionPreparationFailureRetainedAndNewAttemptRetries(t *testing.T) {
	for _, failure := range []string{"replay", "capture", "target"} {
		t.Run(failure, func(t *testing.T) {
			m, f, target := newSubmissionManagerTest(t)
			acknowledgeSubmissionFeedback(t, m, f)
			run := submissionWorkQueue(t, m)
			if failure == "capture" {
				f.fail = errors.New("capture failed")
			}
			if failure == "replay" {
				m.replay = func(context.Context, string) (string, error) { return "", errors.New("replay failed") }
			}
			if _, err := m.request(context.Background(), "prepare", "one", target); err != nil {
				t.Fatal(err)
			}
			if failure == "target" {
				m.guard = func(core.Target) error { return errors.New("target changed") }
			}
			run()
			for _, op := range []string{"status", "prepare"} {
				r, err := m.request(context.Background(), op, "one", target)
				if err != nil || r.State != "failed" || r.Error == "" {
					t.Fatalf("failure receipt: %+v %v", r, err)
				}
			}
			if _, err := m.store.Read("one"); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failure published bundle: %v", err)
			}
			f.fail = nil
			m.guard = func(core.Target) error { return nil }
			m.replay = func(context.Context, string) (string, error) { return "https://replay", nil }
			if _, err := m.request(context.Background(), "prepare", "two", target); err != nil {
				t.Fatal(err)
			}
			run()
			submissionWait(t, m, "two", target, "prepared")
		})
	}
}

func TestSubmissionResolveFailureRetriesAckWithoutRecapture(t *testing.T) {
	m, f, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, f)
	run := submissionWorkQueue(t, m)
	if _, err := m.request(context.Background(), "prepare", "one", target); err != nil {
		t.Fatal(err)
	}
	run()
	if _, err := m.request(context.Background(), "begin", "one", target); err != nil {
		t.Fatal(err)
	}
	f.resolveFail = errors.New("note log unavailable")
	if _, err := m.request(context.Background(), "ack", "one", target); err != nil {
		t.Fatal(err)
	}
	run()
	r, err := m.request(context.Background(), "status", "one", target)
	if err != nil || r.State != "dispatched" || r.Error != "note log unavailable" {
		t.Fatalf("resolve failure: %+v %v", r, err)
	}
	f.resolveFail = nil
	if _, err := m.request(context.Background(), "ack", "one", target); err != nil {
		t.Fatal(err)
	}
	run()
	submissionWait(t, m, "one", target, "resolved")
	if f.prepared != 1 || f.resolved != 2 {
		t.Fatalf("capture=%d resolution=%d", f.prepared, f.resolved)
	}
}

func TestSubmissionCancellationBeforeWorkerStarts(t *testing.T) {
	for _, phase := range []string{"prepare", "ack"} {
		t.Run(phase, func(t *testing.T) {
			m, f, target := newSubmissionManagerTest(t)
			acknowledgeSubmissionFeedback(t, m, f)
			run := submissionWorkQueue(t, m)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m.ctx = ctx
			if _, err := m.request(context.Background(), "prepare", "one", target); err != nil {
				t.Fatal(err)
			}
			if phase == "ack" {
				run()
				if _, err := m.request(context.Background(), "begin", "one", target); err != nil {
					t.Fatal(err)
				}
				if _, err := m.request(context.Background(), "ack", "one", target); err != nil {
					t.Fatal(err)
				}
			}
			m.replay = func(context.Context, string) (string, error) {
				t.Error("started replay after cancellation")
				return "https://replay", nil
			}
			cancel()
			run()
			if m.job.busy || m.job.status.Error == "" {
				t.Fatalf("cancelled worker left running: %+v", m.job)
			}
			if phase == "prepare" {
				if f.prepared != 0 {
					t.Fatal("captured after cancellation")
				}
				if _, err := m.store.Read("one"); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("cancelled capture published: %v", err)
				}
			} else {
				if f.resolved != 0 {
					t.Fatal("resolved notes after cancellation")
				}
				s, err := m.store.Read("one")
				if err != nil || s.Delivery != core.Dispatched {
					t.Fatalf("lost delivery truth: %+v %v", s, err)
				}
				m.ctx = context.Background()
				m.job = nil
				if _, err := m.request(context.Background(), "ack", "one", target); err != nil {
					t.Fatal(err)
				}
				run()
				submissionWait(t, m, "one", target, "resolved")
				if f.prepared != 1 || f.resolved != 1 {
					t.Fatalf("restart repeated wrong work: capture=%d resolve=%d", f.prepared, f.resolved)
				}
			}
		})
	}
}

func TestSubmissionAckDoesNotResolveReplacementCast(t *testing.T) {
	m, f, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, f)
	run := submissionWorkQueue(t, m)
	if _, err := m.request(context.Background(), "prepare", "one", target); err != nil {
		t.Fatal(err)
	}
	run()
	if _, err := m.request(context.Background(), "begin", "one", target); err != nil {
		t.Fatal(err)
	}
	replacement := &submissionCastFake{path: filepath.Join(t.TempDir(), "other-recording")}
	m.current = func() (submissionCast, error) { return replacement, nil }
	if _, err := m.request(context.Background(), "ack", "one", target); err != nil {
		t.Fatal(err)
	}
	run()
	submissionWait(t, m, "one", target, "resolved")
	if f.resolved != 0 || replacement.resolved != 0 {
		t.Fatal("acknowledgment changed a replacement cast")
	}
}

func TestSubmissionEveryOperationKeepsExactTargetAcrossRestart(t *testing.T) {
	m, feedbackCast, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, feedbackCast)
	run := submissionWorkQueue(t, m)
	if _, err := m.request(context.Background(), "prepare", "one", target); err != nil {
		t.Fatal(err)
	}
	check := func() {
		t.Helper()
		for _, field := range []string{"scope", "tag", "session", "nonce"} {
			other := target
			switch field {
			case "scope":
				other.Scope = "other"
			case "tag":
				other.Tag = "other"
			case "session":
				other.Session = "other"
			case "nonce":
				other.Nonce = "other"
			}
			for _, op := range []string{"prepare", "status", "begin", "ack"} {
				if _, err := m.request(context.Background(), op, "one", other); err == nil {
					t.Fatalf("%s accepted changed %s", op, field)
				}
			}
		}
	}
	check() // volatile preparing receipt
	run()
	m.job = nil
	check() // durable prepared bundle after restart
	sub, err := m.store.ReadMetadata("one")
	if err != nil || sub.Delivery != core.Prepared {
		t.Fatalf("refusal changed delivery: %+v %v", sub, err)
	}
}

func TestSubmissionClientCancellationDoesNotAbandonAdmittedWork(t *testing.T) {
	m, feedbackCast, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, feedbackCast)
	run := submissionWorkQueue(t, m)
	requestCtx, cancel := context.WithCancel(context.Background())
	if _, err := m.request(requestCtx, "prepare", "one", target); err != nil {
		t.Fatal(err)
	}
	cancel()
	run()
	submissionWait(t, m, "one", target, "prepared")
	if _, err := m.request(requestCtx, "begin", "one", target); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled admission: %v", err)
	}
	sub, err := m.store.ReadMetadata("one")
	if err != nil || sub.Delivery != core.Prepared {
		t.Fatalf("cancelled begin changed state: %+v %v", sub, err)
	}
}

func TestSubmissionStatusForwardsStoredReadingGuide(t *testing.T) {
	m, _, target := newSubmissionManagerTest(t)
	guide := core.ReadingGuide("/recording path")
	sub := core.Submission{ID: "guide", Target: target, Delivery: core.Prepared, HowToRead: guide}
	if got := m.status(sub); got.HowToRead != guide {
		t.Fatalf("lost bundle guide: %#v", got)
	}
}

func TestSubmissionPreparedAndStatusPreserveActualReadingGuide(t *testing.T) {
	m, cast, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, cast)
	cast.path = filepath.Join(t.TempDir(), "recordings with spaces", "rec")
	originalPath := cast.path
	want := core.ReadingGuide(originalPath)
	run := submissionWorkQueue(t, m)
	if _, err := m.request(context.Background(), "prepare", "guide", target); err != nil {
		t.Fatal(err)
	}
	run()
	stored, err := m.store.Read("guide")
	if err != nil {
		t.Fatal(err)
	}
	if stored.HowToRead != want {
		t.Fatalf("prepared bundle guide = %q, want %q", stored.HowToRead, want)
	}
	// Both cached and restarted requests must forward the stored prose exactly,
	// even when the current cast points elsewhere. Neither may recapture it.
	for _, restarted := range []bool{false, true} {
		if restarted {
			m.job = nil
			cast.path = filepath.Join(t.TempDir(), "replacement")
		}
		for _, op := range []string{"prepare", "status"} {
			status, err := m.request(context.Background(), op, "guide", target)
			if err != nil {
				t.Fatal(op, err)
			}
			if status.State != "prepared" || status.HowToRead != want {
				t.Fatalf("%s after restart=%v changed guide: %+v", op, restarted, status)
			}
			if !filepath.IsAbs(status.Path) || !strings.Contains(status.HowToRead, originalPath) {
				t.Fatalf("%s lost absolute paths: %+v", op, status)
			}
			for _, fragment := range []string{"submission.json", "crop-N.png", "couch --broadcast-web-query", "notes|dom|events|link", "dom --at <t> --sel @N", "events --from <t-10000> --to <t>", "link --at <t>", "couch --skill"} {
				if !strings.Contains(status.HowToRead, fragment) {
					t.Fatalf("%s omitted %q", op, fragment)
				}
			}
			wire, err := json.Marshal(status)
			if err != nil {
				t.Fatal(err)
			}
			var response couchmessage.SubmissionStatus
			if err = json.Unmarshal(wire, &response); err != nil {
				t.Fatal(err)
			}
			if response.HowToRead != stored.HowToRead {
				t.Fatalf("RPC JSON changed guide: %q", response.HowToRead)
			}
		}
	}
	if cast.prepared != 1 {
		t.Fatalf("reading guide was regenerated by %d preparations", cast.prepared)
	}
}

func TestSubmissionStatusRecoversAckDuringBusySuccessor(t *testing.T) {
	m, f, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, f)
	run := submissionWorkQueue(t, m)
	if _, err := m.request(context.Background(), "prepare", "one", target); err != nil {
		t.Fatal(err)
	}
	run()
	if _, err := m.request(context.Background(), "begin", "one", target); err != nil {
		t.Fatal(err)
	}
	acknowledgeSubmissionFeedback(t, m, f)
	if _, err := m.request(context.Background(), "prepare", "two", target); err != nil {
		t.Fatal(err)
	}
	if got, err := m.request(context.Background(), "ack", "one", target); err != nil || got.State != "dispatched" {
		t.Fatalf("late ack %+v %v", got, err)
	}
	run() // finishes successor, without a parallel resolution worker
	f.resolveFail = errors.New("resolution temporarily unavailable")
	if _, err := m.request(context.Background(), "status", "one", target); err != nil {
		t.Fatal(err)
	}
	run()
	f.resolveFail = nil
	m.job = nil // restart loses volatile failure receipt
	if _, err := m.request(context.Background(), "status", "one", target); err != nil {
		t.Fatal(err)
	}
	run()
	if got, err := m.request(context.Background(), "ack", "one", target); err != nil || got.State != "resolved" {
		t.Fatalf("recovered ack %+v %v", got, err)
	}
	if f.prepared != 2 || f.resolved != 2 {
		t.Fatalf("capture=%d resolution=%d", f.prepared, f.resolved)
	}
}

func TestSubmissionPreparationPublishesInitialProgress(t *testing.T) {
	m, feedbackCast, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, feedbackCast)
	run := submissionWorkQueue(t, m)
	status, err := m.request(context.Background(), "prepare", "progress", target)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Progress struct {
			Phase string `json:"phase"`
		} `json:"progress"`
	}
	if err = json.Unmarshal(wire, &got); err != nil {
		t.Fatal(err)
	}
	if got.Progress.Phase != "replay" {
		t.Fatalf("initial progress phase=%q, want replay", got.Progress.Phase)
	}
	run()
	submissionWait(t, m, "progress", target, "prepared")
}

func TestSubmissionProgressHeldAndFenced(t *testing.T) {
	m, f, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, f)
	replayEntered, replayRelease := make(chan struct{}), make(chan struct{})
	m.replay = func(ctx context.Context, _ string) (string, error) {
		close(replayEntered)
		select {
		case <-replayRelease:
			return "https://replay", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	callbacks := make(chan func(core.SubmissionProgress), 1)
	f.gate = make(chan struct{})
	f.onPrepare = func(report func(core.SubmissionProgress)) {
		report(core.SubmissionProgress{Phase: "settling", Current: 1, Total: 3})
		callbacks <- report
	}
	if _, err := m.request(context.Background(), "prepare", "held", target); err != nil {
		t.Fatal(err)
	}
	<-replayEntered
	replay, err := m.request(context.Background(), "status", "held", target)
	if err != nil || replay.Progress == nil || replay.Progress.Phase != "replay" {
		t.Fatalf("held replay %+v %v", replay, err)
	}
	close(replayRelease)
	report := <-callbacks
	held, err := m.request(context.Background(), "status", "held", target)
	if err != nil || held.Progress == nil || *held.Progress != (core.SubmissionProgress{Phase: "settling", Current: 1, Total: 3}) {
		t.Fatalf("held capture %+v %v", held, err)
	}
	report(core.SubmissionProgress{Phase: "capturing", Current: 2, Total: 3})
	if held.Progress.Phase != "settling" {
		t.Fatal("progress mutated an already returned snapshot")
	}
	close(f.gate)
	done := submissionWait(t, m, "held", target, "prepared")
	report(core.SubmissionProgress{Phase: "capturing", Current: 3, Total: 3})
	if done.Progress != nil {
		t.Fatal("terminal status retained preparation progress")
	}

	successor := &submissionCastFake{path: f.path, gate: make(chan struct{})}
	m.current = func() (submissionCast, error) { return successor, nil }
	acknowledgeSubmissionFeedback(t, m, successor)
	m.replay = func(context.Context, string) (string, error) { return "https://replay", nil }
	if _, err := m.request(context.Background(), "prepare", "next", target); err != nil {
		t.Fatal(err)
	}
	report(core.SubmissionProgress{Phase: "publishing"})
	next, err := m.request(context.Background(), "status", "next", target)
	if err != nil || next.Progress == nil || next.Progress.Phase != "replay" {
		t.Fatalf("retired callback changed successor %+v %v", next, err)
	}
	prior, err := m.request(context.Background(), "status", "held", target)
	if err != nil || prior.Progress != nil {
		t.Fatalf("retired callback revived terminal progress %+v %v", prior, err)
	}
	close(successor.gate)
	submissionWait(t, m, "next", target, "prepared")
}
