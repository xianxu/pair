package fidelity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Offsets are the moments around each screenshot the replay is tried at,
// best one kept: a screenshot shows the last frame painted before its
// timestamp, up to a few hundred ms earlier. #453's harness tried the same
// window in 200 ms steps; a page that re-renders every 50 ms (the stress
// page) needs 50 ms steps, or the step is what is measured.
var Offsets = func() []int64 {
	var out []int64
	for o := int64(-600); o <= 200; o += 50 {
		out = append(out, o)
	}
	return out
}()

// Shot is one screenshot's result: the share of differing pixels, in %.
type Shot struct {
	File   string
	T      int64
	Diff   float64
	Offset int64
}

// Report is a recording's fidelity: per shot, and median and max in %.
// Skipped counts shots with no live replay to compare.
type Report struct {
	Shots       []Shot
	Median, Max float64
	Skipped     int
}

func (r Report) String() string {
	var per []string
	for _, s := range r.Shots {
		per = append(per, fmt.Sprintf("%.2f", s.Diff))
	}
	out := fmt.Sprintf("median %.2f%% max %.2f%% (%s)", r.Median, r.Max, strings.Join(per, " "))
	if r.Skipped > 0 {
		out += fmt.Sprintf(", %d shots without a live replay", r.Skipped)
	}
	return out
}

// Run replays a recording directory at each of its frames in a headless
// Chrome at chromeBin and diffs against the screenshots.
func Run(ctx context.Context, chromeBin, dir string) (Report, error) {
	r, err := rec.ReadDir(dir)
	if err != nil {
		return Report{}, err
	}
	if len(r.Frames) == 0 {
		return Report{}, errors.New("fidelity: the recording has no frames")
	}
	b, err := launch(ctx, chromeBin)
	if err != nil {
		return Report{}, err
	}
	defer b.close()
	var rep Report
	var shots []Shot
	for _, f := range r.Frames {
		want, err := os.ReadFile(filepath.Join(dir, f.File))
		if err != nil {
			return rep, err
		}
		var tries []Try
		for _, off := range Offsets {
			st, err := rec.Reduce(r.Records, f.T+off)
			if err != nil {
				return rep, err
			}
			if st.Top == nil || !st.Live {
				continue // nothing live to replay at this moment
			}
			got, err := b.shoot(ctx, r, st, f)
			if err != nil {
				return rep, fmt.Errorf("%s at %d: %w", f.File, f.T+off, err)
			}
			d, err := Diff(want, got)
			if err != nil {
				return rep, fmt.Errorf("%s: %w", f.File, err)
			}
			tries = append(tries, Try{Offset: off, Diff: d * 100})
			if out := os.Getenv("FIDELITY_OUT"); out != "" && d*100 <= Pick(f.File, f.T, tries).Diff {
				// The best replay of each shot so far, for looking at.
				name := strings.ReplaceAll(filepath.Base(dir)+"-"+f.File, "/", "-")
				os.WriteFile(filepath.Join(out, name), got, 0o600)
			}
		}
		shots = append(shots, Pick(f.File, f.T, tries))
	}
	rep = Summarize(shots)
	if len(rep.Shots) == 0 {
		return rep, errors.New("fidelity: no shot has a live replay")
	}
	return rep, nil
}

// Try is one replay of a shot: at an offset from its time, with the % of
// pixels that differ.
type Try struct {
	Offset int64
	Diff   float64
}

// Pick is a shot's result: its best try; a shot with no try (nothing live
// near it) has Diff -1.
func Pick(file string, t int64, tries []Try) Shot {
	s := Shot{File: file, T: t, Diff: -1}
	for _, tr := range tries {
		if s.Diff < 0 || tr.Diff < s.Diff {
			s.Diff, s.Offset = tr.Diff, tr.Offset
		}
	}
	return s
}

// Summarize is a report over picked shots: the shots without a try are
// counted as skipped, never scored, and the rest give the median and max.
func Summarize(shots []Shot) Report {
	var rep Report
	for _, s := range shots {
		if s.Diff < 0 {
			rep.Skipped++
			continue
		}
		rep.Shots = append(rep.Shots, s)
	}
	if len(rep.Shots) == 0 {
		return rep
	}
	ds := make([]float64, len(rep.Shots))
	for i, s := range rep.Shots {
		ds[i] = s.Diff
	}
	sort.Float64s(ds)
	rep.Median, rep.Max = ds[len(ds)/2], ds[len(ds)-1]
	return rep
}

// browser is a headless Chrome on its debugging pipe, one page attached.
// Not webcast.Chrome: that one owns its event loop, and the replay needs the
// raw Fetch events to serve the recording's bodies.
type browser struct {
	cmd     *exec.Cmd
	conn    *cdp.Conn
	session string
	profile string
	// serve answers a paused request: body and mime, or ok false (fail).
	mu     sync.Mutex
	serve  func(url string) ([]byte, string, bool)
	loaded chan struct{}
	done   chan struct{}
}

