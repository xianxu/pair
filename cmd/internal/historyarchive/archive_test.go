package historyarchive

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type archiveTestCodec struct {
	calls  int
	refuse bool
}

func (c *archiveTestCodec) CheckRecipients(context.Context, []string) error {
	if c.refuse {
		return errors.New("no key")
	}
	return nil
}
func (c *archiveTestCodec) Encode(ctx context.Context, r io.Reader, w io.Writer, _ []string) error {
	c.calls++
	_, e := io.Copy(w, r)
	return e
}
func archiveFixture(t *testing.T) (ArchiveRequest, *archiveTestCodec) {
	t.Helper()
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	src := filepath.Join(root, "source")
	os.Mkdir(src, 0700)
	os.WriteFile(filepath.Join(src, "qa.md"), []byte("private fixture\n"), 0600)
	return ArchiveRequest{Version: 1, Kind: "ops-documents", Source: SourceIdentity{Host: "fixture", Root: src}, Files: []string{"qa.md"}, Recipients: []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}, Sink: filepath.Join(root, "sink"), Catalog: filepath.Join(root, "catalog", "history.jsonl")}, &archiveTestCodec{}
}
func TestArchiveRoundTripRetryAndPrivacy(t *testing.T) {
	req, codec := archiveFixture(t)
	ctx := context.Background()
	a := Archiver{Codec: codec, Now: func() time.Time { return time.Now().UTC() }}
	got, e := a.Create(ctx, req)
	if e != nil {
		t.Fatal(e)
	}
	if len(got) != 1 {
		t.Fatal(got)
	}
	got2, e := a.Create(ctx, req)
	if e != nil {
		t.Fatal(e)
	}
	if len(got2) != 1 || codec.calls != 1 {
		t.Fatalf("retry re-encoded: %d %v", codec.calls, e)
	}
	raw, e := os.ReadFile(filepath.Join(req.Sink, got[0].LabelKey))
	if e != nil {
		t.Fatal(e)
	}
	if stringContainsAny(string(raw), "qa.md", "private fixture", req.Source.Root) {
		t.Fatal("label leaks paths/content")
	}
	source, _ := os.ReadFile(filepath.Join(req.Source.Root, "qa.md"))
	if string(source) != "private fixture\n" {
		t.Fatal("source changed")
	}
}
func stringContainsAny(s string, values ...string) bool {
	for _, v := range values {
		if len(v) > 0 && stringsContains(s, v) {
			return true
		}
	}
	return false
}
func stringsContains(s, v string) bool {
	for i := 0; i+len(v) <= len(s); i++ {
		if s[i:i+len(v)] == v {
			return true
		}
	}
	return false
}
func TestArchiveAdoptsVerifiedObjectsAfterCatalogLoss(t *testing.T) {
	req, codec := archiveFixture(t)
	ctx := context.Background()
	a := Archiver{Codec: codec}
	first, e := a.Create(ctx, req)
	if e != nil {
		t.Fatal(e)
	}
	// Models the last catalog line being torn; objects remain complete.
	if e = os.WriteFile(req.Catalog, []byte(`{"label":`), 0600); e != nil {
		t.Fatal(e)
	}
	got, e := a.Create(ctx, req)
	if e != nil {
		t.Fatal(e)
	}
	if got[0].Label.ID != first[0].Label.ID || codec.calls != 1 {
		t.Fatal("failed adoption")
	}
}
func TestArchiveRejectsCorruptionAndMissingRecipient(t *testing.T) {
	req, codec := archiveFixture(t)
	ctx := context.Background()
	a := Archiver{Codec: codec}
	codec.refuse = true
	if _, e := a.Create(ctx, req); e == nil {
		t.Fatal("missing recipient accepted")
	}
	codec.refuse = false
	got, e := a.Create(ctx, req)
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(req.Sink, got[0].BundleKey), []byte("bad"), 0600)
	if _, e = a.Create(ctx, req); e == nil {
		t.Fatal("corrupt object accepted")
	}
}

func TestArchiveDestinationNamespacesDoNotOverlap(t *testing.T) {
	if !overlaps("/", "/private/tmp/sink") || !overlaps("/private/tmp/sink", "/") {
		t.Error("filesystem root overlaps every descendant")
	}
	req, codec := archiveFixture(t)
	req.Sink = req.Catalog + ".staging"
	_, err := (Archiver{Codec: codec}).Create(context.Background(), req)
	if err == nil {
		t.Fatal("accepted sink in private staging namespace")
	}
	if _, statErr := os.Stat(req.Catalog); !os.IsNotExist(statErr) {
		t.Fatalf("overlap must refuse before writes: %v", statErr)
	}
}
