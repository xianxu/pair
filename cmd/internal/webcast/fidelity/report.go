package fidelity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// Tolerance is the share of a frame's pixels, in %, the replay may differ
// by. Desktop Chromium replaying in desktop Chromium targets exact
// fidelity; what is left is anti-aliasing at scaled edges and a transition
// caught at another phase. Measured with Diff (#478): the fixtures' frames
// at most 0.74% (stress), a live blog cast by #458's recorder at most
// 0.10% over 17 frames, and the blog without its stylesheets 81% median.
const Tolerance = 3.0

// CaptureSkew is how much earlier than its t a frame's pixels can be (ms):
// a frame's t is its screenshot call's midpoint, but the pixels are a
// compositor frame from inside the call, 40–90 ms early on a busy page.
// Measured on rec-app frame 5: 77% off at t, 0.4% at t-50 (#472).
const CaptureSkew = 50

// Offsets are the moments (ms from a frame's t) a frame is compared at, in
// order until one is within the tolerance; the best is kept. #458's recorder
// reports synchronously, so its records never trail the tab: a frame is
// tried only at t and CaptureSkew before it. A later moment would hide a
// recorder that is late (pair#475's stylesheet bound after the screenshot
// that shows it). recording.FrameTimes drops a moment outside the frame's
// own segment, and the report shows a match's offset.
var Offsets = []int64{0, -CaptureSkew}

// MaxWorst bounds the moments a report shows side by side.
const MaxWorst = 10

// minWorst moments are shown even when every frame is within tolerance.
const minWorst = 3

// tmpPrefix names a report being written, beside the recording's files.
const tmpPrefix = ".fidelity-"

// staleWrite is the age past which a report's temporary directory is a
// crashed write's, not one in progress.
const staleWrite = time.Minute

// ReportDir is where a recording's report goes, beside its frames/.
const ReportDir = "fidelity"

// Shot is one frame's result. Diff is the % of pixels that differ; Crop
// names its side-by-side image when it is one of the report's worst.
type Shot struct {
	File string  `json:"file"`
	T    int64   `json:"t"`
	Diff float64 `json:"diff"`
	// Offset is when the replay matched the frame best, in ms from its
	// t: a frame over the tolerance at t is retried at Offsets.
	Offset int64 `json:"offset,omitempty"`
	// AtT is the diff at t itself when the best was at an offset.
	AtT  float64 `json:"at_t,omitempty"`
	Over bool    `json:"over,omitempty"`
	Crop string  `json:"crop,omitempty"`
}

// Report is a recording's fidelity. Shots are the frames compared, in time
// order; Skipped counts frames with nothing live to replay at their time.
// Worst indexes Shots, worst first: every shot over the tolerance and at
// least minWorst, at most MaxWorst.
type Report struct {
	Recording string  `json:"recording"`
	Tolerance float64 `json:"tolerance"`
	Frames    int     `json:"frames"`
	Shots     []Shot  `json:"shots"`
	Skipped   int     `json:"skipped"`
	Median    float64 `json:"median"`
	Max       float64 `json:"max"`
	Over      int     `json:"over"`
	Worst     []int   `json:"worst"`
}

// Summarize scores the compared shots of recording id.
func Summarize(id string, shots []Shot, skipped int) Report {
	shots = slices.Clone(shots) // the report's own: Over and Crop are set here
	rep := Report{Recording: id, Tolerance: Tolerance, Frames: len(shots) + skipped, Shots: shots, Skipped: skipped, Worst: []int{}}
	if len(shots) == 0 {
		return rep
	}
	order := make([]int, len(shots))
	for i := range shots {
		order[i] = i
		shots[i].Over = shots[i].Diff > Tolerance
		if shots[i].Over {
			rep.Over++
		}
	}
	sort.SliceStable(order, func(a, b int) bool { return shots[order[a]].Diff > shots[order[b]].Diff })
	rep.Max = shots[order[0]].Diff
	rep.Median = shots[order[(len(order)-1)/2]].Diff // the upper middle of an even count
	rep.Worst = order[:min(max(rep.Over, minWorst), MaxWorst, len(order))]
	for n, i := range rep.Worst {
		shots[i].Crop = fmt.Sprintf("%s/worst-%d.png", ReportDir, n+1)
	}
	return rep
}

