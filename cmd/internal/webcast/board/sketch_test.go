package board

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func sketchServiceRig(t *testing.T) (*rig, *SketchService, SketchAuthority) {
	t.Helper()
	r := newRig(t).page(recording.SegLive)
	h := recording.NewHub(0)
	for _, v := range r.mem.Records() {
		if err := h.Put(v); err != nil {
			t.Fatal(err)
		}
	}
	r.w.sink = recording.Tee(r.mem, r.bd, h)
	s := r.bd.NewSketchService(r.w.WriteSketch, r.w.ObserveSketch, h.CandidateSketchSource)
	a, err := r.bd.AdmitAnnotation()
	if err != nil {
		t.Fatal(err)
	}
	return r, s, SketchAuthority{Author: recording.Author{ID: recording.OperatorID}, Owner: recording.OperatorID, Epoch: 1, Annotation: a, Check: func() error { return nil }}
}

func TestSketchServiceFrozenAnchorAndExactRetry(t *testing.T) {
	r, s, a := sketchServiceRig(t)
	c := SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0}
	v := s.Execute(context.Background(), a, c)
	if v.Outcome != SketchAccepted || v.Source == nil {
		t.Fatalf("begin: %+v", v)
	}
	first := len(r.mem.Records())
	if again := s.Execute(context.Background(), a, c); again.Sketch != v.Sketch || again.Outcome != SketchAccepted || len(r.mem.Records()) != first {
		t.Fatalf("duplicate: %+v", again)
	}
	r.b.At(10)
	r.add(&recording.Text{ID: r.p1.C[0].ID, V: "changed live"})
	c = SketchCommand{Kind: "sketch-stroke-begin", Operation: "stroke", Sequence: 1, Epoch: 1, Sketch: v.Sketch, ID: "s", Node: r.p1.ID, Box: recording.Box{1, 2, 3, 4}}
	if got := s.Execute(context.Background(), a, c); got.Outcome != SketchAccepted {
		t.Fatalf("stroke: %+v", got)
	}
	stroke := r.kinds("sketch-stroke-begin")[0].(*recording.SketchStrokeBegin)
	if stroke.At.Text != "First" {
		t.Fatalf("anchored against changed live source: %+v", stroke.At)
	}
	c.Box[0]++
	if got := s.Execute(context.Background(), a, c); got.Outcome != SketchRefused {
		t.Fatal("conflicting retry accepted")
	}
}

func TestSketchServiceOwnerAndEpochFence(t *testing.T) {
	_, s, a := sketchServiceRig(t)
	c := SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0}
	v := s.Execute(context.Background(), a, c)
	if v.Outcome != SketchAccepted {
		t.Fatalf("%+v", v)
	}
	other := a
	other.Owner = strings.Repeat("a", 64)
	status := SketchCommand{Kind: "sketch-status", Operation: "begin", Epoch: 1, Sketch: v.Sketch}
	if got := s.Status(context.Background(), other, status); got.Outcome != SketchRefused {
		t.Fatal("other owner saw receipt")
	}
	changed := a
	changed.Epoch = 3
	c = SketchCommand{Kind: "sketch-label", Operation: "label", Sequence: 1, Epoch: 3, Sketch: v.Sketch, Label: "late"}
	if got := s.Execute(context.Background(), changed, c); got.Outcome != SketchRefused {
		t.Fatal("new epoch rewrote old sketch origin")
	}
	// Original owner can label after driving authority is gone.
	a.Check = func() error { return ErrDisabled }
	c.Epoch = 1
	if got := s.Execute(context.Background(), a, c); got.Outcome != SketchAccepted {
		t.Fatalf("old owner label: %+v", got)
	}
}

