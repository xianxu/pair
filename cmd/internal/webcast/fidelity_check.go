package webcast

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/cdp"
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
		tab, err := os.ReadFile(filepath.Join(abs, f.File))
		if err != nil {
			return fidelity.Report{}, err
		}
		shot, got, live, err := b.score(ctx, f, tab)
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

// serveReplay serves the recordings under root on loopback under a fresh
// token, returning the server's base URL and how to stop it.
func serveReplay(root string) (string, func(), error) {
	token, err := broadcast.NewToken()
	if err != nil {
		return "", nil, err
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	srv := &http.Server{Handler: NewReplayServer(token, root), ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(l)
	return fmt.Sprintf("http://%s/%s", l.Addr(), token), func() { srv.Close() }, nil
}

// headless is a headless Chrome on its debugging pipe with one page, on a
// throwaway profile.
type headless struct {
	cmd     *exec.Cmd
	conn    *cdp.Conn
	session string
	profile string
	vp      image.Point // the page's current device metrics
}

func launchHeadless(ctx context.Context, bin string) (*headless, error) {
	profile, err := os.MkdirTemp("", "couch-fidelity-")
	if err != nil {
		return nil, err
	}
	cmdR, cmdW, err := os.Pipe()
	if err != nil {
		os.RemoveAll(profile)
		return nil, err
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		cmdR.Close()
		cmdW.Close()
		os.RemoveAll(profile)
		return nil, err
	}
	cmd := exec.Command(bin, "--remote-debugging-pipe", "--headless=new", "--user-data-dir="+profile,
		"--no-first-run", "--no-default-browser-check", "--hide-scrollbars", "--force-device-scale-factor=1", "about:blank")
	cmd.ExtraFiles = []*os.File{cmdR, outW}
	err = cmd.Start()
	cmdR.Close()
	outW.Close()
	if err != nil {
		cmdW.Close()
		outR.Close()
		os.RemoveAll(profile)
		return nil, err
	}
	h := &headless{cmd: cmd, conn: cdp.New(outR, cmdW), profile: profile}
	go func() {
		for range h.conn.Events() { // nothing listened for; keep the queue empty
		}
	}()
	var target struct {
		TargetID string `json:"targetId"`
	}
	var att struct {
		SessionID string `json:"sessionId"`
	}
	if err := h.call(ctx, "", "Target.createTarget", map[string]any{"url": "about:blank"}, &target); err != nil {
		h.close()
		return nil, err
	}
	if err := h.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}, &att); err != nil {
		h.close()
		return nil, err
	}
	h.session = att.SessionID
	return h, nil
}

func (h *headless) call(ctx context.Context, session, method string, params, result any) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return h.conn.Call(ctx, session, method, params, result)
}

// eval runs a script in the page, awaiting a promise, into result.
func (h *headless) eval(ctx context.Context, expr string, result any) error {
	var res struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err := h.call(ctx, h.session, "Runtime.evaluate", map[string]any{"expression": expr, "awaitPromise": true, "returnByValue": true}, &res); err != nil {
		return err
	}
	if e := res.ExceptionDetails; e != nil {
		return fmt.Errorf("%s %s", e.Text, e.Exception.Description)
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(res.Result.Value, result)
}

// open loads the fidelity page and waits until it has the records.
func (h *headless) open(ctx context.Context, url string) error {
	if err := h.call(ctx, h.session, "Page.navigate", map[string]any{"url": url}, nil); err != nil {
		return err
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		var ok bool
		err := h.eval(ctx, "window.fidelity ? window.fidelity.ready.then(() => true) : false", &ok)
		if err == nil && ok {
			return nil
		}
		if time.Now().After(deadline) {
			if err == nil {
				err = errors.New("the page never loaded")
			}
			return fmt.Errorf("fidelity page: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// score replays f's moment and diffs it with the tab's frame; over the
// tolerance, it tries fidelity.Retries and keeps the best. It returns the
// shot and its replay; live is false when nothing live was drawn at t.
func (h *headless) score(ctx context.Context, f *recording.Frame, tabPNG []byte) (fidelity.Shot, []byte, bool, error) {
	shot := fidelity.Shot{File: f.File, T: f.T}
	tab, err := png.Decode(bytes.NewReader(tabPNG))
	if err != nil {
		return shot, nil, false, fmt.Errorf("the tab's frame: %w", err)
	}
	var best []byte
	for _, off := range append([]int64{0}, fidelity.Retries...) {
		got, live, err := h.replayFrame(ctx, f, f.T+off)
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

// replayFrame draws the recording as the tab was at at (f's time, or
// near it) and screenshots it
// as the recorder screenshots the tab: the viewport, at f's scale. live is
// false when nothing live was drawn there.
func (h *headless) replayFrame(ctx context.Context, f *recording.Frame, at int64) (shot []byte, live bool, err error) {
	scale := f.Scale
	if scale <= 0 {
		scale = 0.5 // the fixtures' frames carry none
	}
	// The viewport in CSS px, from the frame: the replay lays out at it.
	vp := image.Pt(int(math.Round(float64(f.W)/scale)), int(math.Round(float64(f.H)/scale)))
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
	low, _, _, err := recorder.LowRes(full, 1/(2*scale))
	return low, true, err
}

func (h *headless) close() {
	h.conn.Close()
	if h.cmd.Process != nil {
		h.cmd.Process.Kill()
		h.cmd.Wait()
	}
	os.RemoveAll(h.profile)
}
