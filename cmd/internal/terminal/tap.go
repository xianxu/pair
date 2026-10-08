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

// Overlay draws on top of each composed frame before it is painted, for
// presenter-owned decoration such as a broadcast's remote-pointer marks
// (#412). It runs on the Presenter goroutine: it must not block, it must not
// modify f in place, and any lock it takes must be a leaf. It returns f
// itself when it has nothing to draw. History rows come from the endpoint,
// not the frame, so an overlay never reaches the parent's scrollback.
type Overlay func(f Frame, class FrameClass) Frame

// SetOverlay installs, or with nil removes, the overlay, ordered with paints.
func (p *Presenter) SetOverlay(ctx context.Context, o Overlay) error {
	return p.call(ctx, func(context.Context) error { p.overlay = o; return nil })
}

// Refresh repaints what is on screen, endpoint or panel, through the same
// path as any paint, so an overlay that changed with no new output (a mark
// fading) reaches the screen.
func (p *Presenter) Refresh(ctx context.Context) error {
	return p.call(ctx, func(ctx context.Context) error {
		switch {
		case p.selected != nil:
			return p.paintEndpoint(ctx, p.selected, false)
		case p.panel.Cells != nil:
			return p.paint(ctx, p.panel, true)
		}
		return nil
	})
}

// SetTap installs, or with nil removes, the tap, ordered with paints.
func (p *Presenter) SetTap(ctx context.Context, t Tap) error {
	return p.call(ctx, func(context.Context) error { p.tap = t; return nil })
}
