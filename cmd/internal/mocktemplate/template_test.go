package mocktemplate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/mockserve"
	"github.com/xianxu/pair/internal/reporoot"
	"github.com/xianxu/pair/internal/testbinary"
)

// A nested Go module is invisible to the root go test ./...; this boundary makes
// the sample's contracts and real production builds part of the standard suite.
var sampleBinaryCache testbinary.Cache

func TestMain(m *testing.M) {
	code := m.Run()
	if err := sampleBinaryCache.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}

func sampleExecutable(t *testing.T, module string) string {
	t.Helper()
	directory, err := sampleBinaryCache.Directory(func(directory string) error {
		//lint:allow test-go-build — this distinct sample app is built once per test process and removed by TestMain after every child joins
		build := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(directory, "app"), "./cmd/app")
		build.Dir = module
		if out, err := build.CombinedOutput(); err != nil {
			return fmt.Errorf("sample build: %w: %s", err, out)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(directory, "app")
}

func TestMockToRealTemplateModule(t *testing.T) {
	root, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "go", "test", "./...", "-count=1")
	cmd.Dir = filepath.Join(root, "examples", "mock-to-real")
	cmd.WaitDelay = 5 * time.Second
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("nested template: %v\n%s", err, out)
	}
}

func TestMockToRealTemplateHostConformance(t *testing.T) {
	root, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(root, "examples", "mock-to-real")
	bin := sampleExecutable(t, module)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	output := &startupOutput{ready: make(chan string, 1)}
	cmd := exec.CommandContext(ctx, bin, "--listen", "127.0.0.1:0")
	cmd.Dir = module
	cmd.Stdout, cmd.Stderr = output, output
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 5 * time.Second
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	appDone := make(chan error, 1)
	go func() { appDone <- cmd.Wait() }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-appDone:
		case <-time.After(10 * time.Second):
			t.Error("sample process not joined")
		}
	})
	appURL := awaitURL(t, output.ready)
	mocks := filepath.Join(t.TempDir(), "mocks")
	if err := os.CopyFS(mocks, os.DirFS(filepath.Join(module, "mocks"))); err != nil {
		t.Fatal(err)
	}
	frontCtx, stopFront := context.WithCancel(t.Context())
	frontOutput := &startupOutput{ready: make(chan string, 1)}
	frontDone := make(chan error, 1)
	go func() { frontDone <- mockserve.ServeWithReal(frontCtx, mocks, 0, frontOutput, appURL) }()
	joined := false
	t.Cleanup(func() {
		stopFront()
		if !joined {
			select {
			case <-frontDone:
			case <-time.After(10 * time.Second):
				t.Error("front not joined")
			}
		}
	})
	frontURL := awaitURL(t, frontOutput.ready)
	for _, tc := range []struct{ path, want string }{{"/", "Real overview"}, {"/runs", "Real runs"}, {"/runs/detail", "MOCK"}, {"/api/runs", "live-42"}, {"/fixtures.json", "runs"}} {
		body := requestBody(t, frontURL+tc.path)
		if !strings.Contains(body, tc.want) {
			t.Fatalf("%s missing %q: %s", tc.path, tc.want, body)
		}
	}
	stopFront()
	select {
	case err := <-frontDone:
		joined = true
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("front failed to stop")
	}
	// Closing the front must never stop the independently owned application.
	if body := requestBody(t, appURL+"/"); !strings.Contains(body, "Real overview") {
		t.Fatal("front shutdown stopped app")
	}
}

func requestBody(t *testing.T, target string) string {
	t.Helper()
	client := http.Client{Timeout: 10 * time.Second}
	res, err := client.Get(target)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("%s: %s %s %v", target, res.Status, body, err)
	}
	return string(body)
}

type startupOutput struct {
	mu    sync.Mutex
	data  bytes.Buffer
	ready chan string
	sent  bool
}

func (o *startupOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.data.Len()+len(p) > 64<<10 {
		return 0, fmt.Errorf("startup output exceeds bound")
	}
	_, _ = o.data.Write(p)
	if !o.sent {
		for _, line := range strings.Split(o.data.String(), "\n") {
			line = strings.TrimPrefix(line, "listening ")
			if strings.HasPrefix(line, "http://") {
				o.ready <- strings.TrimSpace(line)
				o.sent = true
				break
			}
		}
	}
	return len(p), nil
}
func awaitURL(t *testing.T, ready <-chan string) string {
	t.Helper()
	select {
	case u := <-ready:
		return strings.TrimSuffix(u, "/")
	case <-time.After(15 * time.Second):
		t.Fatal("listener did not publish readiness")
		return ""
	}
}
