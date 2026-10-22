package drive

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func drawingController(t *testing.T, sink recording.Sink, inj Injector) *MemController {
	t.Helper()
	c := NewMemController(sink, inj, func() int64 { return 1 }, time.Hour)
	t.Cleanup(func() { c.Close() })
	if err := c.SetDriveAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	return c
}
func drawingSeat(t *testing.T, c *MemController, name string) Seat {
	t.Helper()
	s, err := c.Join(name)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func drawingGrant(t *testing.T, c *MemController, s Seat) {
	t.Helper()
	if err := c.Request(s.Key); err != nil {
		t.Fatal(err)
	}
	if err := c.Grant(s.Viewer.ID); err != nil {
		t.Fatal(err)
	}
}
func drawingWait(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("drawing barrier not reached")
	}
}

func TestDrawingLocalOffAndClose(t *testing.T) {
	c := NewMemController(&recording.MemSink{}, nil, func() int64 { return 1 }, time.Hour)
	a := c.DrawingAuthority()
	if a.Epoch == 0 || a.Holder != recording.OperatorID || !a.Ready {
		t.Fatalf("initial local authority %+v", a)
	}
	p, err := c.AdmitOperatorDrawing(a.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if p.Principal().ID != recording.OperatorID || p.Check() != nil {
		t.Fatal("local operator permit missing")
	}
	if _, err := c.AdmitDrawing(a.Epoch, "unknown"); !errors.Is(err, ErrNoSeat) {
		t.Fatalf("unknown seat %v", err)
	}
	if (DrawingPermit{}).Check() == nil {
		t.Fatal("zero permit admitted")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if p.Check() == nil || c.DrawingAuthority().Ready {
		t.Fatal("close retained drawing authority")
	}
	if _, err := c.AdmitOperatorDrawing(c.DrawingAuthority().Epoch); err == nil {
		t.Fatal("closed controller admitted drawing")
	}
}

func TestDrawingHolderEpochRejectsABAAndPrincipalClaims(t *testing.T) {
	c := drawingController(t, &recording.MemSink{}, nil)
	a := drawingSeat(t, c, "Ana")
	b := drawingSeat(t, c, "Bo")
	drawingGrant(t, c, a)
	first := c.DrawingAuthority()
	p, err := c.AdmitDrawing(first.Epoch, a.Key)
	if err != nil {
		t.Fatal(err)
	}
	if p.Principal() != a.Viewer {
		t.Fatalf("principal %+v", p.Principal())
	}
	if _, err := c.AdmitDrawing(first.Epoch, b.Key); !errors.Is(err, ErrNotDriver) {
		t.Fatalf("spectator permit %v", err)
	}
	if _, err := c.AdmitOperatorDrawing(first.Epoch); !errors.Is(err, ErrNotDriver) {
		t.Fatalf("operator while remote %v", err)
	}
	drawingGrant(t, c, b)
	drawingGrant(t, c, a)
	if p.Check() == nil {
		t.Fatal("old A witness revived")
	}
	if _, err := c.AdmitDrawing(first.Epoch, a.Key); err == nil {
		t.Fatal("pre-ingress A->B->A begin admitted")
	}
	last := c.DrawingAuthority()
	if last.Epoch <= first.Epoch || last.Holder != a.Viewer.ID || !last.Ready {
		t.Fatalf("ABA authority %+v", last)
	}
	if _, err := c.AdmitDrawing(last.Epoch, a.Key); err != nil {
		t.Fatal(err)
	}
}

type drawingControlSink struct {
	recording.MemSink
	hold           string
	entered, allow chan struct{}
	failure        error
	acceptFailure  bool
}

func (s *drawingControlSink) Put(r recording.Record) error {
	return s.WriteControl(context.Background(), r.(*recording.Ctl))
}
func (s *drawingControlSink) WriteControl(ctx context.Context, r *recording.Ctl) error {
	if r.Op == s.hold {
		close(s.entered)
		select {
		case <-s.allow:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if s.failure != nil && !s.acceptFailure {
		return s.failure
	}
	if err := s.MemSink.Put(r); err != nil {
		return err
	}
	return s.failure
}

func TestDrawingHandoffFencesBeforeControlWriteAndWakesCapabilities(t *testing.T) {
	sink := &drawingControlSink{}
	c := drawingController(t, sink, nil)
	a := drawingSeat(t, c, "Ana")
	before := c.DrawingAuthority()
	permit, err := c.AdmitOperatorDrawing(before.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Request(a.Key); err != nil {
		t.Fatal(err)
	}
	wake, cancel := c.SubscribeDriveCapability()
	defer cancel()
	<-wake
	sink.hold = recording.CtlGrant
	sink.entered = make(chan struct{})
	sink.allow = make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- c.Grant(a.Viewer.ID) }()
	drawingWait(t, sink.entered)
	if permit.Check() == nil {
		t.Fatal("old witness live at held ctl write")
	}
	during := c.DrawingAuthority()
	if during.Ready || during.Epoch <= before.Epoch {
		t.Fatalf("handoff authority %+v", during)
	}
	select {
	case <-wake:
	default:
		t.Fatal("unchanged capability enum did not wake authority feed")
	}
	if _, err := c.AdmitDrawing(during.Epoch, a.Key); err == nil {
		t.Fatal("grant became ready before acknowledgment")
	}
	close(sink.allow)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	after := c.DrawingAuthority()
	if !after.Ready || after.Holder != a.Viewer.ID || after.Epoch != during.Epoch {
		t.Fatalf("settled authority %+v", after)
	}
	select {
	case <-wake:
	default:
		t.Fatal("ready transition did not wake authority feed")
	}
}

func TestDrawingPendingOnlyControlPreservesWitnessEvenWhileWriteHeld(t *testing.T) {
	sink := &drawingControlSink{}
	c := drawingController(t, sink, nil)
	a := drawingSeat(t, c, "Ana")
	drawingGrant(t, c, a)
	b := drawingSeat(t, c, "Bo")
	before := c.DrawingAuthority()
	permit, err := c.AdmitDrawing(before.Epoch, a.Key)
	if err != nil {
		t.Fatal(err)
	}
	sink.hold = recording.CtlRequest
	sink.entered = make(chan struct{})
	sink.allow = make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- c.Request(b.Key) }()
	drawingWait(t, sink.entered)
	if permit.Check() != nil || c.DrawingAuthority() != before {
		t.Fatal("pending request retired drawing")
	}
	close(sink.allow)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := c.Deny(b.Viewer.ID); err != nil {
		t.Fatal(err)
	}
	sink.hold = ""
	if err := c.Request(b.Key); err != nil {
		t.Fatal(err)
	}
	if err := c.Release(b.Key); err != nil {
		t.Fatal(err)
	}
	if permit.Check() != nil || c.DrawingAuthority() != before {
		t.Fatal("pending deny/withdrawal retired drawing")
	}
	// Final writer validation must not need the controller lock.
	c.mu.Lock()
	check := permit.Check()
	c.mu.Unlock()
	if check != nil {
		t.Fatal(check)
	}
}

func TestDrawingWaitsForHeldReleaseAndFailureStaysClosed(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "unknown release"}[fail], func(t *testing.T) {
			inj := &barrierInjector{entered: make(chan struct{}), allow: make(chan struct{})}
			if fail {
				inj.err = errors.New("release unknown")
			}
			c := drawingController(t, &recording.MemSink{}, inj)
			a := drawingSeat(t, c, "Ana")
			drawingGrant(t, c, a)
			before := c.DrawingAuthority()
			permit, err := c.AdmitDrawing(before.Epoch, a.Key)
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- c.Revoke() }()
			drawingWait(t, inj.entered)
			if permit.Check() == nil || c.DrawingAuthority().Ready {
				t.Fatal("old/new drawer admitted during release")
			}
			if _, err := c.AdmitOperatorDrawing(c.DrawingAuthority().Epoch); err == nil {
				t.Fatal("operator admitted during release")
			}
			close(inj.allow)
			err = <-done
			if fail {
				if err == nil || c.DrawingAuthority().Ready {
					t.Fatal("failed release reopened drawing")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				a := c.DrawingAuthority()
				if !a.Ready || a.Holder != recording.OperatorID {
					t.Fatal(a)
				}
			}
		})
	}
}

