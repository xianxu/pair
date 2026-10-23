package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
)

func TestSketchImageExportPrivateAtomicPackage(t *testing.T) {
	b := imageBatch(t, 1)
	output := filepath.Join(t.TempDir(), "package")
	request := query.SketchImageRequest{RecordingDir: b.RecordingDir, OutputDir: output, Records: b.Records, Sketch: b.Sketches[0]}
	if err := ExportSketchImage(context.Background(), sketchimage.NewFakeRenderer(), request); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sketch.png", "metadata.json"} {
		st, err := os.Stat(filepath.Join(output, name))
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Perm() != 0600 {
			t.Fatalf("public evidence mode: %s", st.Mode())
		}
	}
	st, _ := os.Stat(output)
	if st.Mode().Perm() != 0700 {
		t.Fatalf("public package mode: %s", st.Mode())
	}
	data, err := os.ReadFile(filepath.Join(output, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		Sketch     query.SketchView `json:"sketch"`
		Provenance string           `json:"provenance"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Sketch.Source.URL != "https://source.test/" || metadata.Provenance != "replay-rendered" {
		t.Fatalf("export lost historical facts: %s", data)
	}
	if err := ExportSketchImage(context.Background(), sketchimage.NewFakeRenderer(), request); err == nil {
		t.Fatal("replaced existing export")
	}
}
func TestSketchImageExportRejectsRecordingDescendantAliases(t *testing.T) {
	b := imageBatch(t, 1)
	parent := t.TempDir()
	alias := filepath.Join(parent, "alias")
	if err := os.Symlink(b.RecordingDir, alias); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{filepath.Join(b.RecordingDir, "export"), filepath.Join(alias, "export")} {
		fake := sketchimage.NewFakeRenderer()
		request := query.SketchImageRequest{RecordingDir: b.RecordingDir, OutputDir: output, Records: b.Records, Sketch: b.Sketches[0]}
		if err := ExportSketchImage(context.Background(), fake, request); err == nil {
			t.Fatalf("wrote inside recording: %s", output)
		}
		if fake.Snapshot().Opens != 0 {
			t.Fatal("launched renderer before path admission")
		}
	}
}

type interceptImageRenderer struct {
	renderer sketchimage.ImageRenderer
	after    func()
}

func (r interceptImageRenderer) Render(ctx context.Context, b sketchimage.Batch) ([]sketchimage.Image, error) {
	images, err := r.renderer.Render(ctx, b)
	if r.after != nil {
		r.after()
	}
	return images, err
}
func TestSketchImageExportPreservesDestinationCreatedDuringRender(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(fmt.Sprint(symlink), func(t *testing.T) {
			b := imageBatch(t, 1)
			parent := t.TempDir()
			output := filepath.Join(parent, "package")
			other := t.TempDir()
			renderer := interceptImageRenderer{renderer: sketchimage.NewFakeRenderer(), after: func() {
				var err error
				if symlink {
					err = os.Symlink(other, output)
				} else {
					err = os.Mkdir(output, 0755)
				}
				if err != nil {
					t.Fatal(err)
				}
			}}
			request := query.SketchImageRequest{RecordingDir: b.RecordingDir, OutputDir: output, Records: b.Records, Sketch: b.Sketches[0]}
			if err := ExportSketchImage(context.Background(), renderer, request); err == nil {
				t.Fatal("overwrote raced export destination")
			}
			entries, err := os.ReadDir(parent)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != "package" {
				t.Fatalf("partial package/staging remains: %v", entries)
			}
			st, err := os.Lstat(output)
			if err != nil {
				t.Fatal(err)
			}
			if (st.Mode()&os.ModeSymlink != 0) != symlink {
				t.Fatal("destination identity replaced")
			}
		})
	}
}

func TestSketchImageExportRejectsCaseAliasWhenSupported(t *testing.T) {
	base := t.TempDir()
	recordingDir := filepath.Join(base, "CaseRecording")
	if err := os.Mkdir(recordingDir, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(base, "caserecording")
	actual, _ := os.Stat(recordingDir)
	alternate, err := os.Stat(alias)
	if err != nil || !os.SameFile(actual, alternate) {
		t.Skip("filesystem is case sensitive")
	}
	b := imageBatch(t, 1)
	request := query.SketchImageRequest{RecordingDir: recordingDir, OutputDir: filepath.Join(alias, "export"), Records: b.Records, Sketch: b.Sketches[0]}
	if err := ExportSketchImage(context.Background(), sketchimage.NewFakeRenderer(), request); err == nil {
		t.Fatal("exported through case alias inside recording")
	}
}
