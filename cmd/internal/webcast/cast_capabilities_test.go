package webcast

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"testing"
)

func TestCastCapabilitiesDefaultsAndStableSecrets(t *testing.T) {
	c, d := startDrivenCast(t, &recording.MemSink{})
	caps := c.Capabilities()
	if !caps.AnnotationSupported || !caps.DriveSupported || caps.Annotation.Phase != capability.Off || caps.Drive.Phase != capability.Off {
		t.Fatalf("defaults: %+v", caps)
	}
	tokens := [3]string{c.token, c.pointerToken, c.driveToken}
	codes := c.Codes()
	d.Open()
	for _, kind := range []CapabilityKind{CapabilityAnnotation, CapabilityDrive} {
		if err := c.SetCapability(context.Background(), kind, true); err != nil {
			t.Fatal(err)
		}
		if err := c.SetCapability(context.Background(), kind, false); err != nil {
			t.Fatal(err)
		}
	}
	if tokens != [3]string{c.token, c.pointerToken, c.driveToken} {
		t.Fatal("toggle changed tokens")
	}
	after := c.Codes()
	for i := range codes {
		if codes[i] != after[i] {
			t.Fatal("toggle changed codes")
		}
	}
	c.End()
	if c.Capabilities().Annotation.Phase != capability.Ended || c.Capabilities().Drive.Phase != capability.Ended {
		t.Fatal("capabilities did not end")
	}
}

func TestCastCapabilitySingleFlightRetryAndEnd(t *testing.T) {
	bc := startRecordedBoardCast(t)
	c := bc.c
	bc.open(t)
	entered, release := make(chan struct{}), make(chan struct{})
	c.board.WithAnnotationCleanup(func(ctx context.Context, id uint64) error {
		close(entered)
		select {
		case <-release:
			return &capability.Failure{Outcome: capability.Retryable, Cause: context.DeadlineExceeded}
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	if err := c.SetCapability(context.Background(), CapabilityAnnotation, true); err != nil {
		t.Fatal(err)
	}
	done, err := c.EnqueueCapability(CapabilityAnnotation, capability.Disable)
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	if _, err := c.EnqueueCapability(CapabilityAnnotation, capability.Disable); err != ErrCapabilityBusy {
		t.Fatalf("duplicate: %v", err)
	}
	if got := c.Capabilities().Project(CapabilityAnnotation); got.State != "turning-off" || got.Enabled {
		t.Fatalf("in flight: %+v", got)
	}
	close(release)
	if err := <-done; err == nil {
		t.Fatal("missing failure")
	}
	if got := c.Capabilities().Project(CapabilityAnnotation); got.Action != "retry" {
		t.Fatalf("retry: %+v", got)
	}
	c.board.WithAnnotationCleanup(bc.eb.ClearAnnotations)
	if err := c.RetryCapability(context.Background(), CapabilityAnnotation); err != nil {
		t.Fatal(err)
	}
	if c.Capabilities().Annotation.Phase != capability.Off {
		t.Fatal("retry enabled capability")
	}
	if err := c.SetCapability(context.Background(), CapabilityAnnotation, true); err != nil {
		t.Fatal(err)
	}
	entered = make(chan struct{})
	exited := make(chan struct{})
	c.board.WithAnnotationCleanup(func(ctx context.Context, id uint64) error {
		close(entered)
		<-ctx.Done()
		close(exited)
		return ctx.Err()
	})
	done, err = c.EnqueueCapability(CapabilityAnnotation, capability.Disable)
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	c.End()
	<-exited
	<-done
	if c.Capabilities().Annotation.Phase != capability.Ended {
		t.Fatal("late work replaced ended state")
	}
}

func TestCastBrokenCapabilityFencesOtherOwner(t *testing.T) {
	c, d := startDrivenCast(t, &recording.MemSink{})
	for _, kind := range []CapabilityKind{CapabilityAnnotation, CapabilityDrive} {
		if err := c.SetCapability(context.Background(), kind, true); err != nil {
			t.Fatal(err)
		}
	}
	d.FailEmits(context.Canceled) // unclassified transport failure is terminal
	if err := c.SetCapability(context.Background(), CapabilityAnnotation, false); err == nil {
		t.Fatal("missing failure")
	}
	if _, err := c.drive.AdmitDrive(); err == nil {
		t.Fatal("other authority still admits")
	}
	for _, kind := range []CapabilityKind{CapabilityAnnotation, CapabilityDrive} {
		if got := c.Capabilities().Project(kind); got.Action != "end-cast" || got.Enabled {
			t.Fatalf("broken: %+v", got)
		}
		if err := c.SetCapability(context.Background(), kind, true); err != ErrCastBroken {
			t.Fatalf("reopen: %v", err)
		}
	}
}

func TestCastCapabilitySubscriptionReplaysLatest(t *testing.T) {
	c, _ := startDrivenCast(t, &recording.MemSink{})
	slow, cancel := c.SubscribeCapabilities()
	defer cancel()
	if err := c.SetCapability(context.Background(), CapabilityAnnotation, true); err != nil {
		t.Fatal(err)
	}
	if got := <-slow; got.Annotation.Phase != capability.On || got.Drive.Phase != capability.Off {
		t.Fatalf("slow subscriber: %+v", got)
	}
	latest, stop := c.SubscribeCapabilities()
	defer stop()
	if got := <-latest; got.Annotation.Phase != capability.On || got.Drive.Phase != capability.Off {
		t.Fatalf("late subscriber: %+v", got)
	}
	if got := c.Current().Capabilities; got.Annotation.Phase != capability.On {
		t.Fatalf("current update: %+v", got)
	}
	c.End()
	if got, ok := <-latest; !ok || got.Annotation.Phase != capability.Ended {
		t.Fatalf("final: %+v %v", got, ok)
	}
	if _, ok := <-latest; ok {
		t.Fatal("subscription did not close")
	}
}

func TestCastSimultaneousCapabilityFailuresPreserveBroken(t *testing.T) {
	c, _ := startDrivenCast(t, &recording.MemSink{})
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	fail := func(ctx context.Context, id uint64) error { entered <- struct{}{}; <-release; return context.Canceled }
	c.board.WithAnnotationCleanup(fail)
	c.drive.(*drive.MemController).WithControlReset(fail)
	for _, kind := range []CapabilityKind{CapabilityAnnotation, CapabilityDrive} {
		if err := c.SetCapability(context.Background(), kind, true); err != nil {
			t.Fatal(err)
		}
	}
	a, err := c.EnqueueCapability(CapabilityAnnotation, capability.Disable)
	if err != nil {
		t.Fatal(err)
	}
	d, err := c.EnqueueCapability(CapabilityDrive, capability.Disable)
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	<-entered
	close(release)
	<-a
	<-d
	if !c.Capabilities().Broken() {
		t.Fatalf("lost terminal failure: %+v", c.Capabilities())
	}
	if _, err := c.board.AdmitAnnotation(); err == nil {
		t.Fatal("annotation still admits")
	}
	if _, err := c.drive.AdmitDrive(); err == nil {
		t.Fatal("drive still admits")
	}
}