func TestDrawingResetFencesAndLocalOffRenewsOnlyAfterRelease(t *testing.T) {
	resetIn, resetAllow := make(chan struct{}), make(chan struct{})
	inj := &barrierInjector{entered: make(chan struct{}), allow: make(chan struct{})}
	c := drawingController(t, &recording.MemSink{}, inj)
	c.WithControlReset(func(ctx context.Context, _ uint64) error {
		close(resetIn)
		select {
		case <-resetAllow:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	a := drawingSeat(t, c, "Ana")
	drawingGrant(t, c, a)
	before := c.DrawingAuthority()
	permit, err := c.AdmitDrawing(before.Epoch, a.Key)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- c.SetDriveAllowed(context.Background(), false) }()
	drawingWait(t, resetIn)
	if permit.Check() == nil || c.DrawingAuthority().Ready || c.DrawingAuthority().Epoch <= before.Epoch {
		t.Fatal("reset failed to fence before write")
	}
	close(resetAllow)
	drawingWait(t, inj.entered)
	if c.DrawingAuthority().Ready {
		t.Fatal("reset reopened before input release")
	}
	close(inj.allow)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if c.DriveCapability().Phase != capability.Off {
		t.Fatal("reset did not disable drive")
	}
	if _, err := c.AdmitOperatorDrawing(c.DrawingAuthority().Epoch); err != nil {
		t.Fatalf("local off permit %v", err)
	}
}

func TestDrawingUnknownControlSinkDoesNotRenewWitness(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject before write", true: "accept then fail"}[accept], func(t *testing.T) {
			sink := &drawingControlSink{}
			c := drawingController(t, sink, nil)
			a := drawingSeat(t, c, "Ana")
			if err := c.Request(a.Key); err != nil {
				t.Fatal(err)
			}
			before := c.DrawingAuthority()
			permit, err := c.AdmitOperatorDrawing(before.Epoch)
			if err != nil {
				t.Fatal(err)
			}
			sink.failure = errors.New("ctl result unknown")
			sink.acceptFailure = accept
			if c.Grant(a.Viewer.ID) == nil {
				t.Fatal("missing control sink failure")
			}
			if permit.Check() == nil || c.DrawingAuthority().Ready {
				t.Fatal("unknown write renewed drawing")
			}
		})
	}
}

