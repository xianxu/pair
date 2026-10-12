package assets

import (
	"errors"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// ErrReadOnly: a reopened recording's store takes no new bodies.
var ErrReadOnly = errors.New("assets: a reopened recording is read-only")

// Reopen is the store of a finished recording, for its replay (#471): the
// bindings and types its asset records state, bodies read from its
// directory. It is a Recorded, so the Proxy serves a replay exactly as it
// serves the cast: same types allowed, same CSS rewrite. Put refuses.
func Reopen(rec *recording.Recording) *Recorded {
	s := &Recorded{read: rec.Asset, types: map[string]string{}, urls: map[string]*recording.Asset{}, done: map[string]*recording.Asset{}}
	for _, r := range rec.Records {
		if a, ok := r.(*recording.Asset); ok {
			s.types[a.Hash] = a.MIME
			s.urls[a.URL] = clone(a)
		}
	}
	return s
}
