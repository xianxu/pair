package buildsample

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func copySample(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	err := filepath.WalkDir("../..", func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, e := filepath.Rel("../..", p)
		if e != nil {
			return e
		}
		dst := filepath.Join(root, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		return os.WriteFile(dst, b, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}
func launch(t *testing.T, binary string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	cmd := exec.CommandContext(ctx, binary, "--listen", "127.0.0.1:0")
	out, e := cmd.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { cancel(); _ = cmd.Wait() })
	ready := make(chan string, 1)
	go func() {
		s := bufio.NewScanner(out)
		if s.Scan() {
			ready <- s.Text()
		} else {
			ready <- ""
		}
	}()
	select {
	case line := <-ready:
		if !strings.HasPrefix(line, "listening http://") {
			t.Fatalf("startup %q %s", line, stderr.String())
		}
		return strings.TrimPrefix(line, "listening ")
	case <-ctx.Done():
		t.Fatal("startup timeout")
		return ""
	}
}
func get(t *testing.T, url string) (int, string) {
	t.Helper()
	c := http.Client{Timeout: 2 * time.Second}
	r, e := c.Get(url)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Body.Close()
	b, e := io.ReadAll(r.Body)
	if e != nil {
		t.Fatal(e)
	}
	return r.StatusCode, string(b)
}
func TestProductionPartialAndCompletedPrivateCopy(t *testing.T) {
	root := copySample(t)
	out := filepath.Join(t.TempDir(), "candidate")
	var report bytes.Buffer
	if e := Build(context.Background(), root, out, &report); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(report.String(), "detail (/runs/detail)") {
		t.Fatalf("missing report: %s", &report)
	}
	artifact, e := os.ReadFile(out)
	if e != nil {
		t.Fatal(e)
	}
	for _, sentinel := range []string{"MOCK_TO_REAL_FIXTURE_757", "MOCK_TO_REAL_HTML_757", "fixtures.json", "mock-1/detail.html", "example.com/mock-to-real/internal/catalogue", "example.com/mock-to-real/internal/buildsample"} {
		if bytes.Contains(artifact, []byte(sentinel)) {
			t.Fatalf("production leaked %s", sentinel)
		}
	}
	url := launch(t, out)
	if code, body := get(t, url+"/runs/detail"); code != 200 || !strings.Contains(body, "Not implemented yet") || !strings.Contains(body, `href="/"`) {
		t.Fatalf("placeholder %d %s", code, body)
	}
	if _, body := get(t, url+"/runs"); !strings.Contains(body, "Not implemented") {
		t.Fatal(body)
	}
	for _, path := range []string{"/fixtures.json", "/mocks/fixtures.json", "/mocks/mock-1/detail.html", "/reload", "/__reload", "/version", "/versions"} {
		if code, _ := get(t, url+path); code != 404 {
			t.Fatalf("exposed %s", path)
		}
	}
	var existing bytes.Buffer
	if e := Build(context.Background(), root, out, &existing); e == nil {
		t.Fatal("overwrote existing output")
	}
	after, _ := os.ReadFile(out)
	if !bytes.Equal(artifact, after) {
		t.Fatal("changed existing artifact")
	}
	file := filepath.Join(root, "mocks/screens.json")
	b, _ := os.ReadFile(file)
	var cat struct {
		Version int              `json:"version"`
		Screens []map[string]any `json:"screens"`
	}
	if e := json.Unmarshal(b, &cat); e != nil {
		t.Fatal(e)
	}
	for _, s := range cat.Screens {
		if s["id"] == "detail" {
			s["state"] = "real"
		}
	}
	b, _ = json.Marshal(cat)
	if e := os.WriteFile(file, b, 0644); e != nil {
		t.Fatal(e)
	}
	if e := Build(context.Background(), root, filepath.Join(t.TempDir(), "invalid"), io.Discard); e == nil || !strings.Contains(e.Error(), "detail") {
		t.Fatalf("missing handler not refused: %v", e)
	}
	source := filepath.Join(root, "internal/app/app.go")
	b, _ = os.ReadFile(source)
	patched := strings.Replace(string(b), "return map[string]http.HandlerFunc{", `return map[string]http.HandlerFunc{"detail":func(w http.ResponseWriter,r *http.Request){fmt.Fprint(w,"Real detail implementation 757")},`, 1)
	if patched == string(b) {
		t.Fatal("registration anchor missing")
	}
	if e := os.WriteFile(source, []byte(patched), 0644); e != nil {
		t.Fatal(e)
	}
	completed := filepath.Join(t.TempDir(), "completed")
	report.Reset()
	if e := Build(context.Background(), root, completed, &report); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(report.String(), "Unimplemented") {
		t.Fatal(report.String())
	}
	completeURL := launch(t, completed)
	if _, body := get(t, completeURL+"/runs/detail"); !strings.Contains(body, "Real detail implementation 757") || strings.Contains(body, "Not implemented") {
		t.Fatal(body)
	}
}
func TestCanceledBuildPublishesNothing(t *testing.T) {
	root := copySample(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	output := filepath.Join(t.TempDir(), "candidate")
	if e := Build(ctx, root, output, io.Discard); e == nil {
		t.Fatal("canceled build succeeded")
	}
	if _, e := os.Stat(output); !os.IsNotExist(e) {
		t.Fatalf("published on cancellation: %v", e)
	}
}

func TestActiveChildCancellationJoins(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, t.TempDir(), "sh", "-c", `printf ready > "$1"; exec sleep 30`, "build-probe", marker)
	}()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, e := os.Stat(marker); e == nil {
			break
		}
		select {
		case e := <-done:
			t.Fatalf("child ended before cancellation: %v", e)
		case <-ctx.Done():
			t.Fatal("child readiness timeout")
		case <-ticker.C:
		}
	}
	cancel()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("canceled child succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("child was not joined")
	}
}

type brokenReport struct{}

func (brokenReport) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestReportFailureDoesNotLeaveArtifact(t *testing.T) {
	scratch := t.TempDir()
	t.Setenv("TMPDIR", scratch)
	root := copySample(t)
	output := filepath.Join(t.TempDir(), "candidate")
	if e := Build(context.Background(), root, output, brokenReport{}); e == nil {
		t.Fatal("report failure hidden")
	}
	if _, e := os.Stat(output); !os.IsNotExist(e) {
		t.Fatal("failed build left an output")
	}
	stages, e := filepath.Glob(filepath.Join(scratch, "mock-to-real-build-*"))
	if e != nil || len(stages) != 0 {
		t.Fatalf("stages %v: %v", stages, e)
	}
}
