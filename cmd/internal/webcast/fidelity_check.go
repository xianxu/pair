package webcast

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/xianxu/pair/cmd/internal/webcast/fidelity"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// ReplayWindow is how much of the timeline before a frame the fidelity
// replay plays in real time (ms), so a transition running at the frame is
// at the tab's phase (their timing isn't recorded). Tailwind's longest
// common duration is 1s.
const ReplayWindow = 1000

// CheckFidelity is the fidelity check (#478, project Principle 6): it
// replays the recording at dir in a headless Chrome with the viewer's
// renderer at each frame's time, takes the screenshot the recorder took
// of the tab (recorder.LowRes of the viewport), scores the two with
// fidelity.Diff, and writes the report to <dir>/fidelity/. It reads the
// recording through a ReplayServer on loopback and needs Chrome at
// chromeBin; the Chrome and its profile are gone when it returns.
func CheckFidelity(ctx context.Context, chromeBin, dir string) (fidelity.Report, error) {
	abs, err := filepath.Abs(dir)
	if err == nil {
		abs, err = filepath.EvalSymlinks(abs)
	}
	if err != nil {
		return fidelity.Report{}, err
	}
	root, id := filepath.Dir(abs), filepath.Base(abs)
	if !ValidRecordingID(id) {
		return fidelity.Report{}, fmt.Errorf("fidelity: %q is not a recording directory's name", id)
	}
	rec, err := recording.ReadDir(abs)
	if err != nil {
		return fidelity.Report{}, err
	}
	frames := append([]*recording.Frame(nil), rec.Frames...)
	sort.SliceStable(frames, func(i, j int) bool { return frames[i].T < frames[j].T })
	if len(frames) == 0 {
		return fidelity.Report{}, errors.New("fidelity: the recording has no frames")
	}

	base, stop, err := serveReplay(root)
	if err != nil {
		return fidelity.Report{}, err
	}
	defer stop()
	b, err := launchHeadless(ctx, chromeBin)
	if err != nil {
		return fidelity.Report{}, err
	}
	defer b.close()
	if err := b.open(ctx, base+"/r/"+id+"/fidelity.html"); err != nil {
		return fidelity.Report{}, err
	}

	var shots []fidelity.Shot
	skipped := 0
	replays := map[string][]byte{} // low-res replays of the worst shots so far, by frame file
	for _, f := range frames {
		if _, err := frameViewport(f); err != nil {
			return fidelity.Report{}, err
		}
		tab, err := os.ReadFile(filepath.Join(abs, f.File))
		if err != nil {
			return fidelity.Report{}, err
		}
		// The moments to compare at, kept in the frame's own segment (#472's
		// one rule): t first.
		times := recording.FrameTimes(rec.Records, f.T, fidelity.Offsets)
		shot, got, live, err := b.score(ctx, f, tab, times)
		if err != nil {
			return fidelity.Report{}, fmt.Errorf("fidelity: %s at t=%d: %w", f.File, f.T, err)
		}
		if !live {
			skipped++
			continue
		}
		shots = append(shots, shot)
		replays[f.File] = got
		keepWorst(replays, shots)
	}
	rep := fidelity.Summarize(id, shots, skipped)
	crops := make([][]byte, len(rep.Worst))
	for n, i := range rep.Worst {
		s := rep.Shots[i]
		tab, err := os.ReadFile(filepath.Join(abs, s.File))
		if err != nil {
			return rep, err
		}
		if crops[n], err = sideBySide(tab, replays[s.File]); err != nil {
			return rep, fmt.Errorf("fidelity: %s: %w", s.File, err)
		}
	}
	return rep, fidelity.Write(abs, rep, crops)
}

// keepWorst drops the replays no report can show: only the MaxWorst
// highest shots' are kept, so memory stays bounded however long the cast.
func keepWorst(replays map[string][]byte, shots []fidelity.Shot) {
	if len(replays) <= fidelity.MaxWorst {
		return
	}
	order := make([]fidelity.Shot, 0, len(replays))
	for _, s := range shots {
		if _, ok := replays[s.File]; ok {
			order = append(order, s)
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].Diff > order[j].Diff })
	for _, s := range order[fidelity.MaxWorst:] {
		delete(replays, s.File)
	}
}

