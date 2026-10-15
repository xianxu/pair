package couchcore

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Copies of a configured checker share its scope; different constructors and
// roots cannot accidentally consume each other's initial observations.
type sessionOwnerScope struct{ marker byte }
type sessionOwnerPassKey struct{}
type sessionOwnerPassBinding struct {
	scope *sessionOwnerScope
	root  string
	pass  *launcher.SessionOwnerPass
}

// WithSessionOwnerPass creates a lazy initial observation for this checker.
// Opaque custom probes keep their explicit behavior; SessionOwnerProbe's IO
// seam remains available for stateful compositions. Configuration is fixed for
// the lifetime of a pass. Callers cancel when the pass ends.
func (c ScopedThreadArtifactCollisionChecker) WithSessionOwnerPass(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	var probe launcher.SessionOwnerProbe
	switch p := c.OwnerProbe.(type) {
	case nil:
	case launcher.SessionOwnerProbe:
		probe = p
	case *launcher.SessionOwnerProbe:
		if p == nil {
			return ctx, func() {}
		}
		probe = *p
	default:
		return ctx, func() {}
	}
	if c.ownerScope == nil {
		return ctx, func() {}
	}
	parent, cancel := context.WithCancel(ctx)
	binding := sessionOwnerPassBinding{scope: c.ownerScope, root: c.GlobalDataDir, pass: launcher.NewSessionOwnerPass(parent, c.serverStates(), probe)}
	return context.WithValue(parent, sessionOwnerPassKey{}, binding), cancel
}
func (c ScopedThreadArtifactCollisionChecker) ownerProbeForContext(ctx context.Context) SessionOwnerProber {
	if ctx != nil {
		if b, ok := ctx.Value(sessionOwnerPassKey{}).(sessionOwnerPassBinding); ok && b.scope != nil && b.scope == c.ownerScope && b.root == c.GlobalDataDir {
			return b.pass
		}
	}
	return c.ownerProbe()
}
func withoutSessionOwnerPass(ctx context.Context) context.Context {
	return context.WithValue(ctx, sessionOwnerPassKey{}, sessionOwnerPassBinding{})
}

// WithSessionOwnerPass scopes only native artifact discovery, leaving other
// artifact controllers' existing seams unchanged.
func (c *Couch) WithSessionOwnerPass(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c != nil {
		if factory, ok := c.Artifacts.(interface {
			WithSessionOwnerPass(context.Context) (context.Context, context.CancelFunc)
		}); ok {
			return factory.WithSessionOwnerPass(ctx)
		}
	}
	return ctx, func() {}
}
