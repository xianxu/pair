package historyarchive_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

func testRequest(t *testing.T) ha.ArchiveRequest {
	t.Helper()
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	src := filepath.Join(root, "source")
	os.Mkdir(src, 0700)
	os.WriteFile(filepath.Join(src, "one"), []byte("one"), 0600)
	return ha.ArchiveRequest{Version: 1, Kind: "ops-documents", Source: ha.SourceIdentity{Host: "fixture", Root: src}, Files: []string{"one"}, Recipients: []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}, Sink: filepath.Join(root, "sink"), Catalog: filepath.Join(root, "catalog", "history.jsonl")}
}
func testCodec() *archivetest.TestCodec {
	return &archivetest.TestCodec{KnownRecipients: map[string]bool{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA": true}}
}
func TestArchiverVerificationFailureNeverCatalogsAndFreshRetryAdopts(t *testing.T) {
	req := testRequest(t)
	sink := &archivetest.MemorySink{VerifyError: errors.New("read-back unavailable")}
	codec := testCodec()
	a := ha.Archiver{Sink: sink, Codec: codec}
	if _, e := a.Create(context.Background(), req); e == nil {
		t.Fatal("verification failure accepted")
	}
	raw, e := os.ReadFile(req.Catalog)
	if e != nil {
		t.Fatal(e)
	}
	if len(raw) != 0 {
		t.Fatal("cataloged before verification")
	}
	sink.VerifyError = nil
	got, e := (ha.Archiver{Sink: sink, Codec: codec}).Create(context.Background(), req)
	if e != nil {
		t.Fatal(e)
	}
	if len(got) != 1 || codec.Attempts != 1 {
		t.Fatal("retry did not adopt existing verified bytes")
	}
}
func TestArchiverIncompletePublicationPreservesSource(t *testing.T) {
	req := testRequest(t)
	sink := &archivetest.MemorySink{AfterPutError: errors.New("publication sync uncertain")}
	codec := testCodec()
	if _, e := (ha.Archiver{Sink: sink, Codec: codec}).Create(context.Background(), req); e == nil {
		t.Fatal("uncertain publication accepted")
	}
	sink.AfterPutError = nil
	if _, e := (ha.Archiver{Sink: sink, Codec: codec}).Create(context.Background(), req); e == nil {
		t.Fatal("incomplete object pair adopted")
	}
	if codec.Attempts != 1 {
		t.Fatal("retry blindly re-encoded incomplete publication")
	}
	raw, _ := os.ReadFile(filepath.Join(req.Source.Root, "one"))
	if string(raw) != "one" {
		t.Fatal("source lost")
	}
}
func TestArchiverCancellationAtTransformJoinsAndLeavesNoRecord(t *testing.T) {
	req := testRequest(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered := make(chan struct{})
	exited := make(chan struct{})
	codec := testCodec()
	codec.Barrier = func(ctx context.Context, stage string) error {
		if stage == "before-encode" {
			close(entered)
			<-ctx.Done()
			close(exited)
			return ctx.Err()
		}
		return nil
	}
	done := make(chan error, 1)
	go func() {
		_, e := (ha.Archiver{Codec: codec, Sink: &archivetest.MemorySink{}}).Create(ctx, req)
		done <- e
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("transform not entered")
	}
	cancel()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("cancel accepted")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled call stranded")
	}
	select {
	case <-exited:
	default:
		t.Fatal("transform outlived invocation")
	}
	raw, _ := os.ReadFile(req.Catalog)
	if len(raw) != 0 {
		t.Fatal("canceled operation cataloged")
	}
	entries, _ := os.ReadDir(req.Catalog + ".staging")
	if len(entries) != 0 {
		t.Fatal("staging residue after cancellation")
	}
}
