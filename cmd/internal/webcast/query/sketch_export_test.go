package query

import (
	"bytes"
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"os"
	"path/filepath"
	"testing"
)

func TestSketchImageCLIArguments(t *testing.T) {
	if _, err := parseArgs([]string{"recording", "sketch-image", "--sketch", "s0", "--out", "private-export", "--at", "14"}); err != nil {
		t.Fatalf("image export query is unavailable: %v", err)
	}
	for _, args := range [][]string{{"recording", "sketch-image"}, {"recording", "sketch-image", "--sketch", "s0"}, {"recording", "sketch-image", "--out", "private-export"}, {"recording", "sketch-image", "--sketch", "", "--out", "private-export"}, {"recording", "sketch-image", "--sketch", "s0", "--out", ""}, {"recording", "sketch-image", "--sketch", "s0", "--out", "private-export", "--json"}, {"recording", "sketch-image", "--sketch", "s0", "--out", "a", "--out", "b"}} {
		if _, err := parseArgs(args); err == nil {
			t.Fatalf("accepted invalid export arguments %q", args)
		}
	}
}

// The fake models one immutable export attempt. It observes read-complete input
// and can refuse publication; it never writes an image or launches a renderer.
type fakeSketchExporter struct {
	requests []SketchImageRequest
	err      error
	during   func()
}

func (f *fakeSketchExporter) export(ctx context.Context, r SketchImageRequest) error {
	f.requests = append(f.requests, r)
	if f.during != nil {
		f.during()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return f.err
}
func exportRecording(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	recs, _ := sketchRecords(1)
	var data bytes.Buffer
	for _, r := range recs {
		line, err := recording.Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		data.Write(line)
		data.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, recording.FileEvents), data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}
func TestSketchImageExporterReceivesFrozenReadCompleteEvidence(t *testing.T) {
	dir := exportRecording(t)
	output := filepath.Join(t.TempDir(), "image")
	var out, errout bytes.Buffer
	fake := &fakeSketchExporter{during: func() {
		if out.Len() != 0 {
			t.Error("success printed before publication")
		}
		if err := os.Remove(filepath.Join(dir, recording.FileEvents)); err != nil {
			t.Fatal(err)
		}
	}}
	code := RunWithOptions([]string{dir, "sketch-image", "--sketch", "s0", "--out", output, "--at", "14"}, RunOptions{ExportSketchImage: fake.export}, &out, &errout)
	if code != 0 || errout.Len() != 0 || out.String() != output+"\n" || len(fake.requests) != 1 {
		t.Fatalf("exit%d out=%q err=%q requests=%d", code, out.String(), errout.String(), len(fake.requests))
	}
	r := fake.requests[0]
	if r.RecordingDir != dir || r.OutputDir != output || len(r.Records) == 0 || r.Sketch.History.Phase != "complete" || r.Sketch.History.NextSequence != 5 || r.Sketch.Source.URL != "https://original.test/settings" {
		t.Fatalf("bad frozen export: %+v", r)
	}
	if begin, ok := r.Records[r.Sketch.Source.RecordIndex].(*recording.SketchBegin); !ok || begin.ID != "s0" {
		t.Fatal("missing exact source boundary")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("query shell unexpectedly wrote output: %v", err)
	}
}
func TestSketchImageExportRefusalsDoNotClaimSuccess(t *testing.T) {
	dir := exportRecording(t)
	output := filepath.Join(t.TempDir(), "image")
	for _, kind := range []string{"unconfigured", "missing", "before-begin", "publication", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			fake := &fakeSketchExporter{}
			opts := RunOptions{ExportSketchImage: fake.export}
			id, at := "s0", "14"
			wantCalls := 1
			switch kind {
			case "unconfigured":
				opts.ExportSketchImage = nil
				wantCalls = 0
			case "missing":
				id = "missing"
				wantCalls = 0
			case "before-begin":
				at = "9"
				wantCalls = 0
			case "publication":
				fake.err = errors.New("publication refused")
			case "canceled":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				opts.Context = ctx
				wantCalls = 0
			}
			var out, errout bytes.Buffer
			code := RunWithOptions([]string{dir, "sketch-image", "--sketch", id, "--out", output, "--at", at}, opts, &out, &errout)
			if code != 1 || out.Len() != 0 || errout.Len() == 0 || len(fake.requests) != wantCalls {
				t.Fatalf("exit%d out=%q err=%q requests=%d", code, out.String(), errout.String(), len(fake.requests))
			}
		})
	}
}
func TestSketchImageExportDefaultHistoryAndLegacyRun(t *testing.T) {
	dir := exportRecording(t)
	fake := &fakeSketchExporter{}
	var out, errout bytes.Buffer
	if code := RunWithOptions([]string{dir, "sketch-image", "--sketch", "s0", "--out", "export"}, RunOptions{ExportSketchImage: fake.export}, &out, &errout); code != 0 || len(fake.requests) != 1 || fake.requests[0].Sketch.History.Phase != "cleared" {
		t.Fatalf("default history: exit%d %s", code, errout.String())
	}
	out.Reset()
	errout.Reset()
	link := func(path string, at int64) (string, error) {
		if path != dir || at != 7 {
			t.Fatal("legacy link arguments changed")
		}
		return "https://replay.test/", nil
	}
	if code := Run([]string{dir, "link", "--at", "7"}, link, &out, &errout); code != 0 || out.String() != "https://replay.test/\n" {
		t.Fatalf("legacy forwarding: exit%d %q %q", code, out.String(), errout.String())
	}
}
