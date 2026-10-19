package historyarchive_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

var downloadLabelName = strings.Repeat("a", 64) + ".json"
var downloadCipherName = strings.Repeat("a", 64) + ".tar.zst.gpg"
var downloadUnselectedName = strings.Repeat("b", 64) + ".tar.zst.gpg"

func downloadFixture(t *testing.T) (*archivetest.FakeRclone, ha.Rclone, ha.PinnedDrive, string, string) {
	t.Helper()
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	if err := os.Mkdir(dest, 0700); err != nil {
		t.Fatal(err)
	}
	fake := &archivetest.FakeRclone{Objects: map[string][]archivetest.FakeRemoteObject{downloadLabelName: {{ID: "a", Data: []byte(`{"label":true}`)}}, downloadCipherName: {{ID: "b", Data: []byte("ciphertext")}}, downloadUnselectedName: {{ID: "secret", Data: []byte("not selected")}}}}
	return fake, ha.Rclone{Runner: fake}, ha.PinnedDrive{Remote: "test", Root: "test:", FolderID: "pinned"}, dest, filepath.Join(root, "files")
}
func TestRcloneDownloadBatchSelectedOnly(t *testing.T) {
	f, r, pin, dest, list := downloadFixture(t)
	f.Before = func(ctx context.Context, operation, name string) error {
		if operation != "copy" {
			t.Fatalf("unexpected operation %s", operation)
		}
		st, err := os.Stat(list)
		if err != nil {
			return err
		}
		if st.Mode().Perm() != 0600 {
			t.Fatalf("filter permissions %v", st.Mode())
		}
		b, err := os.ReadFile(list)
		if string(b) != "+ /"+downloadLabelName+"\n+ /"+downloadCipherName+"\n- **\n" {
			t.Fatalf("filter %q", b)
		}
		return err
	}
	if err := r.DownloadBatch(context.Background(), pin, []string{downloadLabelName, downloadCipherName}, dest, list, 100, 100); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{downloadLabelName, downloadCipherName} {
		b, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil || string(b) != string(f.Objects[name][0].Data) {
			t.Fatalf("download %s %q %v", name, b, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, downloadUnselectedName)); !os.IsNotExist(err) {
		t.Fatal("unselected object downloaded")
	}
	if _, err := os.Stat(list); !os.IsNotExist(err) {
		t.Fatal("filter leaked")
	}
	if f.DownloadBatches != 1 || f.DownloadedLabelBytes != 14 || f.DownloadedCipherBytes != 10 || f.HashBatches != 0 || f.CopyBatches != 0 {
		t.Fatalf("counts %+v", f)
	}
}
func TestRcloneDownloadBatchRejectsInputs(t *testing.T) {
	for _, names := range [][]string{nil, {"a.json"}, {"a*.json"}, {downloadLabelName, downloadLabelName}, {"../a"}, {"a/b"}, {"a\nb"}, {"a\x00b"}, {"."}, make([]string, 100001)} {
		f, r, pin, dest, list := downloadFixture(t)
		if err := r.DownloadBatch(context.Background(), pin, names, dest, list, 100, 100); err == nil {
			t.Fatalf("accepted names %.100v", names)
		}
		if f.Invocations != 0 {
			t.Fatal("invalid input ran rclone")
		}
	}
	for _, bounds := range [][2]int64{{0, 1}, {1, 0}, {4<<30 + 1, 1}, {1, 2<<30 + 1}, {-1, 1}} {
		f, r, pin, dest, list := downloadFixture(t)
		if err := r.DownloadBatch(context.Background(), pin, []string{downloadLabelName}, dest, list, bounds[0], bounds[1]); err == nil {
			t.Fatalf("accepted bounds %v", bounds)
		}
		if f.Invocations != 0 {
			t.Fatal("invalid bounds ran rclone")
		}
	}
	for _, which := range []string{"dest", "list", "exists"} {
		f, r, pin, dest, list := downloadFixture(t)
		switch which {
		case "dest":
			dest = "relative"
		case "list":
			list = "relative"
		case "exists":
			if err := os.WriteFile(list, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := r.DownloadBatch(context.Background(), pin, []string{downloadLabelName}, dest, list, 100, 100); err == nil {
			t.Fatalf("accepted %s", which)
		}
		if f.Invocations != 0 {
			t.Fatal("unsafe paths ran rclone")
		}
		if which == "exists" {
			b, _ := os.ReadFile(list)
			if string(b) != "keep" {
				t.Fatal("existing filter modified")
			}
		}
	}
}
func TestRcloneDownloadBatchPartialAndCancel(t *testing.T) {
	for _, mode := range []string{"partial", "bytes", "late", "cancel", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			f, r, pin, dest, list := downloadFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "partial":
				f.FailAfterObjects = 1
			case "bytes":
				f.PartialCopyBytes = 3
				f.AfterCopyError = os.ErrPermission
			case "late":
				f.AfterCopyError = os.ErrPermission
			case "cancel":
				f.Before = func(context.Context, string, string) error { cancel(); return nil }
			case "duplicate":
				f.Objects[downloadLabelName] = append(f.Objects[downloadLabelName], f.Objects[downloadLabelName][0])
			}
			err := r.DownloadBatch(ctx, pin, []string{downloadLabelName, downloadCipherName}, dest, list, 100, 100)
			if err == nil {
				t.Fatal("fault not returned")
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel error %v", err)
			}
			if mode == "partial" {
				if b, e := os.ReadFile(filepath.Join(dest, downloadLabelName)); e != nil || len(b) != 14 {
					t.Fatalf("partial data %q %v", b, e)
				}
				if _, e := os.Stat(filepath.Join(dest, downloadCipherName)); !os.IsNotExist(e) {
					t.Fatal("partial copy continued")
				}
			}
			if mode == "bytes" {
				b, e := os.ReadFile(filepath.Join(dest, downloadLabelName))
				if e != nil || len(b) != 3 || f.DownloadedLabelBytes != 3 || f.DownloadedCipherBytes != 3 {
					t.Fatalf("partial bytes %q %v %+v", b, e, f)
				}
			}
			if _, e := os.Stat(list); !os.IsNotExist(e) {
				t.Fatal("filter leaked on failure")
			}
		})
	}
}
func TestRcloneDownloadBatchEnforcesByteLimits(t *testing.T) {
	t.Run("transfer", func(t *testing.T) {
		f, r, pin, dest, list := downloadFixture(t)
		if err := r.DownloadBatch(context.Background(), pin, []string{downloadLabelName, downloadCipherName}, dest, list, 18, 100); err == nil {
			t.Fatal("transfer overflow accepted")
		}
		if f.DownloadedLabelBytes+f.DownloadedCipherBytes > 18 {
			t.Fatalf("unbounded download %+v", f)
		}
		entries, _ := os.ReadDir(dest)
		var total int64
		for _, entry := range entries {
			s, _ := entry.Info()
			total += s.Size()
		}
		if total > 18 {
			t.Fatalf("unbounded retained files %d", total)
		}
	})
	t.Run("size", func(t *testing.T) {
		f, r, pin, dest, list := downloadFixture(t)
		if err := r.DownloadBatch(context.Background(), pin, []string{downloadLabelName, downloadCipherName}, dest, list, 100, 10); err != nil {
			t.Fatal(err)
		}
		if f.DownloadedLabelBytes != 0 || f.DownloadedCipherBytes != 10 {
			t.Fatalf("oversize copied %+v", f)
		}
		if _, err := os.Stat(filepath.Join(dest, downloadLabelName)); !os.IsNotExist(err) {
			t.Fatal("oversized file exists")
		}
	})
}

