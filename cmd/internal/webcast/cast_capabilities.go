package webcast

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
)

type CapabilityKind string

const (
	CapabilityAnnotation CapabilityKind = "annotation"
	CapabilityDrive      CapabilityKind = "drive"
)

// Capabilities derives from the two authorities; support is independent of admission.
type Capabilities struct {
	Annotation          capability.Snapshot
	Drive               capability.Snapshot
	AnnotationSupported bool
	DriveSupported      bool
}

func (s Capabilities) Broken() bool {
	return s.Annotation.Outcome == capability.Broken || s.Drive.Outcome == capability.Broken
}

// Project applies a terminal failure to both controls without inventing an
// enabled bit. Cast fences the other authority before completing the action.
func (s Capabilities) Project(kind CapabilityKind) capability.Presentation {
	if s.Broken() {
		return capability.Presentation{State: "broken", Action: "end-cast"}
	}
	if kind == CapabilityAnnotation {
		return capability.Project(s.Annotation.CapabilityState)
	}
	return capability.Project(s.Drive.CapabilityState)
}

func (c *Cast) Capabilities() Capabilities {
	s := Capabilities{AnnotationSupported: c.board != nil, DriveSupported: c.drive != nil}
	if c.board != nil {
		s.Annotation = c.board.AnnotationCapability()
	}
	if c.drive != nil {
		s.Drive = c.drive.DriveCapability()
	}
	return s
}
func (c *Cast) SubscribeCapabilities() (<-chan Capabilities, func()) {
	c.capWork.publish.Lock()
	defer c.capWork.publish.Unlock()
	c.capPub.publish(c.Capabilities())
	return c.capPub.subscribe()
}

var ErrCapabilityBusy = errors.New("webcast: capability operation in progress")
var ErrCapabilityUnsupported = errors.New("webcast: capability unsupported")
var ErrCastBroken = errors.New("webcast: cast broken; end the cast")

type capabilityWork struct {
	publish sync.Mutex
	fence   sync.Mutex
	mu      sync.Mutex
	running map[CapabilityKind]bool
	wg      sync.WaitGroup
	stopped bool
}

// EnqueueCapability never waits for recorder/CDP. At most one operation per
// capability is owned at a time; result delivery cannot block its worker.
func (c *Cast) EnqueueCapability(kind CapabilityKind, action capability.EventKind) (<-chan error, error) {
	if kind != CapabilityAnnotation && kind != CapabilityDrive {
		return nil, ErrCapabilityUnsupported
	}
	if action != capability.Enable && action != capability.Disable && action != capability.Retry {
		return nil, errors.New("webcast: invalid capability action")
	}
	c.capWork.mu.Lock()
	defer c.capWork.mu.Unlock()
	if c.capWork.stopped || c.capCtx.Err() != nil {
		return nil, ErrEnded
	}
	if c.capWork.running[kind] {
		return nil, ErrCapabilityBusy
	}
	c.capWork.running[kind] = true
	c.capWork.wg.Add(1)
	result := make(chan error, 1)
	go func() {
		defer c.capWork.wg.Done()
		ctx, cancel := context.WithTimeout(c.capCtx, 10*time.Second)
		defer cancel()
		err := c.changeCapability(ctx, kind, action)
		c.capWork.mu.Lock()
		delete(c.capWork.running, kind)
		c.capWork.mu.Unlock()
		result <- err
		close(result)
	}()
	return result, nil
}
func (c *Cast) SetCapability(ctx context.Context, kind CapabilityKind, allowed bool) error {
	action := capability.Disable
	if allowed {
		action = capability.Enable
	}
	return c.waitCapability(ctx, kind, action)
}
func (c *Cast) RetryCapability(ctx context.Context, kind CapabilityKind) error {
	return c.waitCapability(ctx, kind, capability.Retry)
}
func (c *Cast) waitCapability(ctx context.Context, kind CapabilityKind, action capability.EventKind) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	done, err := c.EnqueueCapability(kind, action)
	if err != nil {
		return err
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (c *Cast) changeCapability(ctx context.Context, kind CapabilityKind, action capability.EventKind) error {
	if c.Capabilities().Broken() {
		return ErrCastBroken
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var err error
	switch kind {
	case CapabilityAnnotation:
		if c.board == nil {
			return ErrCapabilityUnsupported
		}
		if action == capability.Retry {
			err = c.board.RetryAnnotations(ctx)
		} else {
			err = c.board.SetAnnotationAllowed(ctx, action == capability.Enable)
		}
	case CapabilityDrive:
		if c.drive == nil {
			return ErrCapabilityUnsupported
		}
		if action == capability.Retry {
			err = c.drive.RetryDrive(ctx)
		} else {
			err = c.drive.SetDriveAllowed(ctx, action == capability.Enable)
		}
	}
	c.fenceBrokenCapability()
	c.publishCapabilities()
	return err
}
func (c *Cast) fenceBrokenCapability() {
	c.capWork.fence.Lock()
	defer c.capWork.fence.Unlock()
	s := c.Capabilities()
	if s.Annotation.Outcome == capability.Broken && c.drive != nil {
		c.drive.Close()
	} else if s.Drive.Outcome == capability.Broken && c.board != nil {
		c.board.Close()
	}
}
func (c *Cast) stopCapabilityWork() {
	c.capWork.mu.Lock()
	c.capWork.stopped = true
	c.cancel()
	c.capWork.mu.Unlock()
	c.capWork.wg.Wait()
}

func (c *Cast) publishCapabilities() {
	c.capWork.publish.Lock()
	defer c.capWork.publish.Unlock()
	c.capPub.publish(c.Capabilities())
}
