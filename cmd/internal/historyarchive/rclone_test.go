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
func TestRclonePinnedCopyAndDownloadedHash(t *testing.T) {
	runner := &adapterRunner{}
	r := Rclone{Runner: runner}
	pin := PinnedDrive{Remote: "pair-backup", FolderID: "folder-123", Root: "pair-backup:"}
	if err := r.Copy(context.Background(), pin, "/tmp/private.gpg", "abc.gpg"); err != nil {
		t.Fatal(err)
	}
	args := strings.Join(runner.calls[0], " ")
	for _, want := range []string{"copyto /tmp/private.gpg pair-backup:abc.gpg", "--immutable", "--ignore-existing", "--checksum", "--drive-root-folder-id folder-123", "--retries 1", "--low-level-retries 1"} {
		if !strings.Contains(args, want) {
			t.Errorf("missing %q: %s", want, args)
		}
	}
	runner.output = []byte(strings.Repeat("a", 64) + "  abc.gpg\n")
	if got, err := r.DownloadSHA256(context.Background(), pin, "abc.gpg"); err != nil || got != strings.Repeat("a", 64) {
		t.Fatalf("hash %q %v", got, err)
	}
	runner.output = []byte(strings.Repeat("a", 64) + "  wrong.gpg\n")
	if _, err := r.DownloadSHA256(context.Background(), pin, "abc.gpg"); err == nil {
		t.Fatal("accepted wrong basename")
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
		case "secret":
			fmt.Fprint(os.Stderr, "SECRET credential")
			os.Exit(7)
		}
		os.Exit(0)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"large", "secret"} {
		runner := ExecRcloneRunner{Binary: binary, Env: append(os.Environ(), "PAIR_RCLONE_HELPER=1", "PAIR_RCLONE_MODE="+mode)}
		out, err := runner.Run(context.Background(), []string{"-test.run=^TestExecRcloneRunnerBoundsAndSanitizesOutput$"}, 128)
		if err == nil || len(out) != 0 || strings.Contains(err.Error(), "SECRET") {
			t.Fatalf("unsafe failure output %q %v", out, err)
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
	// Local backend accepts absolute paths; production Drive config stays relative.
	pin, err := r.ResolveDestination(context.Background(), DriveDestination{Remote: "fixture", Folder: folder})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.gpg")
	if err := os.WriteFile(source, []byte("encrypted-fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.Copy(context.Background(), pin, source, "fixture.gpg"); err != nil {
		t.Fatal(err)
	}
	if err := r.Copy(context.Background(), pin, source, "fixture.gpg"); err != nil {
		t.Fatal(err)
	}
	entries, err := r.ObserveFlatInventory(context.Background(), pin)
	if err != nil || len(entries["fixture.gpg"]) != 1 {
		t.Fatalf("inventory %v %v", entries, err)
	}
	digest, err := r.DownloadSHA256(context.Background(), pin, "fixture.gpg")
	want := sha256.Sum256([]byte("encrypted-fixture"))
	if err != nil || digest != hex.EncodeToString(want[:]) {
		t.Fatalf("download %q %v", digest, err)
	}
	if err := os.WriteFile(source, []byte("conflict"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.Copy(context.Background(), pin, source, "fixture.gpg"); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(filepath.Join(folder, "fixture.gpg"))
	if err != nil || string(actual) != "encrypted-fixture" {
		t.Fatalf("remote changed %q %v", actual, err)
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
