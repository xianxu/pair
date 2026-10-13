package launcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	for _, missing := range []bool{true, false} {
		t.Run(fmt.Sprint(missing), func(t *testing.T) {
			dir := t.TempDir()
			paths, _ := artifactpath.ResolveScoped(dir, "work")
			set, _ := paths.ScrollbackArtifacts("codex")
			os.WriteFile(set.Raw, []byte("retained raw"), 0600)
			if !missing {
				os.Mkdir(set.Events, 0700)
			}
			rt := NewOSRuntime(dir, "/pair")
			base, ok := rt.ParkScrollback("work", "codex", true)
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
