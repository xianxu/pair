package historyarchive

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestDecodeFlatInventoryPreservesDuplicatesAndRejectsAmbiguity(t *testing.T) {
	raw := []byte(`[{"Name":"abc.gpg","ID":"1","Size":3,"IsDir":false,"Hashes":{"SHA-256":"abc"}},{"Name":"abc.gpg","ID":"2","Size":3,"IsDir":false}]`)
	got, err := DecodeFlatInventory(raw)
	if err != nil || len(got["abc.gpg"]) != 2 {
		t.Fatalf("duplicate names lost: %v %v", got, err)
	}
	for _, bad := range []string{`null`, `[{"Name":"a","name":"b","Size":0,"IsDir":false}]`, `[{"Name":"x","Name":"y","Size":1,"IsDir":false}]`, `[{"Name":"../x","Size":1,"IsDir":false}]`, `[{"Name":"x","Size":null,"IsDir":false}]`, `[{"Name":"x","Size":1}]`, `[{"Name":"x","Size":9223372036854775808,"IsDir":false}]`} {
		if _, err := DecodeFlatInventory([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

type adapterRunner struct {
	calls  [][]string
	output []byte
}

func (r *adapterRunner) Run(_ context.Context, args []string, _ int) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	return r.output, nil
}
func TestRclonePinnedBatchCopyAndDownloadedHashes(t *testing.T) {
	runner := &adapterRunner{}
	r := Rclone{Runner: runner}
	pin := PinnedDrive{Remote: "pair-backup", FolderID: "folder-123", Root: "pair-backup:"}
	if err := r.CopyBatch(context.Background(), pin, "/tmp/private-stage"); err != nil {
		t.Fatal(err)
	}
	args := strings.Join(runner.calls[0], " ")
	for _, want := range []string{"copy /tmp/private-stage pair-backup:", "--transfers 8", "--immutable", "--ignore-existing", "--checksum", "--drive-root-folder-id folder-123", "--retries 1", "--low-level-retries 1"} {
		if !strings.Contains(args, want) {
			t.Errorf("missing %q: %s", want, args)
		}
	}
	runner.output = []byte(strings.Repeat("a", 64) + "  abc.gpg\n" + strings.Repeat("b", 64) + "  abc.json\n")
	got, err := r.DownloadSHA256Batch(context.Background(), pin, "/tmp/private-list")
	if err != nil || got["abc.gpg"] != strings.Repeat("a", 64) || len(got) != 2 {
		t.Fatalf("hashes %v %v", got, err)
	}
	args = strings.Join(runner.calls[1], " ")
	for _, want := range []string{"hashsum SHA256 pair-backup:", "--download", "--files-from-raw /tmp/private-list", "--checkers 8"} {
		if !strings.Contains(args, want) {
			t.Errorf("missing %q: %s", want, args)
		}
	}
	for _, raw := range []string{strings.Repeat("a", 64) + "  ../unsafe\n", strings.Repeat("a", 64) + "  abc\n" + strings.Repeat("b", 64) + "  abc\n", "malformed\n"} {
		runner.output = []byte(raw)
		if _, err := r.DownloadSHA256Batch(context.Background(), pin, "/tmp/private-list"); err == nil {
			t.Fatalf("accepted malformed hash result %q", raw)
		}
	}
}

func TestRcloneResolvesOnlyOneStableFolder(t *testing.T) {
	for _, raw := range []string{`[]`, `[{"Name":"pair-history","Size":-1,"IsDir":true}]`, `[{"Name":"pair-history","ID":"1","Size":-1,"IsDir":true},{"Name":"pair-history","ID":"2","Size":-1,"IsDir":true}]`} {
		r := Rclone{Runner: &adapterRunner{output: []byte(raw)}}
		if _, err := r.ResolveDestination(context.Background(), DriveDestination{Remote: "pair-backup", Folder: "backups/pair-history"}); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestExecRcloneRunnerBoundsAndSanitizesOutput(t *testing.T) {
	if os.Getenv("PAIR_RCLONE_HELPER") == "1" {
		switch os.Getenv("PAIR_RCLONE_MODE") {
		case "large":
			fmt.Print(strings.Repeat("x", 4096))
		case "tail":
			fmt.Fprintln(os.Stderr, strings.Repeat("prefix", 4096))
			fmt.Fprintln(os.Stderr, "\x1b[31mrate limit exceeded / destination missing\x1b[0m")
			os.Exit(8)
		case "secret":
			fmt.Fprintln(os.Stderr, `Authorization: Bearer SECRET-auth`)
			fmt.Fprintln(os.Stderr, `{"access_token":"SECRET-access","refresh_token":"SECRET-refresh","client_secret":"SECRET-client","password":"SECRET-password"}`)
			fmt.Fprintln(os.Stderr, "rate limit exceeded / destination missing")
			os.Exit(7)
		}
		os.Exit(0)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"large", "secret", "tail"} {
		runner := ExecRcloneRunner{Binary: binary, Env: append(os.Environ(), "PAIR_RCLONE_HELPER=1", "PAIR_RCLONE_MODE="+mode)}
		out, err := runner.Run(context.Background(), []string{"-test.run=^TestExecRcloneRunnerBoundsAndSanitizesOutput$"}, 128)
		if err == nil || len(out) != 0 || strings.Contains(err.Error(), "SECRET") {
			t.Fatalf("unsafe failure output %q %v", out, err)
		}
		if mode != "large" && !strings.Contains(err.Error(), "rate limit exceeded / destination missing") {
			t.Fatalf("useful stderr lost: %v", err)
		}
		if strings.ContainsRune(err.Error(), 0x1b) {
			t.Fatalf("terminal control retained: %q", err.Error())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := ExecRcloneRunner{Binary: binary}
	if _, err := runner.Run(ctx, nil, 128); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
}

func TestRcloneLocalBackendConformance(t *testing.T) {
	if os.Getenv("PAIR_RCLONE_LOCAL_CONFORMANCE") != "1" {
		t.Skip("opt-in installed rclone local backend")
	}
	binary, err := exec.LookPath("rclone")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	config := filepath.Join(root, "rclone.conf")
	if err := os.WriteFile(config, []byte("[fixture]\ntype = local\n"), 0600); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(root, "backups", "pair-history")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	r := Rclone{Runner: ExecRcloneRunner{Binary: binary, Env: []string{"HOME=" + root, "RCLONE_CONFIG=" + config, "PATH=" + os.Getenv("PATH")}}, LocalConformance: true}
	pin, err := r.ResolveDestination(context.Background(), DriveDestination{Remote: "fixture", Folder: folder})
	if err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "staging")
	if err := os.Mkdir(stage, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"fixture.gpg", "fixture.json"} {
		if err := os.WriteFile(filepath.Join(stage, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "not-selected-secret"), []byte("not ciphertext"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.CopyBatch(context.Background(), pin, stage); err != nil {
		t.Fatal(err)
	}
	if err := r.CopyBatch(context.Background(), pin, stage); err != nil {
		t.Fatal(err)
	}
	entries, err := r.ObserveFlatInventory(context.Background(), pin)
	if err != nil || len(entries) != 2 || len(entries["fixture.gpg"]) != 1 {
		t.Fatalf("inventory %v %v", entries, err)
	}
	list := filepath.Join(root, "hash-list")
	if err := os.WriteFile(list, []byte("fixture.gpg\n"), 0600); err != nil {
		t.Fatal(err)
	}
	hashes, err := r.DownloadSHA256Batch(context.Background(), pin, list)
	sum := sha256.Sum256([]byte("fixture.gpg"))
	if err != nil || len(hashes) != 1 || hashes["fixture.gpg"] != hex.EncodeToString(sum[:]) {
		t.Fatalf("hashes %v %v", hashes, err)
	}
	if err := os.WriteFile(filepath.Join(stage, "fixture.gpg"), []byte("conflict"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.CopyBatch(context.Background(), pin, stage); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(filepath.Join(folder, "fixture.gpg"))
	if err != nil || string(actual) != "fixture.gpg" {
		t.Fatalf("remote overwritten %q %v", actual, err)
	}
}

func FuzzDecodeFlatInventory(f *testing.F) {
	for _, seed := range []string{`[]`, `null`, `[{"Name":"a","name":"b","Size":0,"IsDir":false}]`, `[{"Name":"a","Size":0,"IsDir":false}]`, `[{"Name":"a","Size":0,"IsDir":false,"Hashes":{"MD5":"x","MD5":"y"}}]`} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		entries, err := DecodeFlatInventory(raw)
		if err != nil {
			return
		}
		total := 0
		for name, rows := range entries {
			if !flatName(name) {
				t.Fatal("unsafe accepted name")
			}
			total += len(rows)
			for _, entry := range rows {
				if entry.Name != name || entry.Size < 0 && !entry.IsDir {
					t.Fatal("invalid accepted entry")
				}
			}
		}
		if total > maxFlatInventoryEntries {
			t.Fatal("entry bound exceeded")
		}
	})
}

// The socket is a readiness event: cancellation happens only after the owned
// helper installed its signal waiter and sent its authenticated PID.
func TestExecRcloneRunnerCollectsStartedChildOnCancellation(t *testing.T) {
	if os.Getenv("PAIR_RCLONE_CANCEL_HELPER") == "1" {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM)
		defer signal.Stop(signals)
		conn, err := net.Dial("tcp", os.Getenv("PAIR_RCLONE_READY_ADDRESS"))
		if err != nil {
			os.Exit(9)
		}
		if _, err := fmt.Fprintf(conn, "%s %d\n", os.Getenv("PAIR_RCLONE_READY_TOKEN"), os.Getpid()); err != nil {
			os.Exit(10)
		}
		_ = conn.Close()
		<-signals
		os.Exit(0)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	token := hex.EncodeToString(nonce)
	ctx := t.Context()
	if deadline, ok := t.Deadline(); ok {
		var release context.CancelFunc
		ctx, release = context.WithDeadline(ctx, deadline)
		defer release()
	}
	boundCtx := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	runner := ExecRcloneRunner{Binary: binary, Env: append(os.Environ(), "PAIR_RCLONE_CANCEL_HELPER=1", "PAIR_RCLONE_READY_ADDRESS="+listener.Addr().String(), "PAIR_RCLONE_READY_TOKEN="+token)}
	done := make(chan error, 1)
	settled := make(chan struct{})
	defer func() { cancel(); <-settled }()
	go func() {
		defer close(settled)
		_, err := runner.Run(ctx, []string{"-test.run=^TestExecRcloneRunnerCollectsStartedChildOnCancellation$"}, 128)
		done <- err
	}()
	type readiness struct {
		pid int
		err error
	}
	ready := make(chan readiness, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			ready <- readiness{err: err}
			return
		}
		defer conn.Close()
		if deadline, ok := t.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		}
		var gotToken string
		var pid int
		_, err = fmt.Fscan(conn, &gotToken, &pid)
		if err == nil && (gotToken != token || pid <= 0) {
			err = errors.New("invalid readiness identity")
		}
		ready <- readiness{pid, err}
	}()
	var pid int
	select {
	case event := <-ready:
		if event.err != nil {
			t.Fatal(event.err)
		}
		pid = event.pid
	case err := <-done:
		t.Fatalf("helper exited before readiness: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("running child cancellation = %v", err)
		}
	case <-boundCtx.Done():
		t.Fatal("test canceled before child settled")
	}
	// Signal 0 observes the PID from our authenticated handshake; it sends no signal.
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("owned helper %d still exists after Run returned: %v", pid, err)
	}
}

func TestRcloneDiagnosticRedactsCredentialsAndPreservesTail(t *testing.T) {
	for _, raw := range []string{`password="SECRET\"still-secret"`, `access_token = SECRET-access`, `refresh_token: 'SECRET refresh'`, `client_secret=SECRET-client`, `Authorization: Bearer SECRET-auth`, `request failed Bearer SECRET-token`, `{"password":"SECRET\"still-secret"}`} {
		tail := &rcloneStderrTail{}
		_, _ = tail.Write([]byte(raw + "\nrate limit exceeded\n"))
		got := tail.diagnostic()
		if strings.Contains(got, "SECRET") || strings.Contains(got, "still-secret") || !strings.Contains(got, "rate limit exceeded") {
			t.Fatalf("unsafe or missing diagnostic: %q", got)
		}
	}
	tail := &rcloneStderrTail{}
	_, _ = tail.Write([]byte(strings.Repeat("x", 16<<10) + "\n"))
	_, _ = tail.Write([]byte("\x1b[31mdestination missing\x1b[0m\n"))
	if got := tail.diagnostic(); len(got) > 16<<10 || got != "destination missing" {
		t.Fatalf("bad retained tail %q", got)
	}
	tail = &rcloneStderrTail{}
	exact := "destination missing" + strings.Repeat("x", (16<<10)-len("destination missing"))
	_, _ = tail.Write([]byte(exact))
	if tail.diagnostic() != exact {
		t.Fatal("untruncated exact-bound diagnostic dropped")
	}
}

func TestRcloneLocalDownloadBoundConformance(t *testing.T) {
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
	config := filepath.Join(root, "rclone.conf")
	if err := os.WriteFile(config, []byte("[fixture]\ntype = local\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	objects := map[string]string{strings.Repeat("a", 64) + ".tar.zst.gpg": "12345", strings.Repeat("b", 64) + ".json": "abc", strings.Repeat("c", 64) + ".json": "xy", strings.Repeat("d", 64) + ".json": "123456"}
	for name, data := range objects {
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	r := Rclone{Runner: &downloadBoundStreamingRunner{runner: ExecRcloneRunner{Binary: binary, Env: []string{"HOME=" + root, "RCLONE_CONFIG=" + config, "PATH=" + os.Getenv("PATH")}}}, LocalConformance: true}
	pin, err := r.ResolveDestination(context.Background(), DriveDestination{Remote: "fixture", Folder: source})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name        string
		names       []string
		total, size int64
		copied      int
		wantError   bool
	}{
		{"exact single", []string{strings.Repeat("a", 64) + ".tar.zst.gpg"}, 5, 5, 1, false},
		{"exact batch", []string{strings.Repeat("b", 64) + ".json", strings.Repeat("c", 64) + ".json"}, 5, 3, 2, false},
		{"object grew", []string{strings.Repeat("d", 64) + ".json"}, 5, 5, 1, true},
		{"batch budget", []string{strings.Repeat("a", 64) + ".tar.zst.gpg"}, 4, 5, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dest := filepath.Join(root, strings.ReplaceAll(tc.name, " ", "-"))
			if err := os.Mkdir(dest, 0700); err != nil {
				t.Fatal(err)
			}
			filter := filepath.Join(root, "filter")
			err := r.DownloadBatch(context.Background(), pin, tc.names, dest, filter, tc.total, tc.size)
			if (err != nil) != tc.wantError {
				t.Fatalf("download error=%v, wantError=%v", err, tc.wantError)
			}
			entries, readErr := os.ReadDir(dest)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != tc.copied {
				t.Fatalf("copied=%d want=%d", len(entries), tc.copied)
			}
			for _, entry := range entries {
				data, err := os.ReadFile(filepath.Join(dest, entry.Name()))
				if err != nil || string(data) != objects[entry.Name()] {
					t.Fatal("download differs", err)
				}
			}
			if _, err := os.Lstat(filter); !os.IsNotExist(err) {
				t.Fatal("filter retained", err)
			}
		})
	}
}

// Force native streaming: server-side local copies bypass transfer accounting.
type downloadBoundStreamingRunner struct{ runner ExecRcloneRunner }

func (r *downloadBoundStreamingRunner) Run(ctx context.Context, args []string, limit int) ([]byte, error) {
	if args[0] == "copy" {
		args = append(append([]string(nil), args...), "--disable", "Copy")
	}
	return r.runner.Run(ctx, args, limit)
}
func TestDownloadTransferBackstopBoundaries(t *testing.T) {
	for _, tc := range []struct{ total, want int64 }{{1, 1 << 20}, {3109, 1 << 20}, {1<<20 - 1, 1 << 20}, {1 << 20, 2 << 20}, {4 << 30, 8 << 30}} {
		if got := downloadTransferBackstop(tc.total); got != tc.want || got <= tc.total || got&(got-1) != 0 {
			t.Fatalf("backstop(%d)=%d want=%d", tc.total, got, tc.want)
		}
	}
}