func launch(ctx context.Context, bin string) (*browser, error) {
	profile, err := os.MkdirTemp("", "fidelity-profile-")
	if err != nil {
		return nil, err
	}
	fail := func(err error, files ...*os.File) (*browser, error) {
		for _, f := range files {
			f.Close()
		}
		os.RemoveAll(profile)
		return nil, err
	}
	cmdR, cmdW, err := os.Pipe()
	if err != nil {
		return fail(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		return fail(err, cmdR, cmdW)
	}
	cmd := exec.Command(bin, "--remote-debugging-pipe", "--headless=new", "--user-data-dir="+profile,
		"--no-first-run", "--no-default-browser-check", "--hide-scrollbars", "--force-device-scale-factor=1", "about:blank")
	cmd.ExtraFiles = []*os.File{cmdR, outW}
	if err := cmd.Start(); err != nil {
		return fail(err, cmdR, cmdW, outR, outW)
	}
	cmdR.Close()
	outW.Close()
	b := &browser{cmd: cmd, conn: cdp.New(outR, cmdW), profile: profile, loaded: make(chan struct{}, 1), done: make(chan struct{})}
	go b.events()
	var t struct {
		ID string `json:"targetId"`
	}
	if err := b.call(ctx, "", "Target.createTarget", map[string]any{"url": "about:blank"}, &t); err != nil {
		b.close()
		return nil, err
	}
	var a struct {
		Session string `json:"sessionId"`
	}
	if err := b.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": t.ID, "flatten": true}, &a); err != nil {
		b.close()
		return nil, err
	}
	b.session = a.Session
	for _, m := range []string{"Page.enable", "Runtime.enable"} {
		if err := b.call(ctx, b.session, m, nil, nil); err != nil {
			b.close()
			return nil, err
		}
	}
	if err := b.call(ctx, b.session, "Fetch.enable", map[string]any{"patterns": []map[string]any{{"urlPattern": "*"}}}, nil); err != nil {
		b.close()
		return nil, err
	}
	return b, nil
}

func (b *browser) call(ctx context.Context, session, method string, params, result any) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return b.conn.Call(ctx, session, method, params, result)
}

// events answers paused requests from the recording and notes loads.
func (b *browser) events() {
	defer close(b.done)
	for ev := range b.conn.Events() {
		switch ev.Method {
		case "Fetch.requestPaused":
			var p struct {
				ID      string `json:"requestId"`
				Request struct {
					URL string `json:"url"`
				} `json:"request"`
			}
			if json.Unmarshal(ev.Params, &p) != nil {
				continue
			}
			go b.answer(ev.SessionID, p.ID, p.Request.URL)
		case "Page.loadEventFired":
			select {
			case b.loaded <- struct{}{}:
			default:
			}
		}
	}
}

func (b *browser) answer(session, id, url string) {
	ctx := context.Background()
	b.mu.Lock()
	serve := b.serve
	b.mu.Unlock()
	if serve != nil {
		if body, mime, ok := serve(url); ok {
			b.call(ctx, session, "Fetch.fulfillRequest", map[string]any{"requestId": id, "responseCode": 200,
				"responseHeaders": []map[string]string{{"name": "Content-Type", "value": mime}},
				"body":            base64.StdEncoding.EncodeToString(body)}, nil)
			return
		}
	}
	// Never the origin: a URL the recording doesn't have loads nothing.
	b.call(ctx, session, "Fetch.failRequest", map[string]any{"requestId": id, "errorReason": "BlockedByClient"}, nil)
}

func stripFragment(u string) string {
	if i := strings.IndexByte(u, '#'); i >= 0 {
		return u[:i]
	}
	return u
}

// shoot renders a state at the recording's viewport and screenshots it as
// the recorder did.
func (b *browser) shoot(ctx context.Context, r *rec.Recording, st *rec.State, f *rec.Frame) ([]byte, error) {
	page := Render(st)
	docURL := stripFragment(st.Top.URL)
	if docURL == "" || strings.HasPrefix(docURL, "about:") {
		docURL = "http://fidelity.invalid/"
	}
	serve := func(url string) ([]byte, string, bool) {
		if stripFragment(url) == docURL {
			return []byte(page), "text/html; charset=utf-8", true
		}
		a := st.Assets[url]
		if a == nil {
			return nil, "", false
		}
		body, err := r.Asset(a.Hash)
		return body, a.MIME, err == nil
	}
	b.mu.Lock()
	b.serve = serve
	b.mu.Unlock()
	vp := st.VP
	if err := b.call(ctx, b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": vp.W, "height": vp.H, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
		return nil, err
	}
	select {
	case <-b.loaded:
	default:
	}
	if err := b.call(ctx, b.session, "Page.navigate", map[string]any{"url": docURL}, nil); err != nil {
		return nil, err
	}
	select {
	case <-b.loaded:
	case <-time.After(10 * time.Second):
		return nil, errors.New("the replay didn't load")
	}
	// Scroll offsets, then a moment for fonts and images.
	var js strings.Builder
	js.WriteString("(() => {")
	for id, xy := range st.Scrolls {
		if id == st.Top.ID {
			fmt.Fprintf(&js, "window.scrollTo({left: %g, top: %g, behavior: 'instant'});", xy[0], xy[1])
			continue
		}
		fmt.Fprintf(&js, "{const e = document.querySelector('[%s=\"%d\"]'); if (e) { e.scrollTo({left: %g, top: %g, behavior: 'instant'}); }}", RIDAttr, id, xy[0], xy[1])
	}
	js.WriteString("return [window.scrollX, window.scrollY]; })()")
	if err := b.call(ctx, b.session, "Runtime.evaluate", map[string]any{"expression": js.String(), "returnByValue": true}, nil); err != nil {
		return nil, err
	}
	time.Sleep(250 * time.Millisecond)
	var shot struct {
		Data string `json:"data"`
	}
	if err := b.call(ctx, b.session, "Page.captureScreenshot", map[string]any{"format": "png"}, &shot); err != nil {
		return nil, err
	}
	full, err := base64.StdEncoding.DecodeString(shot.Data)
	if err != nil {
		return nil, err
	}
	low, _, _, err := recorder.LowRes(full, 1) // the replay runs at DPR 1
	return low, err
}

func (b *browser) close() {
	b.conn.Close()
	if b.cmd.Process != nil {
		b.cmd.Process.Kill()
		b.cmd.Wait()
	}
	os.RemoveAll(b.profile)
}