func TestRcloneDownloadLocalBackendConformance(t *testing.T) {
	if os.Getenv("PAIR_RCLONE_LOCAL_CONFORMANCE") != "1" {
		t.Skip("opt-in installed rclone local backend")
	}
	binary, err := exec.LookPath("rclone")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err = os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "rclone.conf")
	if err = os.WriteFile(config, []byte("[fixture]\ntype = local\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r := ha.Rclone{Runner: ha.ExecRcloneRunner{Binary: binary, Env: []string{"HOME=" + root, "RCLONE_CONFIG=" + config, "PATH=" + os.Getenv("PATH")}}, LocalConformance: true}
	pin := ha.PinnedDrive{Root: "fixture:" + source}
	for _, tc := range []struct {
		name                string
		data                []byte
		maxBytes, maxSize   int64
		wantError, wantFile bool
	}{{strings.Repeat("c", 64) + ".json", []byte("label"), 100, 100, false, true}, {strings.Repeat("d", 64) + ".json", []byte("too large"), 100, 4, false, false}, {strings.Repeat("e", 64) + ".tar.zst.gpg", []byte(strings.Repeat("x", 4<<20)), 1 << 20, 8 << 20, true, false}} {
		t.Run(tc.name, func(t *testing.T) {
			if err = os.WriteFile(filepath.Join(source, tc.name), tc.data, 0600); err != nil {
				t.Fatal(err)
			}
			dest := t.TempDir()
			list := filepath.Join(root, tc.name+".list")
			e := r.DownloadBatch(context.Background(), pin, []string{tc.name}, dest, list, tc.maxBytes, tc.maxSize)
			if tc.wantError && (e == nil || !strings.Contains(strings.ToLower(e.Error()), "max transfer limit reached")) {
				t.Fatalf("expected actual transfer-limit failure, got %v", e)
			}
			if (e != nil) != tc.wantError {
				t.Fatalf("transfer error=%v wantError=%v", e, tc.wantError)
			}
			b, e := os.ReadFile(filepath.Join(dest, tc.name))
			if tc.wantFile {
				if e != nil || string(b) != string(tc.data) {
					t.Fatalf("selected data %q %v", b, e)
				}
			} else if !os.IsNotExist(e) {
				t.Fatalf("unexpected retained file %d %v", len(b), e)
			}
			entries, e := os.ReadDir(dest)
			if e != nil {
				t.Fatal(e)
			}
			expectedCount := 0
			if tc.wantFile {
				expectedCount = 1
			}
			if len(entries) != expectedCount {
				t.Fatalf("selected-object boundary: retained %d files want %d", len(entries), expectedCount)
			}
			var total int64
			for _, entry := range entries {
				st, e := entry.Info()
				if e != nil {
					t.Fatal(e)
				}
				total += st.Size()
			}
			if total > tc.maxBytes {
				t.Fatalf("retained bytes %d exceed budget %d", total, tc.maxBytes)
			}
		})
	}
}
