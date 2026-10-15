package recording

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xianxu/pair/internal/reporoot"
)

func TestFixtureFromDifferentPackageDirectories(t *testing.T) {
	root, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "cmd/internal/webcast/recording/testdata/fixtures/segments")
	for _, rel := range []string{".", "cmd/internal/webcast", "cmd/internal/webcast/recording"} {
		t.Run(rel, func(t *testing.T) {
			t.Chdir(filepath.Join(root, rel))
			if got := Fixture("segments"); got != want {
				t.Fatalf("fixture = %q; want %q", got, want)
			}
			if _, err := os.Stat(filepath.Join(Fixture("segments"), PinFile)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

var updatePins = flag.Bool("update-pins", false, "rewrite the fixtures' canonical.sha256 (after a deliberate reducer change)")

func TestFixturesValidateAndReduce(t *testing.T) {
	for _, name := range Fixtures() {
		t.Run(name, func(t *testing.T) {
			dir := Fixture(name)
			fs, _, err := ValidateDir(dir)
			if err != nil || len(fs) != 0 {
				t.Fatalf("%v %v", err, fs)
			}
			r, err := ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			pins, err := PinCanonical(r)
			if err != nil {
				t.Fatal(err)
			}
			if *updatePins {
				os.WriteFile(filepath.Join(dir, PinFile), FormatPins(pins), 0o644)
			}
			want, err := ReadPins(dir)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(FormatPins(pins), FormatPins(want)) {
				t.Fatalf("canonical states moved from %s (-update-pins after a deliberate change)", PinFile)
			}
			if len(pins) == 0 {
				t.Fatal("a fixture needs canonical times")
			}
			// Joining at any pinned moment gives the same page and notes.
			for _, p := range pins {
				if err := CheckCatchUp(r.Records, p.T); err != nil {
					t.Fatalf("catch-up at %d: %v", p.T, err)
				}
			}
		})
	}
}

// TestFixtureCoverage holds each fixture to what the atlas says it covers,
// so a regenerated fixture can't silently lose a case.
func TestFixtureCoverage(t *testing.T) {
	type check struct {
		what string
		ok   func(c *coverage) bool
	}
	want := map[string][]check{
		// #453's stress page from testdata/stress, which leaves out the late
		// FontFace font (a macOS system font), so the page never loads one.
		"rec-stress": {
			{"a cross-origin iframe document", func(c *coverage) bool { return c.frameDocs > 0 }},
			{"a closed shadow root", func(c *coverage) bool { return c.closed > 0 }},
			{"an open shadow root", func(c *coverage) bool { return c.open > 0 }},
			{"insertRule (an owned sheet)", func(c *coverage) bool { return c.ownedSheets > 0 }},
			{"adoptedStyleSheets", func(c *coverage) bool { return c.kinds["adopt"] > 0 }},
			{"checkbox and select state", func(c *coverage) bool { return c.checked > 0 && c.selected > 0 }},
			{"inner scroll", func(c *coverage) bool { return c.elementScroll > 0 }},
			{"svg", func(c *coverage) bool { return c.svg > 0 }},
			{"template content", func(c *coverage) bool { return c.templates > 0 }},
			{"busy re-renders", func(c *coverage) bool { return c.kinds["kids"] > 100 && c.kinds["ins"] > 1000 }},
		},
		"rec-app": {
			{"typing", func(c *coverage) bool { return c.kinds["input"] > 0 }},
			{"clicks and focus", func(c *coverage) bool { return c.kinds["click"] > 0 && c.kinds["focus"] > 0 }},
			{"a text node over CDP's 10k cut", func(c *coverage) bool { return c.longText }},
			{"a cross-document navigation", func(c *coverage) bool { return c.navigateSegs > 0 }},
			{"a resolved note", func(c *coverage) bool { return c.kinds["note-state"] > 0 }},
		},
		"rec-blog": {
			{"images", func(c *coverage) bool { return c.mimes["image/png"] > 0 }},
			{"a web font", func(c *coverage) bool { return c.mimes["font/woff2"] > 0 }},
			{"stylesheets", func(c *coverage) bool { return c.mimes["text/css"] > 0 }},
			{"page scrolling", func(c *coverage) bool { return c.pageScroll > 0 }},
			{"layout", func(c *coverage) bool { return c.kinds["layout"] > 0 }},
		},
		"segments": {
			{"a pause and a resume", func(c *coverage) bool { return c.kinds["stop"] > 1 && c.resumeSegs > 0 }},
			{"a tab switch", func(c *coverage) bool { return c.tabSegs > 0 }},
			{"a same-document navigation", func(c *coverage) bool { return c.kinds["nav"] > 0 }},
		},
		"rec-segments": {
			{"a pause and a resume", func(c *coverage) bool { return c.kinds["stop"] > 1 && c.resumeSegs > 0 }},
			{"a tab switch", func(c *coverage) bool { return c.tabSegs > 0 }},
			{"a same-document navigation", func(c *coverage) bool { return c.kinds["nav"] > 0 }},
			{"a cross-document navigation", func(c *coverage) bool { return c.navigateSegs > 0 }},
		},
		"rec-driven": {
			{"grant, release, revoke, deny and leave", func(c *coverage) bool {
				return c.ctl[CtlGrant] > 0 && c.ctl[CtlRelease] > 0 && c.ctl[CtlRevoke] > 0 && c.ctl[CtlDeny] > 0 && c.ctl[CtlLeave] > 0
			}},
			{"a driver's click and typing", func(c *coverage) bool { return c.by["click"] > 0 && c.by["input"] > 0 }},
			{"the operator acting while a viewer drives", func(c *coverage) bool { return c.kinds["click"] > c.by["click"] }},
			{"a navigation while driving", func(c *coverage) bool { return c.navigateDriven > 0 }},
			{"a pause and resume while driving", func(c *coverage) bool { return c.resumeSegs > 0 }},
			{"a note by the driver", func(c *coverage) bool { return c.driverNotes > 0 }},
		},
	}
	for name := range want {
		if !slices.Contains(Fixtures(), name) {
			t.Errorf("%s: coverage checks for no fixture", name)
		}
	}
	for _, name := range Fixtures() {
		if want[name] == nil {
			t.Errorf("%s: no coverage checks", name)
		}
		r, err := ReadDir(Fixture(name))
		if err != nil {
			t.Fatal(err)
		}
		c := measure(r)
		if c.kinds["note"] == 0 {
			t.Errorf("%s: no notes", name)
		}
		for _, ch := range want[name] {
			if !ch.ok(c) {
				t.Errorf("%s: doesn't cover %s", name, ch.what)
			}
		}
	}
}

func TestFixturesStayWithinTheirBudget(t *testing.T) {
	var total int64
	filepath.Walk(filepath.Dir(Fixture("x")), func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	if budget := int64(4_500_000); total > budget {
		t.Fatalf("fixtures are %d bytes, over the %d budget; trim a run before adding to it", total, budget)
	}
}

type coverage struct {
	kinds                                                map[string]int
	mimes                                                map[string]int
	frameDocs, closed, open, ownedSheets, svg, templates int
	checked, selected, elementScroll, pageScroll         int
	navigateSegs, resumeSegs, tabSegs                    int
	ctl, by                                              map[string]int // ctl ops; attributed records by kind
	navigateDriven, driverNotes                          int
	longText                                             bool
}

func measure(r *Recording) *coverage {
	c := &coverage{kinds: map[string]int{}, mimes: map[string]int{}, ctl: map[string]int{}, by: map[string]int{}}
	driving, driver := false, ""
	docs := map[NodeID]bool{}
	var walk func(n *Node)
	walk = func(n *Node) {
		switch {
		case n.Y == YShadow && n.Mode == "closed":
			c.closed++
		case n.Y == YShadow:
			c.open++
		case n.NS == "svg":
			c.svg++
		case n.TC != nil:
			c.templates++
		case n.Y == YText && len(*n.V) > 10000:
			c.longText = true
		}
		for _, sub := range []*Node{n.SR, n.TC} {
			if sub != nil {
				walk(sub)
			}
		}
		for i := range n.C {
			walk(&n.C[i])
		}
	}
	for _, x := range r.Records {
		c.kinds[x.Kind()]++
		if a, ok := x.(attributed); ok && a.source() != "" {
			c.by[x.Kind()]++
		}
		switch x := x.(type) {
		case *Doc:
			docs[x.Doc] = true
			if x.Host != 0 {
				c.frameDocs++
			}
			walk(&x.Tree)
		case *Ins:
			walk(&x.Node)
		case *Kids:
			for i := range x.C {
				walk(&x.C[i])
			}
		case *Shadow:
			walk(&x.Root)
		case *Text:
			c.longText = c.longText || len(x.V) > 10000
		case *Sheet:
			if x.Owner != 0 {
				c.ownedSheets++
			}
		case *Input:
			if x.Checked != nil {
				c.checked++
			}
			if x.Sel != nil {
				c.selected++
			}
		case *Scroll:
			if docs[x.ID] {
				c.pageScroll++
			} else {
				c.elementScroll++
			}
		case *Asset:
			c.mimes[x.MIME]++
		case *Note:
			if driving && x.Author.ID == driver {
				c.driverNotes++
			}
		case *Ctl:
			c.ctl[x.Op]++
			driving, driver = x.Driver != nil, x.Driving()
		case *Seg:
			switch x.Why {
			case SegNavigate:
				c.navigateSegs++
				if driving {
					c.navigateDriven++
				}
			case SegResume:
				c.resumeSegs++
			case SegTab:
				c.tabSegs++
			}
		}
	}
	return c
}

// An offset from a frame's time never leaves its segment: not into the
// previous page, nor across a resume. A pause keeps its segment's page.
func TestFrameTimesStayInTheFramesSegment(t *testing.T) {
	b := NewBuilder("r")
	b.Page(SegLive, "http://a/", b.El("p", nil))
	b.At(1000).Add(&Stop{Why: StopPause})
	b.At(1500)
	b.Page(SegResume, "http://a/", b.El("p", nil))
	b.At(3000)
	b.Page(SegNavigate, "http://b/", b.El("p", nil))
	b.At(4000).Add(&Stop{Why: StopEnd})
	offs := []int64{0, -50, 150}
	for _, c := range []struct {
		t    int64
		want []int64
	}{
		{500, []int64{500, 450, 650}},
		{960, []int64{960, 910, 1110}},    // into the pause: the same page, frozen
		{1400, []int64{1400, 1350}},       // 1550 is after the resume
		{1500, []int64{1500, 1650}},       // 1450 is before it
		{3020, []int64{3020, 3170}},       // 2970 is the previous page
		{4100, []int64{4100, 4050, 4250}}, // after the end: the last page
	} {
		if got := FrameTimes(b.Recs, c.t, offs); !slices.Equal(got, c.want) {
			t.Errorf("t=%d: %v, want %v", c.t, got, c.want)
		}
	}
	if got := FrameTimes(nil, 100, offs); got != nil {
		t.Errorf("before any segment: %v, want none", got)
	}
}
