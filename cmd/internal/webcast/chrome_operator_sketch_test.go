package webcast

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOperatorSketchEnvelopeClosed(t *testing.T) {
	good := `{"request":"r","kind":"sketch-status","generation":"1","epoch":1,"body":{"operation":"o","sequence":0,"epoch":1}}`
	if _, err := parseOperatorSketchEnvelope([]byte(good)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		`{"request":"r","request":"x","kind":"scene-open","generation":"1","epoch":1,"body":{}}`,
		`{"request":"r","kind":"scene-open","generation":"1","epoch":1,"body":{},"session":"forged"}`,
		`{"request":"\ud800","kind":"scene-open","generation":"1","epoch":1,"body":{}}`,
		`{"request":"r","kind":"scene-open","generation":"01","epoch":1,"body":{}}`,
		`{"request":"r","kind":"scene-open","generation":"1","epoch":9007199254740992,"body":{}}`,
		`{"request":"r","kind":"scene-open","generation":"1","epoch":1,"body":null}`,
	} {
		if _, err := parseOperatorSketchEnvelope([]byte(bad)); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}
func TestOperatorSketchQueueBoundAndExpiredScope(t *testing.T) {
	calls := 0
	r := newOperatorSketchRuntime(operatorSketchRuntimeOptions{Authority: func(uint64, uint64) (board.SketchAuthority, error) { calls++; return board.SketchAuthority{}, nil }})
	life := make(chan struct{})
	close(life)
	env := operatorSketchEnvelope{Request: "r", Kind: "scene-open", Generation: "1", Epoch: 1, Body: json.RawMessage(`{}`)}
	request := operatorSketchRequest{Scope: SketchSceneScope{ParentSession: "top", ParentContext: 1, Loader: "loader"}, Lifetime: life, Envelope: env}
	for i := 0; i < 64; i++ {
		if !r.Enqueue(request) {
			t.Fatal("early refusal")
		}
	}
	if r.Enqueue(request) {
		t.Fatal("queue exceeds64")
	}
	for i := 0; i < 64; i++ {
		r.Turn(context.Background())
	}
	if calls != 0 {
		t.Fatal("expired context reached authority")
	}
	r.Close(context.Background())
}

type operatorSketchFake struct {
	calls    int
	wrong    bool
	raw      []byte
	epoch    string
	total    int
	maxChunk int
}

func (f *operatorSketchFake) Call(_ context.Context, _ string, method string, params, result any) error {
	f.calls++
	p := params.(map[string]any)
	expression, _ := p["expression"].(string)
	var value any = true
	if i := strings.Index(expression, "__couchcastSketchDelivery?.("); i >= 0 {
		var payload struct {
			Phase, Handle, Epoch, Base64 string
			Total, Offset                int
		}
		if err := json.Unmarshal([]byte(expression[i+len("__couchcastSketchDelivery?.("):len(expression)-1]), &payload); err != nil {
			return err
		}
		if payload.Phase == "begin" {
			f.raw = nil
			f.total = payload.Total
			f.epoch = payload.Epoch
			value = map[string]any{"offset": 0}
		} else {
			part, err := base64.StdEncoding.DecodeString(payload.Base64)
			if err != nil {
				return err
			}
			if len(part) > f.maxChunk {
				f.maxChunk = len(part)
			}
			if payload.Offset == len(f.raw) {
				f.raw = append(f.raw, part...)
			}
			ack := map[string]any{"offset": payload.Offset + len(part)}
			if len(f.raw) == f.total {
				var delivery board.SketchDelivery
				if err = json.Unmarshal(f.raw, &delivery); err != nil {
					return err
				}
				var seq uint64
				if delivery.Bootstrap != nil {
					seq = delivery.Bootstrap.Seq
				} else {
					seq = delivery.Batch.To
				}
				if f.wrong {
					seq++
				}
				ack["epoch"] = f.epoch
				ack["seq"] = seq
			}
			value = ack
		}
	}
	if result != nil {
		raw, _ := json.Marshal(map[string]any{"result": map[string]any{"value": value}})
		return json.Unmarshal(raw, result)
	}
	return nil
}
func TestOperatorSketchJournalRequiresExactFinalACK(t *testing.T) {
	b := board.New(notes.Structural{}, nil)
	fake := &operatorSketchFake{wrong: true}
	r := newOperatorSketchRuntime(operatorSketchRuntimeOptions{Caller: fake, Board: b})
	life := make(chan struct{})
	scope := SketchSceneScope{ParentSession: "top", ParentContext: 1, Loader: "l"}
	r.Reconcile([]operatorSketchTarget{{Scope: scope, Lifetime: life, Active: true}})
	state := r.states[operatorContextID{"top", 1}]
	if !r.Turn(context.Background()) || fake.calls != 1 {
		t.Fatal("begin must be one turn")
	}
	r.Turn(context.Background())
	if fake.calls != 2 {
		t.Fatal("chunk must be one turn")
	}
	if state.cursor.Epoch != "" || state.delivery == nil {
		t.Fatal("wrong final ACK advanced cursor")
	}
	// An uncertain final response repeats the same bounded chunk, not a new
	// transfer, and advances only after the exact high-water acknowledgement.
	fake.wrong = false
	state.retryAt = time.Time{}
	r.Turn(context.Background())
	if state.cursor != b.OperatorSketchCursor() || state.delivery != nil {
		t.Fatal("exact replay ACK not adopted")
	}
	if fake.maxChunk > board.MaxSketchTransfer {
		t.Fatal("chunk exceeds32KiB")
	}
	calls := fake.calls
	close(life)
	r.Turn(context.Background())
	if fake.calls != calls {
		t.Fatal("retired parent received delivery")
	}
	if len(r.states) != 0 || len(r.order) != 0 {
		t.Fatal("retired context retained")
	}
	r.Close(context.Background())
}
func TestOperatorSketchEnvelopeRequiredFieldsAndGeneration64(t *testing.T) {
	good := `{"request":"r","kind":"scene-open","generation":"18446744073709551615","epoch":0,"body":{}}`
	if _, err := parseOperatorSketchEnvelope([]byte(good)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{"request":"r","kind":"scene-open","generation":"1","body":{}}`, `{"request":"r","kind":"scene-open","generation":"1","epoch":0,"body":null}`} {
		if _, err := parseOperatorSketchEnvelope([]byte(bad)); err == nil {
			t.Fatal("missing owned field accepted")
		}
	}
}

func TestOperatorSketchSceneBodiesAreClosedAndComplete(t *testing.T) {
	for _, test := range []struct{ kind, body string }{
		{"scene-open", `[]`}, {"scene-open", `{"expression":"window.frames[0]"}`},
		{"scene-hit", `{}`}, {"scene-hit", `{"point":[null,2]}`}, {"scene-hit", `{"point":[1]}`}, {"scene-hit", `{"point":[1,2,3]}`},
		{"scene-placement", `{"path":null}`}, {"scene-placement", `{"path":"x"}`},
		{"scene-prepare", `{"sketch":"s","source":null}`}, {"scene-prepare", `{"sketch":"s","source":{"handle":"h","time":null,"length":1,"doc":1,"tab":"t","vp":{"w":1,"h":1,"dpr":1}}}`}, {"scene-prepare", `{"sketch":"s","source":{"handle":"h","time":1,"length":1,"doc":1,"tab":"t","vp":{"w":1,"h":1,"dpr":1},"future":true}}`},
	} {
		raw, _ := json.Marshal(operatorSketchEnvelope{Request: "r", Kind: test.kind, Generation: "1", Epoch: 1, Body: json.RawMessage(test.body)})
		if _, err := parseOperatorSketchEnvelope(raw); err == nil {
			t.Errorf("accepted %s %s", test.kind, test.body)
		}
	}
}

func TestOperatorSketchFourContextsActiveFirstAndImmediateFrameRetirement(t *testing.T) {
	r := newOperatorSketchRuntime(operatorSketchRuntimeOptions{})
	life := make(chan struct{})
	targets := []operatorSketchTarget{}
	for i := int64(1); i <= 5; i++ {
		targets = append(targets, operatorSketchTarget{Scope: SketchSceneScope{ParentSession: fmt.Sprint(i), ParentContext: i, Loader: "l"}, Lifetime: life, Active: i == 5})
	}
	r.Reconcile(targets)
	if len(r.states) > 4 || r.states[operatorContextID{"5", 5}] == nil {
		t.Fatal("active tab not admitted before dormant tabs")
	}
	scene := NewSketchSceneSession(nil, targets[4].Scope, func(SketchSceneScope) bool { return true }, nil)
	r.frames.Store("private-frame", scene)
	r.RetireFrame("private-frame")
	if !errors.Is(scene.check(), ErrSketchSceneRetired) {
		t.Fatal("navigation did not immediately fence scene")
	}
	r.Close(context.Background())
}

func TestOperatorSketchAsyncWriterDoesNotBlockWorkerAndLifetimeFences(t *testing.T) {
	b := board.New(notes.Structural{}, nil)
	if err := b.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	build := recording.NewBuilder("runtime")
	build.Page(recording.SegLive, "https://app/")
	for _, record := range build.Recs {
		if err := b.Put(record); err != nil {
			t.Fatal(err)
		}
	}
	admission, err := b.AdmitAnnotation()
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	checked := make(chan error, 1)
	service := b.NewSketchService(func(_ context.Context, write board.SketchWrite) (board.SketchWriteResult, error) {
		close(started)
		<-release
		err := write.Check()
		checked <- err
		return board.SketchWriteResult{Outcome: board.SketchRefused}, nil
	}, nil, nil)
	life := make(chan struct{})
	scope := SketchSceneScope{ParentSession: "top", ParentContext: 1, Loader: "l"}
	fake := &operatorSketchFake{}
	r := newOperatorSketchRuntime(operatorSketchRuntimeOptions{Caller: fake, Service: service, Authority: func(uint64, uint64) (board.SketchAuthority, error) {
		return board.SketchAuthority{Author: recording.Author{ID: recording.OperatorID}, Owner: recording.OperatorID, Epoch: 1, Annotation: admission, Check: func() error { return nil }}, nil
	}})
	r.Reconcile([]operatorSketchTarget{{Scope: scope, Lifetime: life, Active: true}})
	state := r.states[operatorContextID{"top", 1}]
	state.scene = NewSketchSceneSession(fake, scope, func(SketchSceneScope) bool { return true }, nil)
	envelope := operatorSketchEnvelope{Request: "begin", Kind: "sketch-begin", Generation: "1", Epoch: 1, Body: json.RawMessage(`{"operation":"begin","sequence":0,"epoch":1,"seg":1,"color":0}`)}
	if !r.Enqueue(operatorSketchRequest{Scope: scope, Lifetime: life, Envelope: envelope}) {
		t.Fatal("enqueue")
	}
	r.Turn(context.Background())
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("writer not started")
	}
	// A blocked writer never owns the CDP worker. A queued reply still gets a
	// bounded worker turn, and a destroyed context fences the eventual commit.
	state.replies = append(state.replies, operatorSketchReply{Request: "other", OK: true})
	if !r.Turn(context.Background()) || fake.calls != 1 {
		t.Fatal("writer blocked unrelated delivery")
	}
	state.scene.Retire()
	if err := state.begin["1:begin"].Check(); !errors.Is(err, ErrSketchSceneRetired) {
		close(release)
		t.Fatalf("pending begin lost original scene fence: %v", err)
	}
	close(release)
	select {
	case err := <-checked:
		if !errors.Is(err, ErrSketchSceneRetired) {
			t.Fatalf("retired private child commit check: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("writer check blocked")
	}
	r.Close(context.Background())
	r.work.Wait()
}

func TestOperatorSketchLifetimeGuardCannotGrantMissingDrawingAuthority(t *testing.T) {
	b := board.New(notes.Structural{}, nil)
	if err := b.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	build := recording.NewBuilder("runtime-deny")
	build.Page(recording.SegLive, "https://app/")
	for _, record := range build.Recs {
		if err := b.Put(record); err != nil {
			t.Fatal(err)
		}
	}
	admission, err := b.AdmitAnnotation()
	if err != nil {
		t.Fatal(err)
	}
	writes := 0
	service := b.NewSketchService(func(context.Context, board.SketchWrite) (board.SketchWriteResult, error) {
		writes++
		return board.SketchWriteResult{Outcome: board.SketchRefused}, nil
	}, nil, nil)
	r := newOperatorSketchRuntime(operatorSketchRuntimeOptions{Service: service, Authority: func(uint64, uint64) (board.SketchAuthority, error) {
		return board.SketchAuthority{Author: recording.Author{ID: recording.OperatorID}, Owner: recording.OperatorID, Epoch: 1, Annotation: admission}, nil
	}})
	life := make(chan struct{})
	scope := SketchSceneScope{ParentSession: "top", ParentContext: 1, Loader: "l"}
	state := r.context(operatorSketchRequest{Scope: scope, Lifetime: life})
	state.scene = NewSketchSceneSession(nil, scope, func(SketchSceneScope) bool { return true }, nil)
	r.command(state, operatorSketchEnvelope{Request: "begin", Kind: "sketch-begin", Generation: "1", Epoch: 1, Body: json.RawMessage(`{"operation":"begin","sequence":0,"epoch":1,"seg":1,"color":0}`)})
	r.work.Wait()
	r.finishCompletion(<-r.completions)
	if len(state.begin) != 0 {
		t.Fatal("definite refused begin retained origin slot")
	}
	if writes != 0 {
		t.Fatal("lifetime wrapper granted missing drawing check")
	}
	r.Close(context.Background())
}

func TestOperatorSketchSceneOpenRequiresDrawingAuthorityBeforeCDP(t *testing.T) {
	fake := &operatorSketchFake{}
	b := board.New(notes.Structural{}, nil)
	if err := b.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	admission, err := b.AdmitAnnotation()
	if err != nil {
		t.Fatal(err)
	}
	checks := 0
	r := newOperatorSketchRuntime(operatorSketchRuntimeOptions{Caller: fake, Authority: func(uint64, uint64) (board.SketchAuthority, error) {
		return board.SketchAuthority{Annotation: admission, Check: func() error { checks++; return board.ErrDisabled }}, nil
	}})
	life := make(chan struct{})
	scope := SketchSceneScope{ParentSession: "top", ParentContext: 1, Loader: "l"}
	r.Reconcile([]operatorSketchTarget{{Scope: scope, Lifetime: life, Active: true}})
	r.handle(context.Background(), operatorSketchRequest{Scope: scope, Lifetime: life, Envelope: operatorSketchEnvelope{Request: "open", Kind: "scene-open", Generation: "1", Epoch: 1, Body: json.RawMessage(`{}`)}})
	if fake.calls != 0 || checks != 1 {
		t.Fatal("unavailable drawing authority reached CDP preflight")
	}
	r.Close(context.Background())
}

func FuzzOperatorSketchEnvelope(f *testing.F) {
	for _, seed := range []string{
		`{"request":"r","kind":"scene-open","generation":"1","epoch":1,"body":{}}`,
		`{"request":"r","kind":"scene-hit","generation":"1","epoch":1,"body":{"point":[1,2]}}`,
		`{"request":"r","kind":"sketch-status","generation":"1","epoch":1,"body":{"operation":"op","sequence":0,"epoch":1}}`,
		`{"request":"\ud800","kind":"scene-open","generation":"1","epoch":1,"body":{}}`,
		`{"request":"r","kind":"scene-open","generation":"1","epoch":1,"body":{"future":{"x":1,"x":2}}}`,
		`{"request":"r","kind":"scene-hit","generation":"1","epoch":1,"body":{"point":[null,2]}}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		envelope, err := parseOperatorSketchEnvelope(raw)
		if err != nil {
			return
		}
		encoded, err := json.Marshal(envelope)
		if err != nil {
			t.Fatal(err)
		}
		again, err := parseOperatorSketchEnvelope(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(envelope, again) {
			t.Fatal("accepted envelope changed on strict roundtrip")
		}
	})
}

func TestOperatorSketchDisposeAfterAuthorityLossAndHitBeforeSource(t *testing.T) {
	fake := &operatorSketchFake{}
	r := newOperatorSketchRuntime(operatorSketchRuntimeOptions{Caller: fake, Authority: func(uint64, uint64) (board.SketchAuthority, error) { return board.SketchAuthority{}, board.ErrDisabled }})
	life := make(chan struct{})
	scope := SketchSceneScope{ParentSession: "top", ParentContext: 1, Loader: "l"}
	state := r.context(operatorSketchRequest{Scope: scope, Lifetime: life})
	state.scene = NewSketchSceneSession(fake, scope, func(SketchSceneScope) bool { return true }, nil)
	r.handle(context.Background(), operatorSketchRequest{Scope: scope, Lifetime: life, Envelope: operatorSketchEnvelope{Request: "dispose", Kind: "scene-dispose", Generation: "1", Epoch: 1, Body: json.RawMessage(`{}`)}})
	if state.scene != nil {
		t.Fatal("authority loss prevented private cleanup")
	}
	// A mere shell cannot answer geometry; there must be a live service-backed
	// prepared source, even if the child claims to have a renderer.
	r.opts.Authority = func(uint64, uint64) (board.SketchAuthority, error) { return board.SketchAuthority{}, nil }
	state.scene = NewSketchSceneSession(fake, scope, func(SketchSceneScope) bool { return true }, nil)
	state.scene.world = 88
	r.handle(context.Background(), operatorSketchRequest{Scope: scope, Lifetime: life, Envelope: operatorSketchEnvelope{Request: "hit", Kind: "scene-hit", Generation: "1", Epoch: 1, Body: json.RawMessage(`{"point":[1,2]}`)}})
	if fake.calls != 0 {
		t.Fatal("unprepared scene answered geometry")
	}
	r.Close(context.Background())
}
