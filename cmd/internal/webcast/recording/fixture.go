package recording

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Fixtures are the checked-in recordings wave-1 components test against
// (atlas/webcast-format.md, Fixtures). The rec-* ones are the real
// recorder's (#472); stress, app and blog are #453's prototype's, and
// retire once the components built on them have landed.
var Fixtures = []string{"stress", "app", "blog", "segments", "driven", "rec-stress", "rec-app", "rec-blog", "rec-segments"}

// PrototypeFixtures are the fixtures #453's prototype recorded, whose
// polled observer reports scroll and input up to 150ms late.
var PrototypeFixtures = []string{"stress", "app", "blog"}

// Fixture is the directory of a checked-in fixture, for tests in any
// package.
func Fixture(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "fixtures", name)
}

// Pin is a recording's canonical state at one moment, by hash: what a
// fixture's canonical.sha256 holds, one "t hash" line each.
type Pin struct {
	T    int64
	Hash string
}

// PinFile is a fixture's pin file.
const PinFile = "canonical.sha256"

// CanonicalTimes are the moments a recording's canonical state is pinned
// at: each frame's (where the render can be compared too), or, without
// frames, each segment start, stop and note.
func CanonicalTimes(r *Recording) []int64 {
	var ts []int64
	for _, f := range r.Frames {
		ts = append(ts, f.T)
	}
	if len(ts) == 0 {
		for _, x := range r.Records {
			switch x.(type) {
			case *Seg, *Stop, *Note:
				ts = append(ts, x.Time())
			}
		}
	}
	return ts
}

// PinCanonical reduces a recording at each canonical time.
func PinCanonical(r *Recording) ([]Pin, error) {
	var pins []Pin
	for _, t := range CanonicalTimes(r) {
		st, err := Reduce(r.Records, t)
		if err != nil {
			return nil, err
		}
		pins = append(pins, Pin{T: t, Hash: HashOf(st.Canonical())})
	}
	return pins, nil
}

// FormatPins is the pin file's content.
func FormatPins(pins []Pin) []byte {
	var b bytes.Buffer
	for _, p := range pins {
		fmt.Fprintf(&b, "%d %s\n", p.T, p.Hash)
	}
	return b.Bytes()
}

// ReadPins reads a fixture's pin file.
func ReadPins(dir string) ([]Pin, error) {
	f, err := os.Open(filepath.Join(dir, PinFile))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var pins []Pin
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ts, h, ok := strings.Cut(sc.Text(), " ")
		t, err := strconv.ParseInt(ts, 10, 64)
		if !ok || err != nil || !ValidHash(h) {
			return nil, fmt.Errorf("%s: bad line %q", PinFile, sc.Text())
		}
		pins = append(pins, Pin{T: t, Hash: h})
	}
	return pins, sc.Err()
}
