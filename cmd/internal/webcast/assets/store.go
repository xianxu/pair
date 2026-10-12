// Package assets is couch broadcast-web's asset store and proxy (#459): the
// bodies of images, stylesheets and fonts a cast page loads, captured during
// recording, stored by content hash and served to viewers, who never reach
// the product's servers (project Principle 4).
//
// #457 defines the seam: Store, its stateful fake MemStore, and
// RunStoreContract, which the real store runs as well. The format of what a
// store produces (the asset record, assets/<hash>) is in
// atlas/webcast-format.md.
package assets

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Capture is one response body the recorder saw.
type Capture struct {
	URL  string // absolute
	MIME string
	Body []byte
	// Declared is the largest size the page shows the image at, in CSS px
	// times the device pixel ratio; zero when unknown. The resample budget
	// keeps at least this.
	DeclaredW, DeclaredH int
}

// Policy is the store's resolution budget. Images over it are resampled;
// CSS and fonts are always exact.
type Policy struct {
	// MaxPixels caps an image's stored pixel count (w×h), unless its
	// declared size needs more.
	MaxPixels int
}

// DefaultPolicy: about a 1920×1080 frame.
var DefaultPolicy = Policy{MaxPixels: 1920 * 1080}

// ErrNotFound: no body is stored under that hash.
var ErrNotFound = errors.New("assets: not found")

// Store keeps a recording's asset bodies. It is their one owner: the real
// store (#459) decides what to keep (dedup, resample) and persists each body
// through the recording's recording.AssetWriter (its DirWriter), the one
// writer of assets/<hash>. The recorder emits only the returned asset record,
// through its Sink.
type Store interface {
	// Put stores a captured body (resampled when it is an image over the
	// budget) and binds its URL to it. The returned record is what the
	// recorder emits (its t is the caller's to set). The same bytes twice
	// give the same hash; a URL put again is rebound.
	Put(ctx context.Context, c Capture) (*recording.Asset, error)
	// Get is the stored body under a hash, or ErrNotFound.
	Get(hash string) ([]byte, error)
	// Lookup is a URL's current binding.
	Lookup(url string) (*recording.Asset, bool)
}