func TestSketchLeaseExpiresOnlyAfterAcceptedMutationIdle(t *testing.T) {
	r, s, a := sketchServiceRig(t)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	v := s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0})
	if v.Outcome != SketchAccepted {
		t.Fatalf("%+v", v)
	}
	now = now.Add(time.Minute)
	if got := s.Status(context.Background(), a, SketchCommand{Kind: "sketch-status", Operation: "begin", Epoch: 1, Sketch: v.Sketch}); got.Outcome != SketchAccepted {
		t.Fatalf("%+v", got)
	}
	now = now.Add(time.Minute)
	if err := s.Expire(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	interrupts := r.kinds("sketch-interrupt")
	if len(interrupts) != 1 || interrupts[0].(*recording.SketchInterrupt).Reason != recording.SketchInterruptExpiry {
		t.Fatalf("expiry: %v", interrupts)
	}
	if _, err := s.ReadSource(a, v.Sketch, v.Source.Handle, 0, 1); err == nil {
		t.Fatal("expired source survived")
	}
}

func TestSketchLeaseRetiresWithCoreLifecycle(t *testing.T) {
	for _, why := range []string{"label", "navigation", "clear", "close"} {
		t.Run(why, func(t *testing.T) {
			r, s, a := sketchServiceRig(t)
			v := s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0})
			if v.Outcome != SketchAccepted {
				t.Fatalf("%+v", v)
			}
			switch why {
			case "label":
				s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-label", Operation: "label", Sequence: 1, Epoch: 1, Sketch: v.Sketch, Label: "done"})
			case "navigation":
				r.b.At(3).Page(recording.SegNavigate, "http://next/")
				r.flush()
			case "clear":
				if err := r.w.ClearAnnotations(context.Background(), 2); err != nil {
					t.Fatal(err)
				}
			case "close":
				r.bd.Close()
			}
			if s.source != nil || s.handle != "" {
				t.Fatal("core lifecycle retained held source")
			}
		})
	}
}

func TestSketchLeaseRefusesChunkAtDeadlineBeforeExpiryPoll(t *testing.T) {
	r, s, a := sketchServiceRig(t)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	v := s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0})
	if v.Outcome != SketchAccepted {
		t.Fatalf("%+v", v)
	}
	now = now.Add(2 * time.Minute)
	got := s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-stroke-begin", Operation: "stroke", Sequence: 1, Epoch: 1, Sketch: v.Sketch, ID: "s", Node: r.p1.ID, Box: recording.Box{1, 2, 3, 4}})
	if got.Outcome != SketchRefused {
		t.Fatalf("expired source admitted before poll: %+v", got)
	}
}

func TestSketchServiceRetainsLastDefiniteRefusal(t *testing.T) {
	r, s, a := sketchServiceRig(t)
	v := s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0})
	c := SketchCommand{Kind: "sketch-stroke-begin", Operation: "bad", Sequence: 1, Epoch: 1, Sketch: v.Sketch, ID: "s", Node: 999999, Box: recording.Box{1, 2, 3, 4}}
	refused := s.Execute(context.Background(), a, c)
	if refused.Outcome != SketchRefused {
		t.Fatalf("%+v", refused)
	}
	got := s.Status(context.Background(), a, SketchCommand{Kind: "sketch-status", Operation: c.Operation, Sequence: c.Sequence, Epoch: c.Epoch, Sketch: c.Sketch})
	if got.Outcome != SketchRefused {
		t.Fatalf("forgot refusal: %+v", got)
	}
	c.Node = r.p1.ID
	if got := s.Execute(context.Background(), a, c); got.Outcome != SketchRefused || got.Code != "conflict" {
		t.Fatalf("refused identity changed: %+v", got)
	}
}

func TestSketchLeaseRenewsAtWriterAcceptanceBeforeReply(t *testing.T) {
	r, s, a := sketchServiceRig(t)
	var clock atomic.Int64
	clock.Store(1000)
	s.now = func() time.Time { return time.Unix(clock.Load(), 0) }
	v := s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0})
	if v.Outcome != SketchAccepted {
		t.Fatalf("%+v", v)
	}
	write := s.write
	accepted := make(chan struct{})
	release := make(chan struct{})
	done := make(chan SketchReply, 1)
	defer func() {
		close(release)
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("writer was not joined")
		}
	}()
	s.write = func(ctx context.Context, in SketchWrite) (SketchWriteResult, error) {
		out, err := write(ctx, in)
		if in.Record.Kind() == "sketch-stroke-begin" {
			close(accepted)
			<-release
		}
		return out, err
	}
	clock.Store(1119)
	go func() {
		done <- s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-stroke-begin", Operation: "stroke", Sequence: 1, Epoch: 1, Sketch: v.Sketch, ID: "s", Node: r.p1.ID, Box: recording.Box{1, 2, 3, 4}})
	}()
	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("writer did not accept")
	}
	clock.Store(1120)
	if err := s.Expire(context.Background(), s.now()); err != nil {
		t.Fatal(err)
	}
	if len(r.kinds("sketch-interrupt")) != 0 {
		t.Fatal("held reply hid accepted mutation renewal")
	}
}

