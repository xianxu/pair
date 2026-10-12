// Command fixturegen writes couch broadcast-web's recording fixtures (#457).
//
//	go run ./cmd/internal/webcast/recording/internal/fixturegen -recorded RUN -name NAME -out DIR
//	go run ./cmd/internal/webcast/recording/internal/fixturegen -segments -out DIR
//	go run ./cmd/internal/webcast/recording/internal/fixturegen -driven -out DIR
//	go run ./cmd/internal/webcast/recording/internal/fixturegen -canonical DIR -at T
//
// -recorded takes a run of the real recorder (#458), whose records need no
// conversion, and adds the fixture's notes and strokes from a small spec per
// fixture (notes.go, #472). -segments writes the hand-made fixture of
// pauses, resumes and tab switches. -canonical prints a fixture's canonical
// state at T (the viewer conformance check's expected value).
//
// Not product code. #453's prototype recorder's CDP-log converter (-raw)
// retired with its fixtures (#474).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func main() {
	recorded := flag.String("recorded", "", "recorder (#458) run directory")
	name := flag.String("name", "", "fixture name (selects the notes spec)")
	out := flag.String("out", "", "fixture directory to (re)write")
	segments := flag.Bool("segments", false, "write the hand-made segments fixture")
	driven := flag.Bool("driven", false, "write the hand-made driven fixture (recording.DrivenRecording)")
	canonical := flag.String("canonical", "", "print this fixture's canonical state at -at")
	at := flag.Int64("at", 0, "time for -canonical, ms")
	flag.Parse()
	var err error
	switch {
	case *canonical != "":
		err = printCanonical(*canonical, *at)
	case *segments && *out != "":
		err = write(*out, segmentsFixture(), nil, nil, nil)
	case *driven && *out != "":
		err = write(*out, rec.DrivenRecording(), nil, nil, nil)
	case *recorded != "" && *name != "" && *out != "":
		err = adopt(*recorded, *name, *out)
	default:
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "fixturegen:", err)
		os.Exit(1)
	}
}

func printCanonical(dir string, t int64) error {
	r, err := rec.ReadDir(dir)
	if err != nil {
		return err
	}
	st, err := rec.Reduce(r.Records, t)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(st.Canonical(), '\n'))
	return err
}

// adopt makes a fixture of a real recorder run: its records, assets and
// frames as recorded, plus the fixture's notes.
func adopt(runDir, name, out string) error {
	r, err := rec.ReadDir(runDir)
	if err != nil {
		return err
	}
	bodies := map[string][]byte{}
	for _, x := range r.Records {
		if a, ok := x.(*rec.Asset); ok && bodies[a.Hash] == nil {
			if bodies[a.Hash], err = r.Asset(a.Hash); err != nil {
				return err
			}
		}
	}
	var pngs [][]byte
	for _, f := range r.Frames {
		png, err := os.ReadFile(filepath.Join(runDir, f.File))
		if err != nil {
			return err
		}
		pngs = append(pngs, png)
	}
	notes, err := annotate(name, r.Records)
	if err != nil {
		return err
	}
	return write(out, mergeByTime(r.Records, notes), bodies, r.Frames, pngs)
}

// mergeByTime inserts extra records (notes, strokes) after every record at
// or before their time.
func mergeByTime(recs, extra []rec.Record) []rec.Record {
	out := append([]rec.Record(nil), recs...)
	out = append(out, extra...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time() < out[j].Time() })
	return out
}

// write (re)creates a fixture directory and validates it.
func write(out string, recs []rec.Record, bodies map[string][]byte, frames []*rec.Frame, pngs [][]byte) error {
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	w, err := rec.NewDirWriter(out)
	if err != nil {
		return err
	}
	used := map[string]bool{}
	for _, r := range recs {
		if a, ok := r.(*rec.Asset); ok && !used[a.Hash] {
			used[a.Hash] = true
			if _, err := w.PutAsset(bodies[a.Hash]); err != nil {
				return err
			}
		}
		if err := w.Put(r); err != nil {
			return err
		}
	}
	for i, f := range frames {
		if err := w.PutFrame(f, pngs[i]); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	fs, _, err := rec.ValidateDir(out)
	if err != nil {
		return err
	}
	if len(fs) > 0 {
		var msgs []string
		for _, f := range fs {
			msgs = append(msgs, f.String())
		}
		return fmt.Errorf("the written fixture doesn't validate:\n%s", strings.Join(msgs, "\n"))
	}
	return writeHashes(out)
}

// writeHashes pins the canonical state at each canonical time. The
// expected value itself is printed on demand (-canonical), so the reducer
// stays its only source.
func writeHashes(dir string) error {
	r, err := rec.ReadDir(dir)
	if err != nil {
		return err
	}
	pins, err := rec.PinCanonical(r)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, rec.PinFile), rec.FormatPins(pins), 0o600)
}
