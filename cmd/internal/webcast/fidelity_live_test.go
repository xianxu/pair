package webcast

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/fidelity"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The fidelity check (#478) on the real fixtures, in a headless Chrome
// outside the agent sandbox:
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveFidelity -v
//
// FIDELITY_DIR=<recording>[,<recording>…] checks recordings in place
// (writing their report) instead of the fixtures, e.g. one made by
// TestLiveRecordURL.
func TestLiveFidelityFixtures(t *testing.T) {
	bin := liveFidelityChrome(t)
	dirs := strings.Split(os.Getenv("FIDELITY_DIR"), ",")
	if dirs[0] == "" {
		dirs = nil
		for _, name := range []string{"stress", "app", "blog"} {
			dirs = append(dirs, copyRecording(t, recording.Fixture(name), name))
		}
	}
	for _, dir := range dirs {
		rep, err := CheckFidelity(context.Background(), bin, dir)
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		t.Logf("%s: %s", filepath.Base(dir), rep.Summary())
		for _, s := range rep.Shots {
			t.Logf("  t=%d %s %.2f%%", s.T, s.File, s.Diff)
		}
		if !rep.Within() {
			t.Errorf("%s: %s, over the %g%% tolerance (report in %s)", filepath.Base(dir), rep.Summary(), fidelity.Tolerance, filepath.Join(dir, fidelity.ReportDir))
		}
		for _, f := range []string{"report.json", "report.md", "worst-1.png"} {
			if _, err := os.Stat(filepath.Join(dir, fidelity.ReportDir, f)); err != nil {
				t.Errorf("%s: %v", filepath.Base(dir), err)
			}
		}
		// The report leaves the recording valid.
		if fs, _, err := recording.ValidateDir(dir); err != nil || len(recording.Errors(fs)) != 0 {
			t.Errorf("%s after the check: %v %v", filepath.Base(dir), err, recording.Errors(fs))
		}
	}
}

// A render that has lost the page's styles is caught: every frame is over
// the tolerance, and the report shows them side by side.
func TestLiveFidelityCatchesBrokenRender(t *testing.T) {
	bin := liveFidelityChrome(t)
	dir := copyRecording(t, recording.Fixture("blog"), "blog-unstyled")
	unstyle(t, dir)
	rep, err := CheckFidelity(context.Background(), bin, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("unstyled blog: %s", rep.Summary())
	if rep.Within() || rep.Over != len(rep.Shots) || rep.Median <= fidelity.Tolerance {
		t.Fatalf("a render without its stylesheets passed: %s", rep.Summary())
	}
	md, err := os.ReadFile(filepath.Join(dir, fidelity.ReportDir, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range rep.Worst {
		if s := rep.Shots[i]; !strings.Contains(string(md), s.File) || !s.Over {
			t.Errorf("worst %+v not reported as over", s)
		}
	}
}

func liveFidelityChrome(t *testing.T) string {
	t.Helper()
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to replay in a real Chrome")
	}
	bin, err := FindChrome(os.Getenv, exec.LookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	return bin
}

// copyRecording copies a recording to a temporary directory named name,
// so the check's report doesn't land in testdata.
func copyRecording(t *testing.T, src, name string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), name)
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o700)
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), body, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// unstyle removes a recording's stylesheets (its CSS assets, sheet
// records, <style> elements and stylesheet links), as a renderer that
// dropped them would draw.
func unstyle(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, recording.FileEvents)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	for _, line := range bytes.SplitAfter(raw, []byte("\n")) {
		var r struct {
			K    string `json:"k"`
			MIME string `json:"mime"`
		}
		if len(bytes.TrimSpace(line)) == 0 || json.Unmarshal(line, &r) != nil {
			out.Write(line)
			continue
		}
		if r.K == "sheet" || (r.K == "asset" && strings.HasPrefix(r.MIME, "text/css")) {
			continue
		}
		// A <style> becomes an inert <script> (the viewer runs none), and
		// a stylesheet link loads nothing.
		line = bytes.ReplaceAll(line, []byte(`"n":"style"`), []byte(`"n":"script"`))
		line = bytes.ReplaceAll(line, []byte(`"rel":"stylesheet"`), []byte(`"rel":"unstyled"`))
		out.Write(line)
	}
	if err := os.WriteFile(path, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}
