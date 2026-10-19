package historyarchive

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
)

type restoreCopyDecoder struct {
	calls int
	late  error
}

func (d *restoreCopyDecoder) Decode(ctx context.Context, in io.Reader, out io.Writer) error {
	d.calls++
	_, err := io.Copy(out, in)
	return errors.Join(err, d.late, ctx.Err())
}

func restoreLocalFixture(t *testing.T) (Restorer, RestoreRequest, ArchiveRequest, string) {
	t.Helper()
	req, codec := archiveFixture(t)
	records, err := (Archiver{Codec: codec}).Create(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	scratch := filepath.Join(filepath.Dir(req.Source.Root), "scratch")
	if err := os.Mkdir(scratch, 0700); err != nil {
		t.Fatal(err)
	}
	request := RestoreRequest{IDs: []string{records[0].Label.ID}, To: filepath.Join(scratch, "result")}
	return Restorer{Config: ContinuousConfig{Sink: req.Sink, Catalog: req.Catalog}, Decoder: &restoreCopyDecoder{}}, request, req, records[0].BundleKey
}

func assertRestoreStageGone(t *testing.T, request RestoreRequest) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(filepath.Dir(request.To), ".pair-restore-staging")); !os.IsNotExist(err) {
		t.Fatal("owned stage remains", err)
	}
}

func TestRestoreTreeSyncSharesEntryBudgetAcrossSiblings(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name, "file"), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, limit := range []int{3, 4} {
		dir, err := openSafeDir(root, false)
		if err != nil {
			t.Fatal(err)
		}
		err = syncRestoreTree(context.Background(), dir, limit)
		dir.Close()
		if (err == nil) != (limit == 4) {
			t.Fatalf("limit %d: %v", limit, err)
		}
	}
}

func TestRestorerLocalPublishesCompletePrivateTree(t *testing.T) {
	r, req, original, _ := restoreLocalFixture(t)
	report, err := r.Run(context.Background(), req)
	if err != nil || report.Bundles != 1 || report.Files != 1 || report.Target != req.To || report.State.Phase != RestorePublished {
		t.Fatal(report, err)
	}
	got, err := os.ReadFile(filepath.Join(req.To, req.IDs[0], "files", "qa.md"))
	if err != nil || string(got) != "private fixture\n" {
		t.Fatal(string(got), err)
	}
	source, err := os.ReadFile(filepath.Join(original.Source.Root, "qa.md"))
	if err != nil || string(source) != "private fixture\n" {
		t.Fatal("source changed", string(source), err)
	}
	assertRestoreStageGone(t, req)
}

func TestRestorerRefusesCorruptInputBeforeDecoder(t *testing.T) {
	r, req, original, key := restoreLocalFixture(t)
	path := filepath.Join(original.Sink, key)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)/2] ^= 1
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	decoder := r.Decoder.(*restoreCopyDecoder)
	if _, err := r.Run(context.Background(), req); err == nil || decoder.calls != 0 {
		t.Fatal("corrupt ciphertext reached decoder", err, decoder.calls)
	}
	if _, err := os.Stat(req.To); !os.IsNotExist(err) {
		t.Fatal("published corrupt input", err)
	}
	assertRestoreStageGone(t, req)
}

func TestRestorerLateIntegrityFailureNeverPublishes(t *testing.T) {
	r, req, _, _ := restoreLocalFixture(t)
	r.Decoder = &restoreCopyDecoder{late: errors.New("late integrity failure")}
	if _, err := r.Run(context.Background(), req); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Fatal(err)
	}
	if _, err := os.Stat(req.To); !os.IsNotExist(err) {
		t.Fatal("published before decoder success", err)
	}
	assertRestoreStageGone(t, req)
}

func TestRestorerConcurrentTargetAndCommitUncertainty(t *testing.T) {
	t.Run("concurrent target", func(t *testing.T) {
		r, req, _, _ := restoreLocalFixture(t)
		r.Fault = func(phase string) error {
			if phase == "before-publish" {
				if err := os.Mkdir(req.To, 0700); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(req.To, "keep"), []byte("foreign"), 0600)
			}
			return nil
		}
		if _, err := r.Run(context.Background(), req); err == nil {
			t.Fatal("replaced concurrent target")
		}
		got, err := os.ReadFile(filepath.Join(req.To, "keep"))
		if err != nil || string(got) != "foreign" {
			t.Fatal(string(got), err)
		}
		assertRestoreStageGone(t, req)
	})
	t.Run("uncertain sync", func(t *testing.T) {
		r, req, _, _ := restoreLocalFixture(t)
		r.Fault = func(phase string) error {
			if phase == "sync-parent" {
				return errors.New("sync failed")
			}
			return nil
		}
		report, err := r.Run(context.Background(), req)
		if err == nil || commitoutcome.Of(err) != commitoutcome.Indeterminate || report.State.Phase != RestorePublicationUncertain {
			t.Fatal(report, err)
		}
		if _, err := os.Stat(filepath.Join(req.To, req.IDs[0], "files", "qa.md")); err != nil {
			t.Fatal("uncertain output incomplete", err)
		}
		assertRestoreStageGone(t, req)
	})
}

func TestRestorerPreservesForeignStageAndRejectsNoMatch(t *testing.T) {
	t.Run("foreign stage", func(t *testing.T) {
		r, req, _, _ := restoreLocalFixture(t)
		stage := filepath.Join(filepath.Dir(req.To), ".pair-restore-staging")
		if err := os.Mkdir(stage, 0700); err != nil {
			t.Fatal(err)
		}
		keep := filepath.Join(stage, "keep")
		if err := os.WriteFile(keep, []byte("operator"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := r.Run(context.Background(), req); err == nil {
			t.Fatal("adopted foreign stage")
		}
		if got, err := os.ReadFile(keep); err != nil || string(got) != "operator" {
			t.Fatal(string(got), err)
		}
	})
	t.Run("no literal match", func(t *testing.T) {
		r, req, _, _ := restoreLocalFixture(t)
		req.Paths = []string{"does-not-exist"}
		if _, err := r.Run(context.Background(), req); err == nil {
			t.Fatal("empty filtered request published")
		}
		if _, err := os.Stat(req.To); !os.IsNotExist(err) {
			t.Fatal("empty target published", err)
		}
		assertRestoreStageGone(t, req)
	})
}
