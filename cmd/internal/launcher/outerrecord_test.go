package launcher

import (
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func TestOuterRecordRoundTrips(t *testing.T) {
	for _, r := range []OuterRecord{{TTY: "/dev/ttys006", Couch: true}, {TTY: "/dev/ttys001"}} {
		got, err := DecodeOuterRecord(EncodeOuterRecord(r))
		if err != nil || got != r {
			t.Fatalf("%+v → %+v, %v", r, got, err)
		}
	}
}

// A record written by a launcher from before #282 has only the tty line. It was
// never Couch-aware, so it says nothing about Couch: not presented.
func TestLegacyOuterRecordIsNotPresented(t *testing.T) {
	got, err := DecodeOuterRecord("/dev/ttys006\n")
	if err != nil || got.Couch || got.TTY != "/dev/ttys006" {
		t.Fatalf("%+v, %v", got, err)
	}
}

// Anything else is refused, never guessed: a truncated or garbled record must
// not read as Couch.
func TestMalformedOuterRecordIsAnError(t *testing.T) {
	for _, raw := range []string{"", "\n", "tty\n", "/dev/ttys006\npresenter=maybe\n", "/dev/ttys006\npresenter=couch\nextra\n", "/dev/ttys006\ncouch\n"} {
		if _, err := DecodeOuterRecord(raw); err == nil {
			t.Errorf("%q decoded without error", raw)
		}
	}
}

// DecodeOuterRecord is the one parser of input this change did not produce:
// the record is truncatable, hand-editable and cross-version. Couch is true
// only for the exact two-line presenter=couch shape, and nothing panics.
func FuzzDecodeOuterRecord(f *testing.F) {
	for _, seed := range []string{
		"/dev/ttys006\n", "/dev/ttys006\npresenter=couch\n", "/dev/ttys006\npresenter=terminal\n",
		"", "\n", "tty\n", "/dev/ttys006\npresenter=maybe\n", "/dev/ttys006\npresenter=couch\nextra\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		got, err := DecodeOuterRecord(raw)
		lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
		exact := len(lines) == 2 && strings.HasPrefix(lines[0], "/dev/") && lines[1] == presenterCouch
		if err == nil && got.Couch != exact {
			t.Fatalf("%q decoded Couch=%v", raw, got.Couch)
		}
	})
}

// Couch launched this client for this thread: its tag must match, so a stray
// COUCH_THREAD_TAG inherited by another launch is not mistaken for Couch.
func TestPresentedByCouchRequiresTheThreadsTag(t *testing.T) {
	if !PresentedByCouch(Env{CouchThreadScope: "s", CouchThreadTag: "couch-a"}, "couch-a") {
		t.Error("Couch's own client not recognised")
	}
	for _, env := range []Env{{}, {CouchThreadScope: "s"}, {CouchThreadTag: "couch-b"}} {
		if PresentedByCouch(env, "couch-a") {
			t.Errorf("%+v recognised as Couch for couch-a", env)
		}
	}
}

// The reader and the writer share one path resolution; a missing record is
// "not presented" (a non-tty attach removes it), not an error.
func TestReadOuterPresenter(t *testing.T) {
	dir := t.TempDir()
	if couch, err := ReadOuterPresenter(dir, "work"); err != nil || couch {
		t.Fatalf("missing record: %v %v", couch, err)
	}
	paths, err := artifactpath.ResolveScoped(dir, "work")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.OuterTTY(), []byte(EncodeOuterRecord(OuterRecord{TTY: "/dev/ttys006", Couch: true})), 0o600); err != nil {
		t.Fatal(err)
	}
	if couch, err := ReadOuterPresenter(dir, "work"); err != nil || !couch {
		t.Fatalf("couch record: %v %v", couch, err)
	}
	if err := os.WriteFile(paths.OuterTTY(), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadOuterPresenter(dir, "work"); err == nil {
		t.Fatal("malformed record read without error")
	}
}

// An attach whose stdin is not a tty removes the record rather than keeping an
// earlier attach's presenter line, which would lie about who presents now.
func TestNonTTYAttachRemovesThePresenterRecord(t *testing.T) {
	dir := t.TempDir()
	paths, err := artifactpath.ResolveScoped(dir, "work")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.OuterTTY(), []byte(EncodeOuterRecord(OuterRecord{TTY: "/dev/ttys006", Couch: true})), 0o600); err != nil {
		t.Fatal(err)
	}
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devnull.Close()
	saved := os.Stdin
	os.Stdin = devnull
	defer func() { os.Stdin = saved }()
	NewScopedOSRuntime(dir, dir, "/pair").RecordOuterTTY("work", true)
	if _, err := os.Stat(paths.OuterTTY()); !os.IsNotExist(err) {
		t.Fatalf("record survived a non-tty attach: %v", err)
	}
}
