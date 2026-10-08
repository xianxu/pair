package crashreport

import (
	"bytes"
	"fmt"
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
		if _, _, err := Install(dir, time.Now(), os.Getpid(), dead); err != nil {
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
	capture, _, err := Install(dir, time.Now(), 4242, dead)
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

	capture, reports, err := Install(dir, time.Now(), 4242, dead)
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
	if crash.Path != filepath.Join(dir, "20261006T135600Z-111.crash") {
		t.Fatalf("crash report %+v", crash)
	}
	if summary := Summary(reports); !strings.Contains(summary, crash.Path) || !strings.Contains(summary, "abruptly") {
		t.Fatalf("one notice must carry both endings: %q", summary)
	}
	if _, err := os.Stat(crash.Path); err != nil {
		t.Fatalf("reported crash not kept: %v", err)
	}
	if abrupt.Kind != Abrupt {
		t.Fatalf("abrupt report %+v", abrupt)
	}
	if _, err := os.Stat(filepath.Join(dir, "20261006T120000Z-222.log")); !os.IsNotExist(err) {
		t.Fatalf("empty previous file kept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatalf("foreign file touched: %v", err)
	}
	capture.Close()

	again2, again, err := Install(dir, time.Now(), 4243, dead)
	if err != nil {
		t.Fatal(err)
	}
	defer again2.Close()
	if len(again) != 0 {
		t.Fatalf("second startup re-reported %+v", again)
	}
}

func dead(int) bool { return false }

// One stuck stale file must not disable capture for good (#397 BR-2).
func TestUnreportableFileStillInstallsCapture(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "crash")
	if err := os.MkdirAll(filepath.Join(dir, "20261006T135600Z-111.crash", "blocker"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "20261006T135600Z-111.log"), []byte("panic: boom\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	capture, _, err := Install(dir, time.Now(), 4242, dead)
	if err == nil {
		t.Fatal("the failed rename was not reported")
	}
	if capture == nil {
		t.Fatal("a stuck stale file disabled this run's capture")
	}
	capture.Close()
}

// A dying owner releases its lease (a defer) before the runtime writes its
// panic. Its .log is still empty then; a relaunch in that window must not
// misread it as an abrupt end and delete it.
func TestInstallLeavesALiveOwnersFileAlone(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "crash")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	dying := filepath.Join(dir, "20261006T135600Z-77.log")
	if err := os.WriteFile(dying, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	capture, reports, err := Install(dir, time.Now(), 4242, func(pid int) bool { return pid == 77 })
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Close()
	if len(reports) != 0 {
		t.Fatalf("live owner's file reported: %+v", reports)
	}
	if _, err := os.Stat(dying); err != nil {
		t.Fatalf("live owner's file removed: %v", err)
	}
}

func TestSummary(t *testing.T) {
	for _, tc := range []struct {
		reports []Report
		want    string
	}{
		{nil, ""},
		{[]Report{{Kind: Abrupt, Path: "a"}}, "previous couch ended abruptly (no panic recorded)"},
		{[]Report{{Kind: Abrupt, Path: "a"}, {Kind: Abrupt, Path: "b"}}, "2 previous couch runs ended abruptly (no panic recorded)"},
		{[]Report{{Kind: Crashed, Path: "old"}, {Kind: Crashed, Path: "new"}}, "previous couch crashed — see new (+1 earlier)"},
		{[]Report{{Kind: Crashed, Path: "c"}, {Kind: Abrupt, Path: "a"}}, "previous couch crashed — see c; previous couch ended abruptly (no panic recorded)"},
		{[]Report{{Kind: Exited, Path: "e", Reason: "terminal stopped accepting output for 5s"}}, "previous couch exited: terminal stopped accepting output for 5s"},
		{[]Report{{Kind: Exited, Reason: "old"}, {Kind: Exited, Reason: "new"}, {Kind: Crashed, Path: "c"}}, "previous couch exited: new (+1 earlier); previous couch crashed — see c"},
	} {
		if got := Summary(tc.reports); got != tc.want {
			t.Fatalf("Summary(%+v) = %q, want %q", tc.reports, got, tc.want)
		}
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
		{Name: "20261006T135600Z-6.log", Size: 40, Head: "couch-exit: terminal stopped\ntrailing"},
	})
	want := []Finding{
		{Name: "20261006T135600Z-1.log", Kind: Crashed},
		{Name: "20261006T135600Z-2.log", Kind: Abrupt},
		{Name: "20261006T135600Z-3.crash", Kind: Reported},
		{Name: "20261006T135600Z-6.log", Kind: Exited, Reason: "terminal stopped"},
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

// Over the bound, gc still drains the first MaxEntries and reports the rest,
// instead of refusing the directory forever.
func TestSweepDrainsAnOverfullDirectory(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-(diagnosticlog.RetentionPeriod + 24*time.Hour))
	for i := 1; i <= MaxEntries+1; i++ {
		path := filepath.Join(dir, fmt.Sprintf("20251001T000000Z-%d.crash", i))
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := Sweep(dir, time.Now(), dead, true, MaxEntries+10)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Eligible || !strings.Contains(rows[0].Reason, "more than") {
		t.Fatalf("overflow not reported first: %+v", rows[0])
	}
	left, _ := os.ReadDir(dir)
	if len(left) != 1 {
		t.Fatalf("%d files left, want one past the bound", len(left))
	}
}

// One capture per process: a second Install ends the first normally.
func TestSecondInstallEndsTheFirst(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "crash")
	if _, _, err := Install(dir, time.Now(), 4242, dead); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Install(dir, time.Now(), 4243, dead); err != nil {
		t.Fatal(err)
	}
	if err := Finish(); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("captures left behind: %v", entries)
	}
}

// TestRecordedExitIsReportedOnceWithItsReason: a run that records why it exits
// and then returns normally leaves its reason for the next start, which
// reports it once, apart from a panic left by another run (pair#409).
func TestRecordedExitIsReportedOnceWithItsReason(t *testing.T) {
	dir := t.TempDir()
	dead := func(int) bool { return false }
	// Another run's panic, already in the directory.
	if err := os.WriteFile(filepath.Join(dir, "20261001T000000Z-7.log"), []byte("panic: boom\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Install(dir, time.Date(2026, 10, 7, 4, 31, 0, 0, time.UTC), 11, dead); err != nil {
		t.Fatal(err)
	}
	if err := RecordExit("terminal stopped accepting output for 5s\n(wrote 0 of 17424 bytes)"); err != nil {
		t.Fatal(err)
	}
	if err := Finish(); err != nil {
		t.Fatal(err)
	}
	_, reports, err := Install(dir, time.Date(2026, 10, 7, 4, 40, 0, 0, time.UTC), 12, dead)
	if err != nil {
		t.Fatal(err)
	}
	var exited *Report
	for i := range reports {
		if reports[i].Kind == Exited {
			exited = &reports[i]
		}
	}
	if exited == nil || exited.Reason != "terminal stopped accepting output for 5s (wrote 0 of 17424 bytes)" || !strings.HasSuffix(exited.Path, reported) {
		t.Fatalf("reports %+v", reports)
	}
	if !strings.Contains(Summary(reports), "previous couch exited: terminal stopped") {
		t.Fatal(Summary(reports))
	}
	if err := Finish(); err != nil {
		t.Fatal(err)
	}
	_, again, err := Install(dir, time.Date(2026, 10, 7, 4, 50, 0, 0, time.UTC), 13, dead)
	if err != nil {
		t.Fatal(err)
	}
	if Summary(again) != "" {
		t.Fatalf("reported twice: %+v", again)
	}
	Finish()
}

func TestRecordExitWithoutCaptureIsANoOp(t *testing.T) {
	Finish()
	if err := RecordExit("x"); err != nil {
		t.Fatal(err)
	}
}
