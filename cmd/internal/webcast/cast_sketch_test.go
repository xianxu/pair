package webcast

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSketchSourceUsesLiveHubOrPrivateFallback(t *testing.T) {
	for _, kind := range []string{"hub", "custom", "absent"} {
		t.Run(kind, func(t *testing.T) {
			var sink recording.Sink
			var hub *recording.Hub
			mem := &recording.MemSink{}
			switch kind {
			case "hub":
				hub = recording.NewHub(0)
				sink = hub
			case "custom":
				sink = mem
			}
			output, source := sketchSourceOutput(sink)
			if kind == "hub" && source != hub {
				t.Fatal("duplicated production live log")
			}
			b := recording.NewBuilder("fallback")
			b.Page(recording.SegLive, "http://x/")
			for _, r := range b.Recs {
				if err := output.Put(r); err != nil {
					t.Fatal(err)
				}
			}
			begin := &recording.SketchBegin{Header: recording.Header{T: 1}, SketchOperation: recording.SketchOperation{Operation: "begin", Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, ID: "sk", Owner: recording.OperatorID, Author: recording.Author{ID: recording.OperatorID, Color: "#123456"}, Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 100, H: 100, DPR: 1}}
			if _, err := source.CandidateSketchSource(begin, recording.SketchSourcePrefixLimit, recording.SketchSourceStateLimit); err != nil {
				t.Fatal(err)
			}
			if kind == "custom" && len(mem.Records()) != len(b.Recs) {
				t.Fatal("fallback replaced custom sink")
			}
		})
	}
}

func TestSketchAssetReaderOwnsBoundedDiskChunks(t *testing.T) {
	dir := t.TempDir()
	hash := strings.Repeat("d", 64)
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	data := bytes.Repeat([]byte("private"), 10000)
	if err := os.WriteFile(filepath.Join(dir, recording.AssetName(hash)), data, 0600); err != nil {
		t.Fatal(err)
	}
	read := sketchAssetReader(dir)
	part, total, err := read(hash, 5, 7)
	if err != nil || total != len(data) || !bytes.Equal(part, data[5:12]) {
		t.Fatalf("%q %d %v", part, total, err)
	}
	for _, args := range [][2]int{{-1, 1}, {0, 0}, {0, 32769}, {len(data) + 1, 1}} {
		if _, _, err := read(hash, args[0], args[1]); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, _, err := read("../../secret", 0, 1); err == nil {
		t.Fatal("invalid hash accepted")
	}
}
