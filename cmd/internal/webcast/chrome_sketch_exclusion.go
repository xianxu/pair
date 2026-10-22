package webcast

import "context"

// ExcludeSketchFrame joins the recorder's exclusion barrier before any private
// scene bytes can be installed into this frame.
func (c *Chrome) ExcludeSketchFrame(ctx context.Context, frameID, session string) error {
	if c.rec == nil {
		return ErrNotRecording
	}
	return c.rec.ExcludeSketchFrame(ctx, frameID, session)
}
func (c *Chrome) ExcludeSketchAssetURL(ctx context.Context, frameID, url string) error {
	if c.rec == nil {
		return ErrNotRecording
	}
	return c.rec.ExcludeSketchAssetURL(ctx, frameID, url)
}