func TestDrawingEpochOverflowStaysClosed(t *testing.T) {
	c := drawingController(t, &recording.MemSink{}, nil)
	a := drawingSeat(t, c, "Ana")
	c.mu.Lock()
	c.drawingEpoch = recording.MaxSketchInteger
	c.mu.Unlock()
	drawingGrant(t, c, a)
	if state := c.DrawingAuthority(); state.Ready || state.Epoch != recording.MaxSketchInteger {
		t.Fatalf("overflow reopened/wrapped drawing %+v", state)
	}
}

func TestDrawingMetadataAuthorSurvivesHandoffWithoutDrawing(t *testing.T) {
	c := drawingController(t, &recording.MemSink{}, nil)
	a := drawingSeat(t, c, "Ana")
	b := drawingSeat(t, c, "Bo")
	drawingGrant(t, c, a)
	old := c.DrawingAuthority()
	drawingGrant(t, c, b)
	author, err := c.AuthenticateDrawingAuthor(a.Key)
	if err != nil || author != a.Viewer {
		t.Fatalf("original author after handoff %+v %v", author, err)
	}
	if _, err := c.AdmitDrawing(old.Epoch, a.Key); err == nil {
		t.Fatal("metadata authentication refreshed drawing")
	}
	if _, err := c.AuthenticateDrawingAuthor(""); !errors.Is(err, ErrNoSeat) {
		t.Fatalf("blank metadata key: %v", err)
	}
	if _, err := c.AuthenticateDrawingAuthor(b.Viewer.ID); !errors.Is(err, ErrNoSeat) {
		t.Fatalf("attribution accepted as key: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AuthenticateDrawingAuthor(a.Key); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed author %v", err)
	}
}

func TestDrawingSeatExpiryRevokesCredentialsAndOnlyHolderTurn(t *testing.T) {
	for _, driverLeaves := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending seat leaves", true: "driver leaves"}[driverLeaves], func(t *testing.T) {
			sink := &drawingControlSink{}
			c := drawingController(t, sink, nil)
			a := drawingSeat(t, c, "Ana")
			drawingGrant(t, c, a)
			leaving := a
			if !driverLeaves {
				leaving = drawingSeat(t, c, "Bo")
				if err := c.Request(leaving.Key); err != nil {
					t.Fatal(err)
				}
			}
			detach, err := c.Attach(leaving.Viewer.ID)
			if err != nil {
				t.Fatal(err)
			}
			before := c.DrawingAuthority()
			permit, err := c.AdmitDrawing(before.Epoch, a.Key)
			if err != nil {
				t.Fatal(err)
			}
			sink.hold = recording.CtlLeave
			sink.entered = make(chan struct{})
			sink.allow = make(chan struct{})
			changes, cancel := c.Subscribe()
			defer cancel()
			<-changes
			c.mu.Lock()
			c.grace = time.Nanosecond
			c.mu.Unlock()
			detach()
			drawingWait(t, sink.entered)
			if _, err := c.AuthenticateDrawingAuthor(leaving.Key); !errors.Is(err, ErrNoSeat) {
				t.Fatalf("expired metadata credential %v", err)
			}
			if driverLeaves {
				if permit.Check() == nil || c.DrawingAuthority().Ready {
					t.Fatal("leaving holder retained witness")
				}
			} else {
				if permit.Check() != nil || c.DrawingAuthority() != before {
					t.Fatal("pending leave changed drawer")
				}
			}
			close(sink.allow)
			select {
			case <-changes:
			case <-time.After(time.Second):
				t.Fatal("leave did not settle")
			}
			if driverLeaves {
				after := c.DrawingAuthority()
				if !after.Ready || after.Holder != recording.OperatorID {
					t.Fatalf("leave authority %+v", after)
				}
			}
		})
	}
}

