package drive

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"sync"
	"testing"
	"time"
)

func TestDriveDefaultOffAndOldAdmission(t *testing.T) {
	c := NewMemController(&recording.MemSink{}, nil, func() int64 { return 1 }, time.Hour)
	defer c.Close()
	c.WithControlReset(func(context.Context, uint64) error { return nil })
	s, _ := c.Join("A")
	if err := c.Request(s.Key); err == nil {
		t.Fatal("default off admitted request")
	}
	if err := c.SetDriveAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	a, err := c.AdmitDrive()
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Request(s.Key); err != nil {
		t.Fatal(err)
	}
	if err = c.SetDriveAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if len(c.Control().Pending) != 0 {
		t.Fatal("pending survived reset")
	}
	if err = c.SetDriveAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if err = a.Request(s.Key); err == nil {
		t.Fatal("old permit resurrected")
	}
}

func TestDriveCleanupExplicitRetry(t *testing.T) {
	c := NewMemController(&recording.MemSink{}, nil, func() int64 { return 1 }, time.Hour)
	defer c.Close()
	calls := 0
	var generation uint64
	c.WithControlReset(func(_ context.Context, g uint64) error {
		calls++
		if calls == 1 {
			generation = g
			return &capability.Failure{Outcome: capability.Retryable, Cause: errors.New("not sent")}
		}
		if g != generation {
			t.Error("retry changed reset identity")
		}
		return nil
	})
	c.SetDriveAllowed(context.Background(), true)
	if err := c.SetDriveAllowed(context.Background(), false); err == nil {
		t.Fatal("lost failure")
	}
	if c.DriveCapability().Outcome != capability.Retryable {
		t.Fatal(c.DriveCapability())
	}
	if err := c.SetDriveAllowed(context.Background(), true); err == nil {
		t.Fatal("enabled during failure")
	}
	if calls != 1 {
		t.Fatal("automatic retry")
	}
	if err := c.RetryDrive(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c.DriveCapability().Phase != capability.Off {
		t.Fatal(c.DriveCapability())
	}
}

// dispatchTab keeps physical holds independently of the injector's bookkeeping.
type dispatchTab struct {
	page     *recording.State
	acts     []Act
	failAt   int
	attempts int
	err      error
}

func (d *dispatchTab) Page() *recording.State                        { return d.page }
func (d *dispatchTab) At(float64, float64) (recording.NodeID, error) { return 0, nil }
func (d *dispatchTab) Dispatch(a Act) error {
	d.attempts++
	if d.failAt > 0 && len(d.acts)+1 == d.failAt {
		return d.err
	}
	d.acts = append(d.acts, a)
	return nil
}

type directRunner struct {
	tab    *dispatchTab
	before func()
}

func (d directRunner) WithTab(_ context.Context, _ string, f func(Tab) error) error {
	if d.before != nil {
		d.before()
	}
	return f(d.tab)
}
func TestReleaseRetainsPartialProgress(t *testing.T) {
	tab := &dispatchTab{page: &recording.State{}, failAt: 2, err: &capability.Failure{Outcome: capability.Retryable, Cause: errors.New("not sent")}}
	inj := NewTabInjector(directRunner{tab: tab})
	inj.held = pressed{by: "v1", button: true, keys: []Act{{Key: "a"}, {Key: "b"}}}
	if err := inj.Release(context.Background(), "v1"); err == nil {
		t.Fatal("lost release failure")
	}
	if inj.held.button || len(inj.held.keys) != 2 {
		t.Fatalf("forgot unfinished release: %+v", inj.held)
	}
	tab.failAt = 0
	if err := inj.Release(context.Background(), "v1"); err != nil {
		t.Fatal(err)
	}
	if len(tab.acts) != 3 || tab.acts[1].Key != "b" || tab.acts[2].Key != "a" {
		t.Fatal(tab.acts)
	}
}
func TestInjectorFinalAdmissionFence(t *testing.T) {
	tab := &dispatchTab{page: &recording.State{Live: true, Seg: 1, Control: recording.Control{Driver: &recording.Author{ID: "v1"}}}}
	allowed := true
	inj := NewTabInjector(directRunner{tab: tab, before: func() { allowed = false }})
	err := inj.Inject(context.Background(), "v1", Batch{Seg: 1, Acts: []Act{{A: ActText, V: "x"}}, Admit: func() bool { return allowed }})
	if err == nil || len(tab.acts) != 0 {
		t.Fatalf("dispatch escaped fence: %v %+v", err, tab.acts)
	}
}

type barrierInjector struct {
	entered chan struct{}
	allow   chan struct{}
	err     error
	calls   int
}

func (b *barrierInjector) Inject(context.Context, string, Batch) error { return nil }
func (b *barrierInjector) Release(ctx context.Context, _ string) error {
	b.calls++
	close(b.entered)
	select {
	case <-b.allow:
		return b.err
	case <-ctx.Done():
		return &capability.Failure{Outcome: capability.Retryable, Cause: ctx.Err()}
	}
}

func TestDriveOffClosesAdmissionBeforeCleanupAndJoinsClose(t *testing.T) {
	entered := make(chan struct{})
	exited := make(chan struct{})
	c := NewMemController(&recording.MemSink{}, nil, func() int64 { return 1 }, time.Hour)
	c.WithControlReset(func(ctx context.Context, _ uint64) error {
		close(entered)
		<-ctx.Done()
		close(exited)
		return &capability.Failure{Outcome: capability.Retryable, Cause: ctx.Err()}
	})
	c.SetDriveAllowed(context.Background(), true)
	seat, _ := c.Join("A")
	c.Request(seat.Key)
	c.Grant(seat.Viewer.ID)
	done := make(chan error, 1)
	go func() { done <- c.SetDriveAllowed(context.Background(), false) }()
	<-entered
	if _, err := c.AdmitDrive(); err == nil {
		t.Fatal("admitted during cleanup")
	}
	if err := c.Grant(seat.Viewer.ID); err == nil {
		t.Fatal("grant during cleanup")
	}
	if err := c.RetryDrive(context.Background()); err == nil {
		t.Fatal("overlapping retry")
	}
	if _, err := c.Join("B"); err != nil {
		t.Fatal("off hid seats", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	default:
		t.Fatal("Close did not join cleanup")
	}
	if err := <-done; err == nil {
		t.Fatal("canceled cleanup reported success")
	}
	if c.DriveCapability().Phase != capability.Ended {
		t.Fatal(c.DriveCapability())
	}
}
func TestControlReleaseBlocksGrantWithoutHoldingMutex(t *testing.T) {
	inj := &barrierInjector{entered: make(chan struct{}), allow: make(chan struct{}), err: errors.New("uncertain release")}
	c := NewMemController(&recording.MemSink{}, inj, func() int64 { return 1 }, time.Hour)
	defer c.Close()
	c.SetDriveAllowed(context.Background(), true)
	a, _ := c.Join("A")
	b, _ := c.Join("B")
	c.Request(a.Key)
	c.Grant(a.Viewer.ID)
	c.Request(b.Key)
	done := make(chan error, 1)
	go func() { done <- c.Revoke() }()
	<-inj.entered
	if err := c.Grant(b.Viewer.ID); err == nil {
		t.Fatal("grant overtook held-input release")
	}
	if err := c.Input(context.Background(), a.Key, Batch{}); err == nil {
		t.Fatal("input overtook release")
	}
	if _, err := c.Join("C"); err != nil {
		t.Fatal(err)
	}
	close(inj.allow)
	if err := <-done; err == nil {
		t.Fatal("failed release returned success")
	}
	if c.DriveCapability().Outcome != capability.Broken {
		t.Fatal(c.DriveCapability())
	}
	if err := c.RetryDrive(context.Background()); err == nil {
		t.Fatal("retried uncertain release")
	}
}
func TestUncertainReleaseNeverRetries(t *testing.T) {
	tab := &dispatchTab{page: &recording.State{}, failAt: 1, err: errors.New("transport lost")}
	inj := NewTabInjector(directRunner{tab: tab})
	inj.held = pressed{by: "v1", button: true}
	if err := inj.Release(context.Background(), "v1"); err == nil {
		t.Fatal("lost failure")
	}
	tab.failAt = 0
	if err := inj.Release(context.Background(), "v1"); err == nil {
		t.Fatal("retried uncertain up")
	}
	if len(tab.acts) != 0 || !inj.held.button {
		t.Fatal("lost uncertain hold")
	}
}
func TestMissingResetFailsClosed(t *testing.T) {
	c := NewMemController(&recording.MemSink{}, nil, func() int64 { return 1 }, time.Hour)
	defer c.Close()
	c.SetDriveAllowed(context.Background(), true)
	if err := c.SetDriveAllowed(context.Background(), false); err == nil {
		t.Fatal("missing ACK reset reported success")
	}
	if c.DriveCapability().Outcome != capability.Broken {
		t.Fatal(c.DriveCapability())
	}
}

func TestDriveOffResetsActivePendingAndReleasesHeld(t *testing.T) {
	stream := &recording.MemSink{}
	page := DrivenPage(100)
	for _, r := range page {
		stream.Put(r)
	}
	inj, err := NewFakeInjector(page, stream)
	if err != nil {
		t.Fatal(err)
	}
	c := NewMemController(recording.Tee(stream, inj), inj, func() int64 { return inj.Page().T + 1 }, time.Hour).WithControlReset(inj.ResetControl)
	defer c.Close()
	c.SetDriveAllowed(context.Background(), true)
	a, _ := c.Join("A")
	b, _ := c.Join("B")
	c.Request(a.Key)
	c.Grant(a.Viewer.ID)
	c.Request(b.Key)
	if err := c.Input(context.Background(), a.Key, Batch{Seg: inj.Page().Seg, Acts: []Act{{A: ActKey, Type: KeyDown, Key: "End"}}}); err != nil {
		t.Fatal(err)
	}
	if err := c.SetDriveAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	for _, ctl := range []recording.Control{c.Control(), inj.Page().Control} {
		if ctl.Driving() != "" || len(ctl.Pending) != 0 {
			t.Fatal(ctl)
		}
	}
	acts := inj.Dispatched()
	if len(acts) != 2 || acts[1].Type != KeyUp {
		t.Fatal(acts)
	}
	if err := inj.ResetControl(context.Background(), c.DriveCapability().Generation); err != nil {
		t.Fatal(err)
	}
	resets := 0
	for _, r := range stream.Records() {
		if ctl, ok := r.(*recording.Ctl); ok && ctl.Op == recording.CtlReset {
			resets++
		}
	}
	if resets != 1 {
		t.Fatalf("reset dedup: %d", resets)
	}
}

func TestUncertainInputUpIsNotRetriedByRefusalCleanup(t *testing.T) {
	tab := &dispatchTab{page: &recording.State{Live: true, Seg: 1, Control: recording.Control{Driver: &recording.Author{ID: "v1"}}}, failAt: 1, err: errors.New("unknown CDP result")}
	inj := NewTabInjector(directRunner{tab: tab})
	inj.held = pressed{by: "v1", seg: 1, keys: []Act{{Key: "End"}}}
	err := inj.Inject(context.Background(), "v1", Batch{Seg: 1, Acts: []Act{{A: ActKey, Type: KeyUp, Key: "End"}}})
	if tab.attempts != 1 {
		t.Fatalf("retried uncertain up %d times", tab.attempts)
	}
	var failure *ReleaseFailure
	if !errors.As(err, &failure) {
		t.Fatalf("uncertain up did not freeze injection: %v", err)
	}
	tab.failAt = 0
	if err = inj.Release(context.Background(), "v1"); err == nil || len(tab.acts) != 0 {
		t.Fatal("retried uncertain input up")
	}
}

func TestDriveCapabilitySubscribersLatestAndEnd(t *testing.T) {
	c := NewMemController(&recording.MemSink{}, nil, func() int64 { return 1 }, time.Hour)
	ch, cancel := c.SubscribeDriveCapability()
	defer cancel()
	if s := <-ch; s.Phase != capability.Off {
		t.Fatal(s)
	}
	c.WithControlReset(func(context.Context, uint64) error { return nil })
	c.SetDriveAllowed(context.Background(), true)
	c.SetDriveAllowed(context.Background(), false)
	if s := <-ch; s.Phase != capability.Off || s.Generation != 2 {
		t.Fatal(s)
	}
	c.Close()
	if s := <-ch; s.Phase != capability.Ended {
		t.Fatal(s)
	}
	if _, ok := <-ch; ok {
		t.Fatal("subscription survived close")
	}
}

func TestRefusedBatchCleanupFailureFreezesController(t *testing.T) {
	tab := &dispatchTab{page: &recording.State{Live: true, Seg: 1, Control: recording.Control{Driver: &recording.Author{ID: "v1"}}}, failAt: 1, err: &capability.Failure{Outcome: capability.Retryable, Cause: errors.New("up not sent")}}
	inj := NewTabInjector(directRunner{tab: tab})
	inj.held = pressed{by: "v1", seg: 1, button: true}
	c := NewMemController(&recording.MemSink{}, inj, func() int64 { return 1 }, time.Hour)
	defer c.Close()
	c.SetDriveAllowed(context.Background(), true)
	a, _ := c.Join("A")
	c.Request(a.Key)
	c.Grant(a.Viewer.ID)
	c.WithControlReset(func(context.Context, uint64) error { tab.page.Control = recording.Control{}; return nil })
	if err := c.Input(context.Background(), a.Key, Batch{Seg: 2, Acts: []Act{{A: ActText, V: "x"}}}); err == nil {
		t.Fatal("lost cleanup failure")
	}
	if c.DriveCapability().Outcome != capability.Retryable {
		t.Fatal(c.DriveCapability())
	}
	if err := c.Grant(a.Viewer.ID); err == nil {
		t.Fatal("grant during failed refusal cleanup")
	}
	tab.failAt = 0
	if err := c.RetryDrive(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(tab.acts) != 1 || tab.acts[0].A != ActUp {
		t.Fatal(tab.acts)
	}
}

type ownedInputInjector struct{ entered, canceled, finish chan struct{} }

func (i *ownedInputInjector) Inject(ctx context.Context, _ string, _ Batch) error {
	close(i.entered)
	<-ctx.Done()
	close(i.canceled)
	<-i.finish
	return ctx.Err()
}
func TestCloseCancelsAndJoinsOwnedInput(t *testing.T) {
	inj := &ownedInputInjector{make(chan struct{}), make(chan struct{}), make(chan struct{})}
	c := NewMemController(&recording.MemSink{}, inj, func() int64 { return 1 }, time.Hour)
	c.SetDriveAllowed(context.Background(), true)
	s, _ := c.Join("A")
	c.Request(s.Key)
	c.Grant(s.Viewer.ID)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inputDone := make(chan error, 1)
	go func() { inputDone <- c.Input(ctx, s.Key, Batch{}) }()
	<-inj.entered
	closed := make(chan struct{})
	go func() { c.Close(); close(closed) }()
	select {
	case <-inj.canceled:
	case <-time.After(time.Second):
		cancel()
		close(inj.finish)
		<-inputDone
		<-closed
		t.Fatal("Close did not cancel owned input")
	}
	select {
	case <-closed:
		close(inj.finish)
		<-inputDone
		t.Fatal("Close did not join owned input")
	default:
	}
	close(inj.finish)
	<-inputDone
	<-closed
}

type serializedRunner struct {
	mu  sync.Mutex
	tab Tab
}

func (r *serializedRunner) WithTab(_ context.Context, _ string, fn func(Tab) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fn(r.tab)
}

type uncertainTextTab struct {
	*dispatchTab
	entered, finish chan struct{}
}

func (t *uncertainTextTab) Dispatch(Act) error {
	close(t.entered)
	<-t.finish
	return errors.New("uncertain text dispatch")
}
func TestConcurrentDisableRetainsUncertainUnheldInput(t *testing.T) {
	tab := &uncertainTextTab{&dispatchTab{page: &recording.State{Live: true, Seg: 1, Control: recording.Control{Driver: &recording.Author{ID: "v1"}}}}, make(chan struct{}), make(chan struct{})}
	runner := &serializedRunner{tab: tab}
	inj := NewTabInjector(runner)
	c := NewMemController(&recording.MemSink{}, inj, func() int64 { return 1 }, time.Hour)
	defer c.Close()
	c.WithControlReset(func(context.Context, uint64) error {
		runner.mu.Lock()
		defer runner.mu.Unlock()
		tab.page.Control = recording.Control{}
		return nil
	})
	c.SetDriveAllowed(context.Background(), true)
	s, _ := c.Join("A")
	c.Request(s.Key)
	c.Grant(s.Viewer.ID)
	inputDone := make(chan error, 1)
	go func() {
		inputDone <- c.Input(context.Background(), s.Key, Batch{Seg: 1, Acts: []Act{{A: ActText, V: "x"}}})
	}()
	<-tab.entered
	offDone := make(chan error, 1)
	go func() { offDone <- c.SetDriveAllowed(context.Background(), false) }()
	if !eventually(func() bool { return c.DriveCapability().Phase == capability.Disabling }) {
		t.Fatal("OFF did not close admission")
	}
	close(tab.finish)
	<-inputDone
	if err := <-offDone; err == nil {
		t.Fatal("uncertain unheld input became successful OFF")
	}
	if c.DriveCapability().Outcome != capability.Broken {
		t.Fatal(c.DriveCapability())
	}
}

type uncertainControlSink struct {
	recording.MemSink
	fail bool
}

func (s *uncertainControlSink) Put(r recording.Record) error {
	if err := s.MemSink.Put(r); err != nil {
		return err
	}
	if s.fail {
		return errors.New("committed then connection lost")
	}
	return nil
}
func TestUncertainControlSinkFreezesAuthority(t *testing.T) {
	sink := &uncertainControlSink{}
	c := NewMemController(sink, nil, func() int64 { return 1 }, time.Hour)
	defer c.Close()
	c.SetDriveAllowed(context.Background(), true)
	s, _ := c.Join("A")
	sink.fail = true
	if err := c.Request(s.Key); err == nil {
		t.Fatal("lost sink failure")
	}
	if c.DriveCapability().Outcome != capability.Broken {
		t.Fatal("unknown control effect left authority open", c.DriveCapability())
	}
}

func TestOldAdmissionCannotReleaseRegrantedSeat(t *testing.T) {
	c := NewMemController(&recording.MemSink{}, nil, func() int64 { return 1 }, time.Hour).WithControlReset(func(context.Context, uint64) error { return nil })
	defer c.Close()
	c.SetDriveAllowed(context.Background(), true)
	s, _ := c.Join("A")
	c.Request(s.Key)
	c.Grant(s.Viewer.ID)
	old, err := c.AdmitDrive()
	if err != nil {
		t.Fatal(err)
	}
	if err = c.SetDriveAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	c.SetDriveAllowed(context.Background(), true)
	c.Request(s.Key)
	c.Grant(s.Viewer.ID)
	if err = old.Release(s.Key); err == nil {
		t.Fatal("old release revoked regranted seat")
	}
	if c.Control().Driving() != s.Viewer.ID {
		t.Fatal("new grant lost")
	}
	fresh, err := c.AdmitDrive()
	if err != nil {
		t.Fatal(err)
	}
	if err = fresh.Release(s.Key); err != nil {
		t.Fatal(err)
	}
	if c.Control().Driving() != "" {
		t.Fatal("fresh release did not revoke")
	}
}

type rejectQueuedRunner struct {
	directRunner
	reject error
}

func (r *rejectQueuedRunner) WithTab(ctx context.Context, by string, fn func(Tab) error) error {
	if r.reject != nil {
		return r.reject
	}
	return r.directRunner.WithTab(ctx, by, fn)
}
func TestCanceledQueuedUpFreezesUntilExplicitCleanup(t *testing.T) {
	tab := &dispatchTab{page: &recording.State{Live: true, Seg: 1, Control: recording.Control{Driver: &recording.Author{ID: "v1"}}}}
	runner := &rejectQueuedRunner{directRunner: directRunner{tab: tab}}
	inj := NewTabInjector(runner)
	c := NewMemController(&recording.MemSink{}, inj, func() int64 { return 1 }, time.Hour)
	defer c.Close()
	c.SetDriveAllowed(context.Background(), true)
	s, _ := c.Join("A")
	c.Request(s.Key)
	c.Grant(s.Viewer.ID)
	c.WithControlReset(func(context.Context, uint64) error { tab.page.Control = recording.Control{}; return nil })
	batch := Batch{Seg: 1, Acts: []Act{{A: ActKey, Type: KeyDown, Key: "End"}}}
	if err := c.Input(context.Background(), s.Key, batch); err != nil {
		t.Fatal(err)
	}
	runner.reject = &capability.Failure{Outcome: capability.Retryable, Cause: context.Canceled}
	batch.Acts[0].Type = KeyUp
	if err := c.Input(context.Background(), s.Key, batch); err == nil {
		t.Fatal("lost queued cancellation")
	}
	if c.DriveCapability().Outcome != capability.Retryable {
		t.Fatal("held key left authority ON", c.DriveCapability())
	}
	if len(tab.acts) != 1 {
		t.Fatal("canceled up was dispatched")
	}
	runner.reject = nil
	if err := c.RetryDrive(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(tab.acts) != 2 || tab.acts[1].Type != KeyUp {
		t.Fatal("held key not cleaned", tab.acts)
	}
}
