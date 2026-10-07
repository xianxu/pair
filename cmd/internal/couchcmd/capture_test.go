package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	term "github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/terminalcapture"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func TestCaptureExplicitOptInAndOptionalIsolation(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name, isolated, path string
		bad                  bool
	}{
		{"off", "", "", false}, {"regular Couch", "", filepath.Join(root, "cap"), false}, {"regular relative", "", "cap", true},
		{"relative", root, "cap", true}, {"outside", root, t.TempDir(), true},
		{"inside", root, filepath.Join(root, "cap"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{"COUCH_ISOLATED_ROOT": tc.isolated, "COUCH_CAPTURE_DIR": tc.path}
			got, err := capturePath(func(k string) string { return env[k] })
			if (err != nil) != tc.bad {
				t.Fatalf("path=%q err=%v", got, err)
			}
			if tc.path == "" && got != "" {
				t.Fatal("off returned path")
			}
		})
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	_, err := capturePath(func(k string) string {
		if k == "COUCH_ISOLATED_ROOT" {
			return root
		}
		return filepath.Join(root, "escape", "capture")
	})
	if err == nil {
		t.Fatal("symlink escape accepted")
	}
}

type partialCaptureWriter struct{ accepted []byte }

func (w *partialCaptureWriter) WriteContext(_ context.Context, p []byte) (int, error) {
	w.accepted = append(w.accepted, p[:2]...)
	return 2, io.ErrUnexpectedEOF
}

func readCaptureRecords(t *testing.T, dir string) []terminalcapture.Record {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var records []terminalcapture.Record
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var r terminalcapture.Record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatal(err)
		}
		records = append(records, r)
	}
	return records
}

func TestCaptureHostWritePreservesPartialFailure(t *testing.T) {
	rec, err := terminalcapture.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dst := &partialCaptureWriter{}
	writer := captureWriter{writer: dst, recorder: rec}
	bytes := []byte{27, '[', '3', '1', 'm', 255}
	n, err := writer.WriteContext(context.Background(), bytes)
	if n != 2 || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("%d %v", n, err)
	}
	if err := rec.Close(); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, r := range readCaptureRecords(t, rec.Dir()) {
		if r.Kind == "host-write" {
			found = true
			if string(r.Data) != string(bytes) || r.Accepted != 2 || r.Requested != len(bytes) || r.Error == "" {
				t.Fatalf("bad receipt %+v", r)
			}
		}
	}
	if !found {
		t.Fatal("no host write")
	}
}

func TestCaptureCompositionOffAndExplicitOpenFailure(t *testing.T) {
	root := t.TempDir()
	env := map[string]string{"COUCH_ISOLATED_ROOT": root, "COUCH_CAPTURE_MAX_MIB": "invalid-but-disabled"}
	config := consoleTraceConfig{getenv: func(k string) string { return env[k] }, root: root}
	con, runner, err := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil, config)
	if err != nil || con == nil {
		t.Fatalf("%v", err)
	}
	if runner.(*couchcore.PtyRunner).Observer != nil {
		t.Fatal("off installed observer")
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("off created capture files")
	}
	con.Stop()
	_ = con.CloseCapture()
	delete(env, "COUCH_CAPTURE_MAX_MIB")
	env["COUCH_CAPTURE_DIR"] = filepath.Join(root, "file")
	if err := os.WriteFile(env["COUCH_CAPTURE_DIR"], nil, 0600); err != nil {
		t.Fatal(err)
	}
	con, _, err = consoleRunnerFor("start", strings.NewReader(""), true, nil, nil, config)
	if err == nil || con != nil {
		t.Fatal("explicit invalid capture silently accepted")
	}
}

// Exercise several MiB through a real PTY, endpoint parser and host Presenter,
// not just the recorder. The final marker proves the whole startup arrived.
func captureStartupPayload() []byte {
	data := bytes.Repeat([]byte("x"), 3800*1024)
	return append(data, []byte("\x1b[2J\x1b[HCAPTURE379")...)
}