// sideBySide is a worst moment's image: cropped to where the two differ,
// or whole when they are the same size and nothing differs.
func sideBySide(tabPNG, replayPNG []byte) ([]byte, error) {
	tab, err := png.Decode(bytes.NewReader(tabPNG))
	if err != nil {
		return nil, err
	}
	replay, err := png.Decode(bytes.NewReader(replayPNG))
	if err != nil {
		return nil, err
	}
	box := tab.Bounds().Sub(tab.Bounds().Min)
	if replay.Bounds().Size() == tab.Bounds().Size() {
		if b, ok := fidelity.DiffBox(tab, replay); ok {
			box = b
		}
	} else {
		box = box.Intersect(replay.Bounds().Sub(replay.Bounds().Min))
	}
	var out bytes.Buffer
	if err := png.Encode(&out, fidelity.SideBySide(tab, replay, box)); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// score replays f's moments (times: t first, then the other offsets, as
// recording.FrameTimes keeps them) and diffs each with the tab's frame,
// stopping at the first within the tolerance and keeping the best. It
// returns the shot and its replay; live is false when nothing live was
// drawn at t.
func (h *headless) score(ctx context.Context, f *recording.Frame, tabPNG []byte, times []int64) (fidelity.Shot, []byte, bool, error) {
	shot := fidelity.Shot{File: f.File, T: f.T}
	if len(times) == 0 || times[0] != f.T {
		return shot, nil, false, nil // before any segment
	}
	tab, err := png.Decode(bytes.NewReader(tabPNG))
	if err != nil {
		return shot, nil, false, fmt.Errorf("the tab's frame: %w", err)
	}
	var best []byte
	for _, at := range times {
		off := at - f.T
		got, live, err := h.replayFrame(ctx, f, at)
		if err != nil {
			return shot, nil, false, err
		}
		if !live {
			if off == 0 {
				return shot, nil, false, nil
			}
			continue
		}
		img, err := png.Decode(bytes.NewReader(got))
		if err != nil {
			return shot, nil, false, err
		}
		d, err := fidelity.DiffImages(tab, img)
		if err != nil {
			d = 1 // a replay at another size differs everywhere
		}
		d = math.Round(d*1e4) / 100
		if off == 0 {
			shot.AtT = d
		}
		if best == nil || d < shot.Diff {
			shot.Diff, shot.Offset, best = d, off, got
		}
		if shot.Diff <= fidelity.Tolerance {
			break
		}
	}
	if shot.Offset == 0 {
		shot.AtT = 0 // the same as Diff
	}
	return shot, best, true, nil
}

// maxViewport bounds a frame's viewport (CSS px per side): a frame record
// is read from disk, and the replay lays a page out at its size.
const maxViewport = 16384

// frameScale is f's image px per CSS px; the fixtures' frames carry none
// and are at the recorder's 0.5.
func frameScale(f *recording.Frame) float64 {
	if f.Scale <= 0 {
		return 0.5
	}
	return f.Scale
}

// frameViewport is the viewport (CSS px) f was taken at, which the replay
// lays out at. A frame naming a file outside frames/, or an implausible
// size, is refused: frame records come from disk.
func frameViewport(f *recording.Frame) (image.Point, error) {
	if !filepath.IsLocal(f.File) || filepath.Dir(filepath.Clean(f.File)) != "frames" {
		return image.Point{}, fmt.Errorf("fidelity: frame file %q is not in frames/", f.File)
	}
	scale := frameScale(f)
	vp := image.Pt(int(math.Round(float64(f.W)/scale)), int(math.Round(float64(f.H)/scale)))
	if f.W <= 0 || f.H <= 0 || scale > 1 || vp.X > maxViewport || vp.Y > maxViewport {
		return image.Point{}, fmt.Errorf("fidelity: frame %s is %dx%d at scale %g, not a plausible viewport", f.File, f.W, f.H, f.Scale)
	}
	return vp, nil
}

// replayFrame draws the recording as the tab was at at (f's time, or
// near it) and screenshots it
// as the recorder screenshots the tab: the viewport, at f's scale. live is
// false when nothing live was drawn there.
func (h *headless) replayFrame(ctx context.Context, f *recording.Frame, at int64) (shot []byte, live bool, err error) {
	vp, err := frameViewport(f)
	if err != nil {
		return nil, false, err
	}
	if vp != h.vp {
		if err := h.call(ctx, h.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": vp.X, "height": vp.Y, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
			return nil, false, err
		}
		h.vp = vp
	}
	var drawn struct {
		Live bool `json:"live"`
	}
	if err := h.eval(ctx, fmt.Sprintf("window.fidelity.replayAt(%d, %d)", at, ReplayWindow), &drawn); err != nil {
		return nil, false, err
	}
	if !drawn.Live {
		return nil, false, nil
	}
	var res struct {
		Data string `json:"data"`
	}
	if err := h.call(ctx, h.session, "Page.captureScreenshot", map[string]any{"format": "png"}, &res); err != nil {
		return nil, false, err
	}
	full, err := base64.StdEncoding.DecodeString(res.Data)
	if err != nil {
		return nil, false, err
	}
	// LowRes halves per CSS px at a DPR; a DPR-1 replay scaled to the
	// frame's scale s is LowRes at 1/(2s).
	low, _, _, err := recorder.LowRes(full, 1/(2*frameScale(f)))
	return low, true, err
}
