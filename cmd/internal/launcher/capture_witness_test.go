package launcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func TestCaptureTransferRejectsReplacedSource(t *testing.T) {
	d := t.TempDir()
	src, dst := filepath.Join(d, "source"), filepath.Join(d, "dest")
	os.WriteFile(src, []byte("original"), 0600)
	id, e := storagegc.CaptureIdentity(src)
	if e != nil {
		t.Fatal(e)
	}
	want := artifactpath.CaptureFileID{Device: id.Device, Inode: id.Inode}
	os.Rename(src, src+".old")
	os.WriteFile(src, []byte("replacement"), 0600)
	if e = transferFile(src, dst, true, &want); e == nil {
		t.Fatal("copied replacement as witnessed source")
	}
	if _, e = os.Stat(dst); !os.IsNotExist(e) {
		t.Fatal("published mismatched destination")
	}
	if _, e = os.Stat(src); e != nil {
		t.Fatal("removed replacement")
	}
	os.Remove(src)
	os.Rename(src+".old", src)
	if e = transferFile(src, dst, false, &want); e != nil {
		t.Fatal(e)
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "original" {
		t.Fatal("wrong capture")
	}
}

func TestCaptureOptionalEventsFailureNeverCertifies(t *testing.T) {
	for _, kind := range []string{"missing", "directory", "fifo", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			paths, _ := artifactpath.ResolveScoped(dir, "work")
			set, _ := paths.ScrollbackArtifacts("codex")
			os.WriteFile(set.Raw, []byte("retained raw"), 0600)
			switch kind {
			case "directory":
				os.Mkdir(set.Events, 0700)
			case "fifo":
				if err := syscall.Mkfifo(set.Events, 0600); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				f, err := os.Create(set.Events)
				if err != nil {
					t.Fatal(err)
				}
				err = f.Truncate(artifactpath.MaxCaptureWitnessEventsBytes + 1)
				f.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			rt := NewOSRuntime(dir, "/pair")
			base, ok := rt.ParkScrollback("work", "codex", true)
			if kind == "oversize" {
				if ok {
					t.Fatal("oversized authority accepted")
				}
				if raw, err := os.ReadFile(set.Raw); err != nil || string(raw) != "retained raw" {
					t.Fatal("rejected authority lost source")
				}
				return
			}
			if !ok {
				t.Fatal("optional events failure rejected raw")
			}
			data, e := os.ReadFile(base + ".capture.json")
			if e != nil {
				t.Fatal(e)
			}
			var meta artifactpath.CaptureMetadata
			if e = json.Unmarshal(data, &meta); e != nil {
				t.Fatal(e)
			}
			if meta.Version != 1 || meta.Witness != nil || meta.EventsIdentity != nil {
				t.Fatalf("invented authority: %+v", meta)
			}
		})
	}
}