func TestSketchLeaseAssetReadsUseOnlyFrozenBindings(t *testing.T) {
	r, s, a := sketchServiceRig(t)
	hash := strings.Repeat("b", 64)
	r.add(&recording.Asset{URL: "http://app/image.png", Hash: hash, MIME: "image/png"})
	v := s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0})
	if v.Outcome != SketchAccepted {
		t.Fatalf("%+v", v)
	}
	if err := s.AdmitSourceAsset(a, v.Sketch, v.Source.Handle, hash); err != nil {
		t.Fatal(err)
	}
	if err := s.AdmitSourceAsset(a, v.Sketch, v.Source.Handle, strings.Repeat("c", 64)); err == nil {
		t.Fatal("unbound asset exposed")
	}
	if n, err := s.SourceLength(a, v.Sketch, v.Source.Handle); err != nil || n != v.Source.Length {
		t.Fatalf("length %d %v", n, err)
	}
}

func TestSketchLeaseWorkerExpiresAndJoins(t *testing.T) {
	r, s, a := sketchServiceRig(t)
	var clock atomic.Int64
	clock.Store(1000)
	s.now = func() time.Time { return time.Unix(clock.Load(), 0) }
	v := s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0})
	if v.Outcome != SketchAccepted {
		t.Fatalf("%+v", v)
	}
	clock.Store(1120)
	wake, cancelWake := r.bd.SubscribeSketchDelivery()
	defer cancelWake()
	select {
	case <-wake:
	default:
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	select {
	case <-wake:
	case <-time.After(time.Second):
		t.Fatal("lease worker did not expire")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("lease worker did not join")
	}
	if len(r.kinds("sketch-interrupt")) != 1 {
		t.Fatal("worker did not record expiry")
	}
}

func TestSketchHandoffNewBeginDoesNotWaitForOldAcceptedReply(t *testing.T) {
	r, s, a := sketchServiceRig(t)
	write := s.write
	accepted := make(chan struct{})
	release := make(chan struct{})
	done := make(chan SketchReply, 1)
	defer func() {
		close(release)
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("old begin did not join")
		}
	}()
	s.write = func(ctx context.Context, in SketchWrite) (SketchWriteResult, error) {
		out, err := write(ctx, in)
		if begin, ok := in.Record.(*recording.SketchBegin); ok && begin.Author.ID == recording.OperatorID {
			close(accepted)
			<-release
		}
		return out, err
	}
	go func() {
		done <- s.Execute(context.Background(), a, SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0})
	}()
	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("begin not accepted")
	}
	who := recording.Author{ID: "v1", Name: "Ana"}
	control, err := recording.NextControl(recording.Control{}, recording.CtlRequest, who)
	if err != nil {
		t.Fatal(err)
	}
	r.add(&recording.Ctl{Op: recording.CtlRequest, Who: who, Control: control})
	control, err = recording.NextControl(control, recording.CtlGrant, who)
	if err != nil {
		t.Fatal(err)
	}
	r.add(&recording.Ctl{Op: recording.CtlGrant, Who: who, Control: control})
	next := a
	next.Author = who
	next.Owner = strings.Repeat("a", 64)
	next.Epoch = 2
	got := s.Execute(context.Background(), next, SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 2, Seg: 1, Color: 1})
	if got.Outcome != SketchAccepted {
		t.Fatalf("new drawer blocked by old HTTP reply: %+v", got)
	}
	if s.sourceSketch != got.Sketch {
		t.Fatal("new source not selected")
	}
}
