// Command fixturegen writes couch broadcast-web's recording fixtures (#457).
//
//	go run ./cmd/internal/webcast/recording/internal/fixturegen -raw RUN -name NAME -out DIR
//	go run ./cmd/internal/webcast/recording/internal/fixturegen -recorded RUN -name NAME -out DIR
//	go run ./cmd/internal/webcast/recording/internal/fixturegen -segments -out DIR
//	go run ./cmd/internal/webcast/recording/internal/fixturegen -driven -out DIR
//	go run ./cmd/internal/webcast/recording/internal/fixturegen -canonical DIR -at T
//
// RUN is a run directory of #453's prototype recorder (branch
// 453-recorder-prototypes, mode B, extended for #457; see the fixtures'
// README). The converter maps its CDP log to records; notes and strokes come
// from a small spec per fixture (notes.go). -recorded takes a run of the
// real recorder (#458) instead, whose records need no conversion, and adds
// the notes (#472). -segments writes the hand-made
// fixture of pauses, resumes and tab switches. -canonical prints a fixture's
// canonical state at T (the viewer conformance check's expected value).
//
// Not product code: the recorder is pair#458. The converter is a reference
// for its CDP-to-record mapping.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func main() {
	raw := flag.String("raw", "", "prototype run directory")
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
	case *raw != "" && *name != "" && *out != "":
		err = convert(*raw, *name, *out)
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

type shot struct {
	I int   `json:"i"`
	T int64 `json:"t"`
}

func convert(runDir, name, out string) error {
	f, err := os.Open(filepath.Join(runDir, "cdp.jsonl"))
	if err != nil {
		return err
	}
	defer f.Close()
	var lines []raw
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<20), 64<<20)
	for n := 1; sc.Scan(); n++ {
		var r raw
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return fmt.Errorf("line %d: %w", n, err)
		}
		r.Line, r.Extra = n, append([]byte(nil), sc.Bytes()...)
		lines = append(lines, r)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if len(lines) == 0 {
		return errors.New("empty run")
	}
	t0 := lines[0].T
	c := newConverter(name, t0)
	c.exclude = excluded[name]
	skipped := 0
	for _, r := range lines {
		if err := c.event(r); errors.Is(err, errSkip) {
			skipped++
		} else if err != nil {
			return fmt.Errorf("line %d (%s): %w", r.Line, r.Type, err)
		}
	}
	end := lines[len(lines)-1].T
	c.emit(end, &rec.Stop{Why: rec.StopEnd})

	var meta struct {
		Shots []shot `json:"shots"`
	}
	mb, err := os.ReadFile(filepath.Join(runDir, "meta.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(mb, &meta); err != nil {
		return err
	}
	var frames []*rec.Frame
	var pngs [][]byte
	for _, s := range meta.Shots {
		png, err := os.ReadFile(filepath.Join(runDir, "shots", strconv.Itoa(s.I)+".png"))
		if err != nil {
			return err
		}
		fr := &rec.Frame{File: fmt.Sprintf("frames/%d.png", s.I), W: c.vp.W / 2, H: c.vp.H / 2, Scale: 0.5}
		fr.SetTime(s.T - t0)
		frames, pngs = append(frames, fr), append(pngs, png)
	}

	fmt.Fprintf(os.Stderr, "%s: %d raw lines, %d records, %d skipped (mirror gaps), dropped: %s\n",
		name, len(lines), len(c.out), skipped, c.summary())
	for _, e := range c.examples {
		fmt.Fprintln(os.Stderr, "  e.g.", e)
	}
	recs := c.out
	notes, err := annotate(name, recs)
	if err != nil {
		return err
	}
	recs = mergeByTime(recs, notes)
	return write(out, recs, c.bodies, frames, pngs)
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
