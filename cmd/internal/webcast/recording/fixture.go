package recording

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xianxu/pair/internal/reporoot"
)

// Fixtures are the checked-in recordings wave-1 components test against
// (atlas/webcast-format.md, Fixtures): every directory under
// testdata/fixtures with a pin file, sorted, as tests/webcast/fixtures.mjs
// finds them, so adding or removing one needs no list kept in step.
func Fixtures() []string {
	es, err := os.ReadDir(filepath.Dir(Fixture("x")))
	if err != nil {
		panic(err) // a test helper: the source tree is always there
	}
	var names []string
	for _, e := range es {
		if _, err := os.Stat(filepath.Join(Fixture(e.Name()), PinFile)); e.IsDir() && err == nil {
			names = append(names, e.Name())
		}
	}
	return names
}

// Fixture is the directory of a checked-in fixture, for tests in any
// package.
func Fixture(name string) string {
	root, err := reporoot.Current()
	if err != nil {
		panic(err) // a test helper: the checkout is required
	}
	return filepath.Join(root, "cmd", "internal", "webcast", "recording", "testdata", "fixtures", name)
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

// FrameTimes are the moments a frame at t may be compared at: t plus each
// offset that stays in the segment last started at or before t (a pause or
// the end keeps that segment's page, frozen). Another segment shows another
// page, and maybe another viewport, so comparing there could only pass
// falsely (#472).
func FrameTimes(recs []Record, t int64, offsets []int64) []int64 {
	seg := func(at int64) int {
		n := 0
		for _, r := range recs {
			if r.Time() > at {
				break
			}
			if x, ok := r.(*Seg); ok {
				n = x.Seg
			}
		}
		return n
	}
	own := seg(t)
	var ts []int64
	for _, off := range offsets {
		if own != 0 && seg(t+off) == own {
			ts = append(ts, t+off)
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
