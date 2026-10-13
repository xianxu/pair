package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// controlPage supplies one actual recorder document. The queue, reducer,
// production Chrome adapter, and control writer are the production ones.
type controlPage struct{ seq atomic.Uint64 }

func (p *controlPage) CallSeq(_ context.Context, _, method string, _ any, result any) (uint64, error) {
	reply := `{}`
	switch method {
	case "DOM.getDocument":
		reply = `{"root":{"nodeId":1,"backendNodeId":1,"nodeType":9,"nodeName":"#document","documentURL":"http://app.local/","children":[{"nodeId":2,"backendNodeId":2,"nodeType":1,"nodeName":"HTML","localName":"html"}]}}`
	case "Page.getFrameTree":
		reply = `{"frameTree":{"frame":{"id":"T","loaderId":"L"}}}`
	}
	var err error
	if result != nil {
		err = json.Unmarshal([]byte(reply), result)
	}
	return p.seq.Add(1), err
}

type controlAckSink struct {
	recording.MemSink
	op               string
	entered, release chan struct{}
	failure          error
}

func (s *controlAckSink) Put(r recording.Record) error {
	if ctl, ok := r.(*recording.Ctl); ok && ctl.Op == s.op {
		close(s.entered)
		<-s.release
		if err := s.MemSink.Put(r); err != nil {
			return err
		}
		return s.failure
	}
	return s.MemSink.Put(r)
}
func newControlAckChrome(t *testing.T, sink recording.Sink) *Chrome {
	t.Helper()
	r := recorder.Start(recorder.Options{ID: "ack", Call: &controlPage{}, ShotEvery: -1, Out: func() (recorder.Opened, error) { return recorder.Opened{Sink: sink}, nil }})
	t.Cleanup(func() { r.Close() })
	r.Send(recorder.Live{Tab: recording.Tab{ID: "T"}, Session: "S"})
	if _, err := r.Stats(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.Opened() {
		t.Fatal("recorder did not open")
	}
	return &Chrome{rec: r}
}
func TestProductionControlWaitsForRecorderOutcome(t *testing.T) {
	for _, op := range []string{recording.CtlRequest, recording.CtlGrant} {
		for _, fails := range []bool{false, true} {
			t.Run(op+map[bool]string{false: "/success", true: "/uncertain"}[fails], func(t *testing.T) {
				sink := &controlAckSink{op: op, entered: make(chan struct{}), release: make(chan struct{})}
				if fails {
					sink.failure = errors.New("sink accepted control then failed")
				}
				chrome := newControlAckChrome(t, sink)
				ctrl := drive.NewMemController(liveGate{chrome}, nil, func() int64 { return 0 }, time.Hour)
				defer ctrl.Close()
				ctrl.SetDriveAllowed(context.Background(), true)
				seat, _ := ctrl.Join("A")
				if op == recording.CtlGrant {
					if err := ctrl.Request(seat.Key); err != nil {
						t.Fatal(err)
					}
				}
				done := make(chan error, 1)
				go func() {
					if op == recording.CtlRequest {
						done <- ctrl.Request(seat.Key)
					} else {
						done <- ctrl.Grant(seat.Viewer.ID)
					}
				}()
				<-sink.entered
				select {
				case err := <-done:
					close(sink.release)
					t.Fatalf("reported control outcome before recorder commit: %v", err)
				default:
				}
				if op == recording.CtlRequest && len(ctrl.Control().Pending) != 0 {
					t.Fatal("published uncommitted request")
				}
				if op == recording.CtlGrant && ctrl.Control().Driving() != "" {
					t.Fatal("published uncommitted grant")
				}
				close(sink.release)
				err := <-done
				if fails {
					if err == nil || ctrl.DriveCapability().Outcome != capability.Broken {
						t.Fatalf("unknown sink outcome: %v %+v", err, ctrl.DriveCapability())
					}
				} else if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestProductionControlRefusesClosedRecorder(t *testing.T) {
	chrome := newControlAckChrome(t, &recording.MemSink{})
	chrome.rec.Close()
	ctrl := drive.NewMemController(liveGate{chrome}, nil, func() int64 { return 0 }, time.Hour)
	defer ctrl.Close()
	ctrl.SetDriveAllowed(context.Background(), true)
	seat, _ := ctrl.Join("A")
	if err := ctrl.Request(seat.Key); err == nil {
		t.Fatal("closed recorder acknowledged request")
	}
	if len(ctrl.Control().Pending) != 0 || ctrl.DriveCapability().Outcome != capability.Broken {
		t.Fatal(ctrl.Control(), ctrl.DriveCapability())
	}
}

func TestProductionControlClosedBeforeFirstLiveIsTerminal(t *testing.T) {
	r := recorder.Start(recorder.Options{ShotEvery: -1})
	r.Close()
	chrome := &Chrome{rec: r}
	ctrl := drive.NewMemController(liveGate{chrome}, nil, func() int64 { return 0 }, time.Hour)
	defer ctrl.Close()
	ctrl.SetDriveAllowed(context.Background(), true)
	s, _ := ctrl.Join("A")
	if err := ctrl.Request(s.Key); !errors.Is(err, drive.ErrClosed) {
		t.Fatalf("closed recorder outcome: %v", err)
	}
	if ctrl.DriveCapability().Outcome != capability.Broken {
		t.Fatal(ctrl.DriveCapability())
	}
}
func ctlRequest(id string, prior ...recording.Author) *recording.Ctl {
	who := recording.Author{ID: id}
	return &recording.Ctl{Op: recording.CtlRequest, Who: who, Control: recording.Control{Pending: append(prior, who)}}
}
func TestProductionControlCancellationRetainsStartedOutcome(t *testing.T) {
	sink := &controlAckSink{op: recording.CtlRequest, entered: make(chan struct{}), release: make(chan struct{}), failure: errors.New("late uncertain control write")}
	chrome := newControlAckChrome(t, sink)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- chrome.WriteControl(ctx, ctlRequest("A")) }()
	<-sink.entered
	cancel()
	select {
	case err := <-done:
		close(sink.release)
		t.Fatalf("lost started write outcome: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(sink.release)
	if err := <-done; !errors.Is(err, sink.failure) || capability.FailureOutcome(err) != capability.Broken {
		t.Fatalf("late write: %v", err)
	}
}
func TestProductionControlQueuedCancellationIsUnsubmitted(t *testing.T) {
	sink := &controlAckSink{op: recording.CtlRequest, entered: make(chan struct{}), release: make(chan struct{})}
	chrome := newControlAckChrome(t, sink)
	done := make(chan error, 1)
	go func() { done <- chrome.WriteControl(context.Background(), ctlRequest("A")) }()
	<-sink.entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := chrome.WriteControl(ctx, ctlRequest("B", recording.Author{ID: "A"}))
	if !errors.Is(err, context.Canceled) || capability.FailureOutcome(err) != capability.Retryable {
		close(sink.release)
		t.Fatalf("queued cancellation: %v", err)
	}
	close(sink.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := chrome.rec.Stats(context.Background()); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, r := range sink.Records() {
		if c, ok := r.(*recording.Ctl); ok {
			count++
			if c.Who.ID != "A" {
				t.Fatalf("canceled request reached sink: %+v", c)
			}
		}
	}
	if count != 1 {
		t.Fatalf("control commits: %d", count)
	}
}

func TestProductionControllerCloseJoinsStartedControlWrite(t *testing.T) {
	sink := &controlAckSink{op: recording.CtlRequest, entered: make(chan struct{}), release: make(chan struct{}), failure: errors.New("late control write failure")}
	chrome := newControlAckChrome(t, sink)
	ctrl := drive.NewMemController(liveGate{chrome}, nil, func() int64 { return 0 }, time.Hour)
	ctrl.SetDriveAllowed(context.Background(), true)
	seat, _ := ctrl.Join("A")
	requested := make(chan error, 1)
	go func() { requested <- ctrl.Request(seat.Key) }()
	<-sink.entered
	closed := make(chan struct{})
	go func() { ctrl.Close(); close(closed) }()
	select {
	case <-closed:
		close(sink.release)
		t.Fatal("Close abandoned started control write")
	case <-time.After(20 * time.Millisecond):
	}
	close(sink.release)
	if err := <-requested; !errors.Is(err, sink.failure) {
		t.Fatalf("lost late write outcome: %v", err)
	}
	<-closed
	if ctrl.DriveCapability().Phase != capability.Ended {
		t.Fatal(ctrl.DriveCapability())
	}
}