func TestCaptureChildHelper(t *testing.T) {
	if os.Getenv("PAIR_CAPTURE_CHILD_HELPER") != "1" {
		return
	}
	if _, err := os.Stdout.Write(captureStartupPayload()); err != nil {
		os.Exit(2)
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}

func TestCapturePTYBothBoundaries(t *testing.T) {
	for _, isolated := range []bool{false, true} {
		name := "regular"
		if isolated {
			name = "isolated"
		}
		t.Run(name, func(t *testing.T) { testCapturePTYBothBoundaries(t, isolated) })
	}
}
func testCapturePTYBothBoundaries(t *testing.T, isolated bool) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	if err := pty.Setsize(slave, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	env := map[string]string{"COUCH_CAPTURE_DIR": filepath.Join(root, "capture")}
	if isolated {
		env["COUCH_ISOLATED_ROOT"] = root
	}
	config := consoleTraceConfig{getenv: func(k string) string { return env[k] }, root: root}
	con, runner, err := consoleRunnerFor("start", slave, true, slave, slave, config)
	if err != nil {
		t.Fatal(err)
	}
	defer con.CloseCapture()
	r := runner.(*couchcore.PtyRunner)
	r.Environment = nil
	h, err := r.Start(root, []string{os.Args[0], "-test.run=^TestCaptureChildHelper$"}, []string{"PAIR_CAPTURE_CHILD_HELPER=1"})
	if err != nil {
		t.Fatal(err)
	}
	child := h.(couchcore.TerminalHandle).Terminal()
	defer child.Close()
	con.Attach(h.ID(), "capture", child)
	transport, err := ttyio.NewFile(master, master, false)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.Close()
	readCtx, cancelRead := context.WithCancel(context.Background())
	defer cancelRead()
	readDone := make(chan struct{})
	seen := make(chan struct{}, 1)
	go func() {
		defer close(readDone)
		var received []byte
		b := make([]byte, 8192)
		for {
			n, e := transport.ReadContext(readCtx, b)
			received = append(received, b[:n]...)
			if bytes.Contains(received, []byte("CAPTURE379")) {
				select {
				case seen <- struct{}{}:
				default:
				}
			}
			if e != nil {
				return
			}
		}
	}()
	done := make(chan int, 1)
	go func() { done <- con.Run() }()
	select {
	case <-seen:
	case <-time.After(30 * time.Second):
		con.Stop()
		t.Error("marker never reached host")
	}
	con.Stop()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("console failed to stop")
	}
	if err := con.CloseCapture(); err != nil {
		t.Fatal(err)
	}
	cancelRead()
	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("capture reader failed to join")
	}
	dirs, err := os.ReadDir(env["COUCH_CAPTURE_DIR"])
	if err != nil || len(dirs) != 1 {
		t.Fatalf("dirs %v %v", dirs, err)
	}
	records := readCaptureRecords(t, filepath.Join(env["COUCH_CAPTURE_DIR"], dirs[0].Name()))
	var feed, host, bind, geometry bool
	var ingress []byte
	for _, e := range records {
		if e.Kind == "endpoint-feed" && e.EndpointID == h.ID() {
			ingress = append(ingress, e.Data...)
		}
		if e.Kind == "endpoint-feed" && bytes.Contains(e.Data, []byte("CAPTURE379")) {
			feed = true
			if e.EndpointID != h.ID() {
				t.Fatal("wrong endpoint identity")
			}
		}
		if e.Kind == "host-write" && bytes.Contains(e.Data[:e.Accepted], []byte("CAPTURE379")) {
			host = true
		}
		if e.Kind == "thread-bind" && e.EndpointID == h.ID() {
			bind = true
		}
		if e.Kind == "host-geometry" && e.Cols == 80 && e.Rows == 24 {
			geometry = true
		}
	}
	if !bytes.Equal(ingress, captureStartupPayload()) {
		t.Fatalf("startup ingress mismatch: got %d bytes want %d", len(ingress), len(captureStartupPayload()))
	}
	if len(records) < 2 || records[len(records)-1].Status != "complete" {
		t.Fatal("startup capture incomplete")
	}
	if !feed || !host || !bind || !geometry {
		t.Fatalf("feed=%t host=%t bind=%t geometry=%t", feed, host, bind, geometry)
	}
}

