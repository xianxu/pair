package historyarchive_test

import (
	"context"
	"errors"
	"fmt"
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
	for _, tc := range []struct {
		name        string
		total, size int64
	}{{"transfer", 18, 100}, {"size", 100, 10}} {
		t.Run(tc.name, func(t *testing.T) {
			f, r, pin, dest, list := downloadFixture(t)
			err := r.DownloadBatch(context.Background(), pin, []string{downloadLabelName, downloadCipherName}, dest, list, tc.total, tc.size)
			if err == nil || !strings.Contains(err.Error(), "exceeds admitted byte limit") {
				t.Fatalf("post-download overflow accepted: %v", err)
			}
			if f.DownloadedLabelBytes != 14 || f.DownloadedCipherBytes != 10 {
				t.Fatalf("selected objects not exercised: %+v", f)
			}
			if _, err := os.Stat(filepath.Join(dest, downloadUnselectedName)); !os.IsNotExist(err) {
				t.Fatal("unselected object downloaded")
			}
			if _, err := os.Stat(list); !os.IsNotExist(err) {
				t.Fatal("filter leaked after postcheck refusal")
			}
		})
	}
}

type streamingDownloadRunner struct {
	runner ha.ExecRcloneRunner
	calls  [][]string
}

func (r *streamingDownloadRunner) Run(ctx context.Context, args []string, limit int) ([]byte, error) {
	if args[0] == "copy" {
		args = append(append([]string(nil), args...), "--disable", "Copy")
	}
	r.calls = append(r.calls, append([]string(nil), args...))
	return r.runner.Run(ctx, args, limit)
}
func TestRcloneDownloadLocalBackendConformance(t *testing.T) {
	if os.Getenv("PAIR_RCLONE_LOCAL_CONFORMANCE") != "1" {
		t.Skip("opt-in installed rclone local backend")
	}
	binary, err := exec.LookPath("rclone")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source")
	if err = os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "rclone.conf")
	if err = os.WriteFile(config, []byte("[fixture]\ntype = local\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runner := &streamingDownloadRunner{runner: ha.ExecRcloneRunner{Binary: binary, Env: []string{"HOME=" + root, "RCLONE_CONFIG=" + config, "PATH=" + os.Getenv("PATH")}}}
	r := ha.Rclone{Runner: runner, LocalConformance: true}
	pin := ha.PinnedDrive{Root: "fixture:" + source}
	for _, tc := range []struct {
		name                     string
		bytes                    int
		total, size, backstop    int64
		wantError, coarseFailure bool
	}{
		{"small", 5, 5, 5, 1 << 20, false, false},
		{"odd3109", 3109, 3109, 3109, 1 << 20, false, false},
		{"exact2048", 2048, 2048, 2048, 1 << 20, false, false},
		{"exact1MiB", 1 << 20, 1 << 20, 1 << 20, 2 << 20, false, false},
		{"object-grew", 9, 100, 4, 1 << 20, true, false},
		{"coarse-transfer-limit", 4 << 20, 1 << 20, 8 << 20, 2 << 20, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := strings.Repeat("c", 64) + ".tar.zst.gpg"
			data := []byte(strings.Repeat("x", tc.bytes))
			if err := os.WriteFile(filepath.Join(source, name), data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, downloadUnselectedName), []byte("not selected"), 0600); err != nil {
				t.Fatal(err)
			}
			dest := t.TempDir()
			filter := filepath.Join(root, "filter")
			err := r.DownloadBatch(context.Background(), pin, []string{name}, dest, filter, tc.total, tc.size)
			if (err != nil) != tc.wantError {
				t.Fatalf("download error=%v wantError=%v", err, tc.wantError)
			}
			if tc.coarseFailure && !strings.Contains(strings.ToLower(err.Error()), "max transfer limit reached") {
				t.Fatalf("coarse transfer limit not exercised: %v", err)
			}
			if tc.wantError && !tc.coarseFailure && !strings.Contains(err.Error(), "exceeds admitted byte limit") {
				t.Fatalf("postcheck not exercised: %v", err)
			}
			if !tc.wantError {
				got, err := os.ReadFile(filepath.Join(dest, name))
				if err != nil || string(got) != string(data) {
					t.Fatalf("selected data bytes=%d err=%v", len(got), err)
				}
			}
			if _, err := os.Stat(filepath.Join(dest, downloadUnselectedName)); !os.IsNotExist(err) {
				t.Fatal("unselected object downloaded")
			}
			if _, err := os.Stat(filter); !os.IsNotExist(err) {
				t.Fatal("filter leaked")
			}
			call := strings.Join(runner.calls[len(runner.calls)-1], " ")
			if strings.Contains(call, "--max-size") || !strings.Contains(call, fmt.Sprintf("--max-transfer %dB", tc.backstop)) || !strings.Contains(call, "--disable Copy") {
				t.Fatalf("unexpected transport limits: %s", call)
			}
			// A failed native copy can retain partial transfer data. The coarse transport
			// cap bounds the attempted stream; the adapter's exact postcheck owns success.
		})
	}
}