// Within says whether some frame was compared and none is over the
// tolerance.
func (r Report) Within() bool { return len(r.Shots) > 0 && r.Over == 0 }

// Summary is the report in one line.
func (r Report) Summary() string {
	if len(r.Shots) == 0 {
		return fmt.Sprintf("no frame to compare (%d without a live replay)", r.Skipped)
	}
	return fmt.Sprintf("median %.2f%% max %.2f%%, %d of %d frames over %g%%", r.Median, r.Max, r.Over, len(r.Shots), r.Tolerance)
}

// Markdown is report.md: the summary, the moments over the tolerance and
// the worst ones with their side-by-side images, and every frame's number.
func (r Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Fidelity of %s\n\n", r.Recording)
	fmt.Fprintf(&b, "%s (tolerance %g%% of pixels per frame).\n", r.Summary(), r.Tolerance)
	switch {
	case r.Skipped == 1:
		b.WriteString("1 frame had nothing live to replay.\n")
	case r.Skipped > 1:
		fmt.Fprintf(&b, "%d frames had nothing live to replay.\n", r.Skipped)
	}
	b.WriteString("\nEach image is the real tab, the replay, and where they differ (red), cropped to the difference.\n\n")
	b.WriteString("A frame over the tolerance is replayed again a little earlier (a screenshot's pixels can predate its t); *matched at* is the best one's offset.\n\n")
	b.WriteString("## Worst moments\n\n| t (ms) | diff | over | matched at | tab frame | side by side |\n|---|---|---|---|---|---|\n")
	for _, i := range r.Worst {
		s := r.Shots[i]
		fmt.Fprintf(&b, "| t=%d | %.2f%% | %s | %s | %s | %s |\n", s.T, s.Diff, yes(s.Over), matched(s), s.File, strings.TrimPrefix(s.Crop, ReportDir+"/"))
	}
	b.WriteString("\n## Every frame\n\n| t (ms) | diff | matched at |\n|---|---|---|\n")
	for _, s := range r.Shots {
		fmt.Fprintf(&b, "| %d | %.2f%% | %s |\n", s.T, s.Diff, matched(s))
	}
	return b.String()
}

func matched(s Shot) string {
	if s.Offset == 0 {
		return "t"
	}
	return fmt.Sprintf("t%+d ms (%.2f%% at t)", s.Offset, s.AtT)
}

func yes(b bool) string {
	if b {
		return "yes"
	}
	return ""
}

// Write puts the report in <dir>/fidelity/: report.json, report.md and
// worst-<n>.png (crops[n-1], one per Worst). It replaces a previous
// report whole, so a rerun leaves no stale crop.
func Write(dir string, r Report, crops [][]byte) error {
	if len(crops) != len(r.Worst) {
		return fmt.Errorf("fidelity: %d crops for %d worst moments", len(crops), len(r.Worst))
	}
	// A write a crash cut short leaves its temporary directory; the next
	// one sweeps it. A write takes milliseconds, so one older than
	// staleWrite isn't another check's write in progress.
	stale, _ := filepath.Glob(filepath.Join(dir, tmpPrefix+"*"))
	for _, d := range stale {
		if info, err := os.Stat(d); err == nil && time.Since(info.ModTime()) > staleWrite {
			os.RemoveAll(d)
		}
	}
	tmp, err := os.MkdirTemp(dir, tmpPrefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	js, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	files := map[string][]byte{"report.json": append(js, '\n'), "report.md": []byte(r.Markdown())}
	for n, c := range crops {
		files[fmt.Sprintf("worst-%d.png", n+1)] = c
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), body, 0o600); err != nil {
			return err
		}
	}
	if err := os.Chmod(tmp, 0o700); err != nil {
		return err
	}
	out := filepath.Join(dir, ReportDir)
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	return os.Rename(tmp, out)
}
