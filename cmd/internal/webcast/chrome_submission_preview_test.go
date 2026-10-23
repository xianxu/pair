package webcast

import (
	"context"
	"encoding/json"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
	"runtime"
	"strings"
	"testing"
)

func previewChromeFixture(t *testing.T) (*Chrome, *loopState) {
	t.Helper()
	b := board.New(notes.Structural{}, nil)
	t.Cleanup(func() { b.Close() })
	c := &Chrome{ctx: context.Background(), opts: LaunchOptions{Record: &RecordOptions{Operator: b, ID: "rec"}}}
	s := &loopState{sessions: map[string]string{"s": "tab"}, contexts: map[string]int64{"s": 17}}
	s.tracker.Apply(TargetCreated{ID: "tab", URL: "https://example.test/"})
	return c, s
}
func TestSubmissionPreviewHostTracksExactActiveLifetime(t *testing.T) {
	c, s := previewChromeFixture(t)
	c.refreshSubmissionPreviewHost(s)
	first, life, err := c.SubmissionPreviewHost(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.Generation == 0 || life == nil {
		t.Fatalf("host identity unavailable: %#v", first)
	}
	c.refreshSubmissionPreviewHost(s)
	same, _, _ := c.SubmissionPreviewHost(context.Background())
	if same != first {
		t.Fatal("unchanged context generated new host")
	}
	c.retireOperatorSketchScopes(s, "s")
	c.refreshSubmissionPreviewHost(s)
	newer, newLife, err := c.SubmissionPreviewHost(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if newer == first || newer.Generation <= first.Generation || newLife == life {
		t.Fatal("ABA scope reused host")
	}
	select {
	case <-life:
	default:
		t.Fatal("original lifetime not retired")
	}
	s.contexts = map[string]int64{}
	c.refreshSubmissionPreviewHost(s)
	if _, _, err = c.SubmissionPreviewHost(context.Background()); err == nil {
		t.Fatal("missing private host accepted")
	}
}
func TestSubmissionPreviewHostSubscriberCoalescesAndCloses(t *testing.T) {
	c, s := previewChromeFixture(t)
	updates, stop := c.SubmissionPreviewUpdates()
	defer stop()
	<-updates
	c.refreshSubmissionPreviewHost(s)
	s.contexts["s"] = 18
	c.refreshSubmissionPreviewHost(s)
	s.contexts["s"] = 19
	c.refreshSubmissionPreviewHost(s)
	if len(updates) != 1 {
		t.Fatalf("wakeups not coalesced: %d", len(updates))
	}
	<-updates
	c.closeSubmissionPreview()
	if _, ok := <-updates; ok {
		t.Fatal("subscriber not closed")
	}
	stop()
}
func TestSubmissionPreviewFreezeRejectsRetiredHost(t *testing.T) {
	c, s, host, selection := resolvedVisibilityChromeFixture(t)
	if _, err := c.FreezeSubmissionPreview(host, selection); err != nil {
		t.Fatal(err)
	}
	c.retireOperatorSketchScopes(s, "s")
	c.refreshSubmissionPreviewHost(s)
	if _, err := c.FreezeSubmissionPreview(host, selection); err == nil {
		t.Fatal("retired host froze board selection")
	}
	if _, err := c.FreezeSubmissionPreview(core.PreviewHost{}, selection); err == nil {
		t.Fatal("zero host froze selection")
	}
}

func TestSubmissionPreviewHostActivationABARejectsOldGeneration(t *testing.T) {
	c, s := previewChromeFixture(t)
	c.refreshSubmissionPreviewHost(s)
	first, life, _ := c.SubmissionPreviewHost(context.Background())
	s.sessions["second"] = "other-tab"
	s.contexts["second"] = 19
	s.tracker.Apply(TargetCreated{ID: "other-tab"})
	c.refreshSubmissionPreviewHost(s)
	second, _, err := c.SubmissionPreviewHost(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s.tracker.Apply(PageReport{ID: "tab", Visible: true, Focused: true})
	c.refreshSubmissionPreviewHost(s)
	again, _, err := c.SubmissionPreviewHost(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != again.ID || again.Generation <= second.Generation || again.Generation <= first.Generation {
		t.Fatalf("activation ABA reused identity: %v %v %v", first, second, again)
	}
	select {
	case <-life:
		t.Fatal("tab activation destroyed reusable context lifetime")
	default:
	}
}

type previewProjectionCaller struct {
	t     *testing.T
	ack   submissionPreviewValue
	run   func()
	calls int
}

func (c *previewProjectionCaller) Call(ctx context.Context, session, method string, params, out any) error {
	c.t.Helper()
	c.calls++
	if session != "session" || method != "Runtime.evaluate" {
		c.t.Fatalf("wrong private target: %q %q", session, method)
	}
	p := params.(map[string]any)
	if p["contextId"] != int64(17) || p["returnByValue"] != true {
		c.t.Fatalf("wrong private context: %#v", p)
	}
	if c.run != nil {
		c.run()
	}
	data, err := json.Marshal(map[string]any{"result": map[string]any{"value": c.ack}})
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}
func projectionValue() submissionPreviewValue {
	return submissionPreviewValue{Host: "host", Generation: "7", Revision: "9007199254740993", Operation: "9007199254740993", Stamp: submissionPreviewStamp{Board: "3", Delivered: "4"}, Items: []submissionPreviewItem{{RecordingID: "rec", ID: "shape", Sequence: 5, Digest: strings.Repeat("a", 64)}}, Notes: []submissionPreviewNote{{ID: "n", Revision: "2"}}, Page: &sketchimage.PageScene{RecordingID: "rec", SketchSourceFacts: query.SketchSourceFacts{T: 5, RecordIndex: 7, Doc: 1, Tab: "tab", URL: "https://example.test/", VP: recording.Viewport{W: 800, H: 600, DPR: 1}}}, Remaining: 2, Cursor: board.SketchDeliveryCursor{Epoch: "cast", Seq: 10}, Accepted: true}
}
func TestSubmissionPreviewProjectionRequiresExactAcknowledgment(t *testing.T) {
	for _, mutate := range []func(*submissionPreviewValue){nil,
		func(v *submissionPreviewValue) { v.Host = "other" }, func(v *submissionPreviewValue) { v.Generation = "8" },
		func(v *submissionPreviewValue) { v.Revision = "9007199254740992" }, func(v *submissionPreviewValue) { v.Operation = "9007199254740992" },
		func(v *submissionPreviewValue) { v.Stamp.Board = "2" }, func(v *submissionPreviewValue) { v.Stamp.Delivered = "3" },
		func(v *submissionPreviewValue) { v.Items[0].Sequence++ }, func(v *submissionPreviewValue) { v.Remaining++ },
		func(v *submissionPreviewValue) { v.Notes[0].Revision = "3" }, func(v *submissionPreviewValue) { v.Page.RecordIndex++ }, func(v *submissionPreviewValue) { v.Page = nil }, func(v *submissionPreviewValue) { v.Cursor.Seq-- }, func(v *submissionPreviewValue) { v.Accepted = false },
	} {
		requested, ack := projectionValue(), projectionValue()
		if mutate != nil {
			mutate(&ack)
		}
		fake := &previewProjectionCaller{t: t, ack: ack}
		err := deliverSubmissionPreview(context.Background(), fake, SketchSceneScope{ParentSession: "session", ParentContext: 17}, requested, func() error { return nil })
		if (err == nil) != (mutate == nil) {
			t.Fatalf("ACK classification wrong: %v", err)
		}
	}
}
func TestSubmissionPreviewProjectionLateAckCannotRestoreRetiredHost(t *testing.T) {
	valid := true
	current := func() error {
		if !valid {
			return board.ErrStale
		}
		return nil
	}
	fake := &previewProjectionCaller{t: t, ack: projectionValue(), run: func() { valid = false }}
	if err := deliverSubmissionPreview(context.Background(), fake, SketchSceneScope{ParentSession: "session", ParentContext: 17}, projectionValue(), current); err == nil {
		t.Fatal("late ACK restored retired host")
	}
	if fake.calls != 1 {
		t.Fatalf("projection duplicated IO: %d", fake.calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fake.calls = 0
	if err := deliverSubmissionPreview(ctx, fake, SketchSceneScope{}, projectionValue(), func() error { return nil }); err == nil || fake.calls != 0 {
		t.Fatal("canceled publication performed IO")
	}
}

func TestSubmissionPreviewShownHasNoPressureWakeups(t *testing.T) {
	c, s := previewChromeFixture(t)
	c.refreshSubmissionPreviewHost(s)
	host, _, err := c.SubmissionPreviewHost(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	updates, stop := c.SubmissionPreviewUpdates()
	defer stop()
	<-updates
	if !c.awaitSubmissionPreviewDelivery(host, 7) {
		t.Fatal("publication did not await receiver")
	}
	c.submissionPreviewDeliveryProgress()
	if len(updates) != 1 {
		t.Fatal("catchup did not wake pending preview")
	}
	<-updates
	// A late ACK for an older operation cannot clear a newer publication's wait.
	c.finishSubmissionPreviewDelivery(host, 6)
	c.submissionPreviewDeliveryProgress()
	if len(updates) != 1 {
		t.Fatal("stale ACK cleared current publication")
	}
	<-updates
	c.finishSubmissionPreviewDelivery(host, 7)
	for _, n := range []int{100, 200} {
		for i := 0; i < n; i++ {
			c.submissionPreviewDeliveryProgress()
			c.refreshOperator()
		}
		if len(updates) != 0 {
			t.Fatalf("%d pressure turns woke Shown preview", n)
		}
	}
}

func TestSubmissionPreviewRetirementSerializesWithFreezeCriticalSection(t *testing.T) {
	c, s := previewChromeFixture(t)
	c.refreshSubmissionPreviewHost(s)
	host, life, err := c.SubmissionPreviewHost(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, err := c.withSubmissionPreviewHost(host, func() (board.Selection, error) {
			close(entered)
			<-release
			select {
			case <-life:
				return board.Selection{}, board.ErrStale
			default:
			}
			return board.Selection{}, nil
		})
		finished <- err
	}()
	<-entered
	retired := make(chan struct{})
	go func() { c.retireOperatorSketchScopes(s, "s"); close(retired) }()
	// Observe the actual retired-scope call blocked on the guard mutex. This
	// avoids assuming a goroutine has reached the lock after a scheduler yield.
	deadline := testwait.After(t)
	for {
		select {
		case <-life:
			close(release)
			<-finished
			t.Fatal("retirement crossed the freeze critical section")
		case <-deadline:
			close(release)
			<-finished
			t.Fatal("retirement never reached the host guard")
		default:
		}
		stack := make([]byte, 64<<10)
		n := runtime.Stack(stack, true)
		blocked := false
		for _, trace := range strings.Split(string(stack[:n]), "\n\n") {
			if strings.Contains(trace, "retireSubmissionPreviewScope") && strings.Contains(trace, "lockSlow") {
				blocked = true
				break
			}
		}
		if blocked {
			break
		}
		runtime.Gosched()
	}
	close(release)
	if err := <-finished; err != nil {
		t.Fatal("freeze lost lifetime while owning guard", err)
	}
	<-retired
	select {
	case <-life:
	default:
		t.Fatal("retirement did not finish after freeze")
	}
	if _, _, err = c.SubmissionPreviewHost(context.Background()); err == nil {
		t.Fatal("retired host remained available")
	}
}
