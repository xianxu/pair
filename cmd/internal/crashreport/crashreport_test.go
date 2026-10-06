package crashreport

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

// TestMain doubles as the crashing child: a panic can only be observed from
// outside the process that panics, so the test re-executes its own binary.
func TestMain(m *testing.M) {
	if dir := os.Getenv("CRASHREPORT_TEST_PANIC_DIR"); dir != "" {
		if _, _, err := Install(dir, time.Now(), os.Getpid()); err != nil {
			os.Stderr.WriteString("install: " + err.Error())
			os.Exit(3)
		}
		panic("crashreport test panic")
	}
	os.Exit(m.Run())
}

func TestPanicLandsInCrashFileAndStderr(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "crash")
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "CRASHREPORT_TEST_PANIC_DIR="+dir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("child did not crash")
	}
	if !strings.Contains(stderr.String(), "crashreport test panic") {
		t.Fatalf("stderr lost the panic: %q", stderr.String())
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("crash dir = %v, %v; want one file", entries, err)
	}
	if _, _, ext, ok := parseName(entries[0].Name()); !ok || ext != unreported {
		t.Fatalf("unexpected crash file name %q", entries[0].Name())
	}
	body, _ := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	for _, want := range []string{"panic: crashreport test panic", "goroutine "} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("crash file lacks %q:\n%s", want, body)
		}
	}
}

func TestCleanCloseLeavesNoFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "crash")
	capture, _, err := Install(dir, time.Now(), 4242)
	if err != nil {
		t.Fatal(err)
	}
	if err := capture.Close(); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("clean exit left %v", entries)
	}
}

// The next incarnation reports each previous ending exactly once.
func TestInstallReportsPreviousEndingsOnce(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "crash")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("20261006T135600Z-111.log", "panic: boom\n\ngoroutine 1 [running]:\n")
	write("20261006T120000Z-222.log", "")
	write("20261001T000000Z-333.crash", "panic: old\n")
	write("notes.txt", "not ours")

	capture, reports, err := Install(dir, time.Now(), 4242)
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Close()
	if len(reports) != 2 {
		t.Fatalf("reports = %+v, want a crash and an abrupt end", reports)
	}
	kinds := map[Kind]Report{}
	for _, r := range reports {
		kinds[r.Kind] = r
	}
	crash, abrupt := kinds[Crashed], kinds[Abrupt]
	if crash.Path != filepath.Join(dir, "20261006T135600Z-111.crash") || !strings.Contains(crash.Notice(), crash.Path) {
		t.Fatalf("crash report %+v", crash)
	}
	if _, err := os.Stat(crash.Path); err != nil {
		t.Fatalf("reported crash not kept: %v", err)
	}
	if abrupt.Notice() == "" {
		t.Fatalf("abrupt report %+v", abrupt)
	}
	if _, err := os.Stat(filepath.Join(dir, "20261006T120000Z-222.log")); !os.IsNotExist(err) {
		t.Fatalf("empty previous file kept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatalf("foreign file touched: %v", err)
	}
	capture.Close()

	_, again, err := Install(dir, time.Now(), 4243)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("second startup re-reported %+v", again)
	}
}

func TestClassify(t *testing.T) {
	got := Classify([]File{
		{Name: "20261006T135600Z-1.log", Size: 10},
		{Name: "20261006T135600Z-2.log", Size: 0},
		{Name: "20261006T135600Z-3.crash", Size: 10},
		{Name: "20261006T135600Z-x.log", Size: 10},
		{Name: "2026-10-06-4.log", Size: 10},
		{Name: "20261006T135600Z-5.log.tmp", Size: 10},
	})
	want := []Finding{
		{Name: "20261006T135600Z-1.log", Kind: Crashed},
		{Name: "20261006T135600Z-2.log", Kind: Abrupt},
		{Name: "20261006T135600Z-3.crash", Kind: Reported},
	}
	if len(got) != len(want) {
		t.Fatalf("Classify = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Classify[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSweepAgesOutAndSparesTheLiveOwner(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	old := now.Add(-(diagnosticlog.RetentionPeriod + 24*time.Hour))
	files := map[string]time.Time{
		"20251001T000000Z-1.crash": old, // expired, reported
		"20251001T000000Z-2.log":   old, // expired, owner dead
		"20251001T000000Z-3.log":   old, // expired, owner alive
		"20261006T000000Z-4.crash": now, // young
	}
	for name, at := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(dir, "20251001T000000Z-1.crash"), filepath.Join(dir, "20251001T000000Z-9.crash")); err != nil {
		t.Fatal(err)
	}
	alive := func(pid int) bool { return pid == 3 }

	preview, err := Sweep(dir, now, alive, false, 100)
	if err != nil {
		t.Fatal(err)
	}
	eligible := map[string]bool{}
	for _, row := range preview {
		eligible[filepath.Base(row.Path)] = row.Eligible
	}
	want := map[string]bool{
		"20251001T000000Z-1.crash": true,
		"20251001T000000Z-2.log":   true,
		"20251001T000000Z-3.log":   false,
		"20261006T000000Z-4.crash": false,
	}
	if len(eligible) != len(want) {
		t.Fatalf("preview rows = %+v (symlinks must be skipped)", preview)
	}
	for name, e := range want {
		if eligible[name] != e {
			t.Fatalf("%s eligible = %v, want %v", name, eligible[name], e)
		}
	}
	for name := range files {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("preview removed %s", name)
		}
	}

	if _, err := Sweep(dir, now, alive, true, 100); err != nil {
		t.Fatal(err)
	}
	for name, e := range want {
		_, err := os.Stat(filepath.Join(dir, name))
		if gone := os.IsNotExist(err); gone != e {
			t.Fatalf("%s removed = %v, want %v", name, gone, e)
		}
	}
}

func TestSweepMissingDirIsEmpty(t *testing.T) {
	rows, err := Sweep(filepath.Join(t.TempDir(), "absent"), time.Now(), func(int) bool { return false }, true, 10)
	if err != nil || len(rows) != 0 {
		t.Fatalf("Sweep(absent) = %v, %v", rows, err)
	}
}
