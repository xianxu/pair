package launcher

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// ListingEvidence describes the exact bytes parsed, not a later diagnostic read.
// Raw is a copied bounded prefix; the hash and count describe the whole input.
type ListingEvidence struct {
	At        time.Time
	Bytes     int
	SHA256    string
	Raw       []byte
	Truncated bool
}

func newListingEvidence(raw []byte, at time.Time) ListingEvidence {
	return ListingEvidence{
		At: at, Bytes: len(raw), SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)),
		Raw: append([]byte(nil), raw[:min(len(raw), 8192)]...), Truncated: len(raw) > 8192,
	}
}