type failedCaptureLaunchRuntime struct{ testRT }

func (r failedCaptureLaunchRuntime) NewCouchWith(runner couchcore.Runner, _ couchcore.CouchNamespace) (*couchcore.Couch, error) {
	// Model output overwhelming the recorder before startup fails. Exercise the
	// same injected observer as real first child output, without launching a child.
	runner.(*couchcore.PtyRunner).Observer(term.Observation{Kind: "endpoint-feed", EndpointID: "early", Data: make([]byte, 9<<20)})
	return nil, errors.New("fixture launch failed")
}
func TestCaptureEarlyLaunchFailureReportsDrainFailure(t *testing.T) {
	root := t.TempDir()
	rt := newRT(t)
	rt.env["COUCH_ISOLATED_ROOT"] = root
	rt.env["COUCH_CAPTURE_DIR"] = filepath.Join(root, "capture")
	op, _ := Resolve("start")
	var stderr bytes.Buffer
	code := runTypedOperationWithConsole(op, map[string]string{}, nil, true, "", nil, nil, strings.NewReader(""), io.Discard, &stderr, failedCaptureLaunchRuntime{rt}, nil)
	if code == 0 || !strings.Contains(stderr.String(), "fixture launch failed") || !strings.Contains(stderr.String(), "capture incomplete") {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if strings.Count(stderr.String(), "capture incomplete") != 1 {
		t.Fatalf("duplicate diagnostic %s", stderr.String())
	}
}

func TestCaptureConfiguredLimitValidationBeforeOpen(t *testing.T) {
	for _, value := range []string{"0", "-1", "+1", "1.5", " 1", "1048577", "999999999999999999999999"} {
		t.Run(value, func(t *testing.T) {
			root := t.TempDir()
			env := map[string]string{"COUCH_CAPTURE_DIR": filepath.Join(root, "capture"), "COUCH_CAPTURE_MAX_MIB": value}
			con, _, err := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil, consoleTraceConfig{getenv: func(k string) string { return env[k] }, root: root})
			if con != nil {
				con.Stop()
				_ = con.CloseCapture()
			}
			if err == nil || !strings.Contains(err.Error(), "COUCH_CAPTURE_MAX_MIB") {
				t.Fatalf("invalid limit accepted: %v", err)
			}
			if _, err := os.Stat(env["COUCH_CAPTURE_DIR"]); !os.IsNotExist(err) {
				t.Fatalf("invalid config touched destination: %v", err)
			}
		})
	}
}

func TestCaptureConfiguredLimitAppliesAtComposition(t *testing.T) {
	root := t.TempDir()
	env := map[string]string{"COUCH_CAPTURE_DIR": filepath.Join(root, "capture"), "COUCH_CAPTURE_MAX_MIB": "1"}
	con, runner, err := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil, consoleTraceConfig{getenv: func(k string) string { return env[k] }, root: root})
	if err != nil {
		t.Fatal(err)
	}
	defer con.Stop()
	runner.(*couchcore.PtyRunner).Observer(term.Observation{Kind: "endpoint-feed", EndpointID: "limit", Data: make([]byte, 1<<20)})
	if err := con.CloseCapture(); !errors.Is(err, terminalcapture.ErrFileLimit) {
		t.Fatalf("configured limit not enforced: %v", err)
	}
	dirs, err := os.ReadDir(env["COUCH_CAPTURE_DIR"])
	if err != nil || len(dirs) != 1 {
		t.Fatalf("capture dirs %v %v", dirs, err)
	}
	info, err := os.Stat(filepath.Join(env["COUCH_CAPTURE_DIR"], dirs[0].Name(), "events.jsonl"))
	if err != nil || info.Size() > 1<<20 {
		t.Fatalf("disk bound %v %v", info, err)
	}
}
