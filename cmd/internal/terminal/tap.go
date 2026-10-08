package terminal

import "context"

// FrameClass says whether a presented frame may leave the operator's screen.
// Private frames (Couch's switcher, which lists the whole fleet) are replaced
// before they reach any viewer (#395).
type FrameClass uint8

const (
	FramePublic FrameClass = iota
	FramePrivate
)

// Tap observes each frame after it has been fully painted to the parent and
// admitted as presented. It runs on the Presenter goroutine: it must not block,
// and it receives an owned clone.
type Tap func(Frame, FrameClass)

// SetTap installs, or with nil removes, the tap, ordered with paints.
func (p *Presenter) SetTap(ctx context.Context, t Tap) error {
	return p.call(ctx, func(context.Context) error { p.tap = t; return nil })
}
