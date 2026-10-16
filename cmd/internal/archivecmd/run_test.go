package archivecmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

type archiveStore struct {
	records  map[string]historyarchive.CatalogRecord
	attempts int
}

func (s *archiveStore) create(ctx context.Context, r historyarchive.ArchiveRequest) ([]historyarchive.CatalogRecord, error) {
	s.attempts++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.records == nil {
		s.records = map[string]historyarchive.CatalogRecord{}
	}
	record := historyarchive.CatalogRecord{Sink: r.Sink, Manifest: historyarchive.Manifest{Source: r.Source}, Label: historyarchive.BundleLabel{ID: "verified", FileCount: len(r.Files), Bytes: 123}}
	s.records[r.Catalog] = record
	return []historyarchive.CatalogRecord{record}, nil
}
func requestFile(t *testing.T, root string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "request.json")
	r := historyarchive.ArchiveRequest{Version: 1, Kind: "pair-scrollback", Source: historyarchive.SourceIdentity{Host: "test", Root: root}, Files: []string{"selected"}, Recipients: []string{strings.Repeat("A", 40)}, Sink: "/sink", Catalog: "/catalog"}
	b, _ := json.Marshal(r)
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestArchiveCLIInventoryAndHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"inventory"}} {
		var out, stderr bytes.Buffer
		s := &archiveStore{}
		if code := RunContext(context.Background(), args, &out, &stderr, s.create); code != 0 {
			t.Fatalf("%v: %d %s", args, code, &stderr)
		}
		if s.attempts != 0 || out.Len() == 0 {
			t.Fatal("read-only command attempted create or omitted output")
		}
		if len(args) > 0 && args[0] == "inventory" {
			if _, err := historyarchive.DecodeInventory(out.Bytes()); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestArchiveCLICreateAndRetry(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", "/configured/pair")
	file := requestFile(t, "")
	s := &archiveStore{}
	for i := 0; i < 2; i++ {
		var out, stderr bytes.Buffer
		if code := RunContext(context.Background(), []string{"create", "--request", file}, &out, &stderr, s.create); code != 0 {
			t.Fatalf("code=%d %s", code, &stderr)
		}
		if !strings.Contains(out.String(), "verified") || strings.Contains(out.String(), "selected") {
			t.Fatalf("bad public result: %s", &out)
		}
	}
	if len(s.records) != 1 || s.records["/catalog"].Manifest.Source.Root != "/configured/pair" {
		t.Fatalf("state=%+v", s.records)
	}
}
func TestArchiveCLIRejectsArguments(t *testing.T) {
	for _, args := range [][]string{{"wat"}, {"inventory", "extra"}, {"create"}, {"create", "--request"}, {"create", "--request", "x", "--request", "y"}, {"create", "--request", "x", "extra"}, {"--help", "extra"}} {
		var out, stderr bytes.Buffer
		s := &archiveStore{}
		if RunContext(context.Background(), args, &out, &stderr, s.create) == 0 || s.attempts != 0 {
			t.Fatalf("accepted %v", args)
		}
		if strings.Count(stderr.String(), "\n") != 1 {
			t.Fatalf("not one diagnostic: %q", stderr.String())
		}
	}
}
func TestArchiveCLIRejectsRequestFiles(t *testing.T) {
	for _, kind := range []string{"unknown field", "oversized", "directory", "symlink", "invalid", "missing", "fifo", "trailing"} {
		t.Run(kind, func(t *testing.T) {
			p := requestFile(t, "/source")
			switch kind {
			case "unknown field":
				os.WriteFile(p, []byte(`{"unknown":1}`), 0600)
			case "oversized":
				os.WriteFile(p, bytes.Repeat([]byte(" "), historyarchive.MaxMetadataBytes+1), 0600)
			case "directory":
				p = filepath.Dir(p)
			case "symlink":
				link := p + ".link"
				os.Symlink(p, link)
				p = link
			case "invalid":
				os.WriteFile(p, []byte(`{"version":`), 0600)
			case "missing":
				p += ".absent"
			case "fifo":
				p += ".fifo"
				if err := unix.Mkfifo(p, 0600); err != nil {
					t.Fatal(err)
				}
			case "trailing":
				b, _ := os.ReadFile(p)
				os.WriteFile(p, append(b, []byte(` {}`)...), 0600)
			}
			var out, stderr bytes.Buffer
			s := &archiveStore{}
			if RunContext(context.Background(), []string{"create", "--request", p}, &out, &stderr, s.create) == 0 || s.attempts != 0 {
				t.Fatal("accepted invalid request")
			}
		})
	}
}
func TestArchiveCLICancellationAndError(t *testing.T) {
	p := requestFile(t, "/source")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := &archiveStore{}
	var out, stderr bytes.Buffer
	if RunContext(ctx, []string{"create", "--request", p}, &out, &stderr, s.create) == 0 || len(s.records) != 0 {
		t.Fatal("canceled create succeeded")
	}
	stderr.Reset()
	if RunContext(context.Background(), []string{"create", "--request", p}, &out, &stderr, func(context.Context, historyarchive.ArchiveRequest) ([]historyarchive.CatalogRecord, error) {
		return nil, errors.New("verification failed\nretry safely")
	}) == 0 || strings.Count(stderr.String(), "\n") != 1 {
		t.Fatalf("bad error: %s", &stderr)
	}
}

type failedOutput struct{}

func (failedOutput) Write([]byte) (int, error) { return 0, errors.New("output closed") }
func TestArchiveCLIOutputFailure(t *testing.T) {
	for _, args := range [][]string{nil, {"inventory"}, {"create", "--request", requestFile(t, "/source")}} {
		var stderr bytes.Buffer
		s := &archiveStore{}
		if RunContext(context.Background(), args, failedOutput{}, &stderr, s.create) == 0 {
			t.Fatalf("output error ignored: %v", args)
		}
		if len(args) > 0 && args[0] == "create" && len(s.records) != 1 {
			t.Fatal("result failure lost committed backend state")
		}
	}
}