func TestDrawingLocalResetRetiresSameHolderAndWaitsForAcknowledgment(t *testing.T) {
	c := drawingController(t, &recording.MemSink{}, nil)
	entered, allow := make(chan struct{}), make(chan struct{})
	c.WithControlReset(func(ctx context.Context, _ uint64) error {
		close(entered)
		select {
		case <-allow:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	before := c.DrawingAuthority()
	permit, err := c.AdmitOperatorDrawing(before.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- c.SetDriveAllowed(context.Background(), false) }()
	drawingWait(t, entered)
	if permit.Check() == nil || c.DrawingAuthority().Ready {
		t.Fatal("same-holder reset kept witness")
	}
	close(allow)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	after := c.DrawingAuthority()
	if !after.Ready || after.Epoch <= before.Epoch || after.Holder != before.Holder {
		t.Fatalf("local reset %+v -> %+v", before, after)
	}
	if _, err := c.AdmitOperatorDrawing(before.Epoch); err == nil {
		t.Fatal("old local epoch refreshed")
	}
}

func TestDrawingFailedResetStaysClosedUntilSuccessfulRetry(t *testing.T) {
	c := drawingController(t, &recording.MemSink{}, nil)
	failed := true
	c.WithControlReset(func(context.Context, uint64) error {
		if failed {
			return &capability.Failure{Outcome: capability.Retryable, Cause: errors.New("reset not dispatched")}
		}
		return nil
	})
	before := c.DrawingAuthority()
	permit, err := c.AdmitOperatorDrawing(before.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if c.SetDriveAllowed(context.Background(), false) == nil {
		t.Fatal("missing reset refusal")
	}
	if permit.Check() == nil || c.DrawingAuthority().Ready {
		t.Fatal("failed reset reopened drawer")
	}
	failed = false
	if err := c.RetryDrive(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AdmitOperatorDrawing(c.DrawingAuthority().Epoch); err != nil {
		t.Fatal(err)
	}
	if permit.Check() == nil {
		t.Fatal("successful cleanup revived old permit")
	}
}

type drawingNormalInput struct{ by []string }

func (i *drawingNormalInput) Inject(_ context.Context, by string, b Batch) error {
	if b.Admit == nil || !b.Admit() {
		return ErrNotDriver
	}
	i.by = append(i.by, by)
	return nil
}
func TestDrawingEpochDoesNotAlterNormalDriveAdmission(t *testing.T) {
	inj := &drawingNormalInput{}
	c := drawingController(t, &recording.MemSink{}, inj)
	a := drawingSeat(t, c, "Ana")
	b := drawingSeat(t, c, "Bo")
	drawingGrant(t, c, a)
	normal, err := c.AdmitDrive()
	if err != nil {
		t.Fatal(err)
	}
	drawingGrant(t, c, b)
	drawingGrant(t, c, a)
	if err := normal.Input(context.Background(), a.Key, Batch{}); err != nil {
		t.Fatalf("sketch epoch changed existing drive admission: %v", err)
	}
	if len(inj.by) != 1 || inj.by[0] != a.Viewer.ID {
		t.Fatal("ordinary input did not dispatch for current holder")
	}
}

func TestDrawingMetadataWitnessSurvivesHandoffButNotSeatLifetime(t *testing.T) {
	c := drawingController(t, &recording.MemSink{}, nil)
	a := drawingSeat(t, c, "Ana")
	proof, err := c.AdmitDrawingMetadata(a.Key)
	if err != nil {
		t.Fatal(err)
	}
	drawingGrant(t, c, a)
	if err := c.Revoke(); err != nil {
		t.Fatal(err)
	}
	if proof.Check() != nil || proof.Principal() != a.Viewer {
		t.Fatal("handoff revoked metadata lifetime")
	}
	// A recorder callback checks this witness without acquiring controller.mu.
	c.mu.Lock()
	if proof.Check() != nil {
		t.Fatal("live identity failed")
	}
	close(c.seats[a.Viewer.ID].drawingLife)
	delete(c.seats, a.Viewer.ID)
	delete(c.keys, a.Key)
	c.mu.Unlock()
	if proof.Check() == nil {
		t.Fatal("removed seat survived")
	}
	if _, err := c.AdmitDrawingMetadata(a.Key); err == nil {
		t.Fatal("expired key admitted")
	}
	b := drawingSeat(t, c, "Bea")
	later, err := c.AdmitDrawingMetadata(b.Key)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
	if later.Check() == nil {
		t.Fatal("cast close retained metadata identity")
	}
}
